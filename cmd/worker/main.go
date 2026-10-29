package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/observability"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// leaseRenewalInterval is how often the worker extends a job's lease while
// that job is executing.
const leaseRenewalInterval = 10 * time.Second

func main() {
	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	// Initialize OpenTelemetry tracer
	tp, err := observability.InitTracer("worker-service")
	if err == nil {
		defer tp.Shutdown(context.Background())
	}

	// Validate NOOP_TIME at startup so a bad value fails fast instead of
	// making every task fail at runtime.
	if noopTime := os.Getenv("NOOP_TIME"); noopTime != "" {
		seconds, err := strconv.Atoi(noopTime)
		if err != nil {
			log.Fatalf("Invalid NOOP_TIME %q: %v\n", noopTime, err)
		}
		if seconds < 0 {
			log.Fatalf("Invalid NOOP_TIME %q: must be non-negative\n", noopTime)
		}
	}

	queueHost := os.Getenv("QUEUE_HOST")
	if queueHost == "" {
		queueHost = "localhost"
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	// Connect to gRPC coordinator
	addr := fmt.Sprintf("%s:%s", queueHost, grpcPort)
	log.Printf("Worker connecting to coordinator at %s...\n", addr)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Worker connection failed: %v", err)
	}
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)

	// Register worker
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	concurrencyLimit := 5
	capabilities := []string{"NOOP", "OUTPUT", "doc.read", "csv.read"}

	// pollCtx controls the polling/acquisition loop; workerCtx covers the
	// worker lifetime (connection, registration, heartbeat and lease
	// renewals). On SIGINT/SIGTERM only pollCtx is cancelled so heartbeats
	// keep running while active tasks drain; workerCtx is cancelled after
	// the drain completes (or times out).
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	pollCtx, pollCancel := context.WithCancel(context.Background())
	defer pollCancel()

	regResp, err := client.RegisterWorker(workerCtx, &pb.RegisterRequest{
		Hostname:     hostname,
		Port:         "0", // no longer listening on port in pull model
		Concurrency:  int32(concurrencyLimit),
		Capabilities: capabilities,
	})
	if err != nil {
		log.Fatalf("Worker registration failed: %v", err)
	}

	workerID := regResp.WorkerId
	log.Printf("Worker registered successfully. Assigned ID: %s\n", workerID)

	// Start Heartbeat loop (worker lifetime)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				_, err := client.Heartbeat(workerCtx, &pb.HeartbeatRequest{WorkerId: workerID})
				if err != nil {
					log.Printf("Heartbeat failed: %v\n", err)
				}
			}
		}
	}()

	// In-flight job tracking: WaitGroup for the drain wait, atomic count for
	// free-slot calculation.
	var inFlight sync.WaitGroup
	var activeCount int32

	// Catch OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v. Initiating graceful shutdown...\n", sig)
		pollCancel() // Stop acquiring new jobs; heartbeats keep running.
	}()

	// Polling loop
	log.Println("Worker starting polling loop...")
	for {
		select {
		case <-pollCtx.Done():
			log.Println("Worker polling loop stopped. Waiting for active tasks to complete (up to 30s)...")
			drained := make(chan struct{})
			go func() {
				inFlight.Wait()
				close(drained)
			}()
			select {
			case <-drained:
				log.Println("All active tasks completed. Exiting gracefully.")
			case <-time.After(30 * time.Second):
				log.Printf("Drain timed out after 30s; abandoning %d task(s) and exiting.\n", atomic.LoadInt32(&activeCount))
			}
			workerCancel() // Stop heartbeats/lease renewals once drained.
			return
		default:
			// Calculate free slots
			freeSlots := concurrencyLimit - int(atomic.LoadInt32(&activeCount))
			if freeSlots <= 0 {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			pollResp, err := client.PollJobs(pollCtx, &pb.PollRequest{
				WorkerId:     workerID,
				Capabilities: capabilities,
				FreeSlots:    int32(freeSlots),
			})
			if err != nil {
				log.Printf("Polling failed: %v. Retrying in 2 seconds...\n", err)
				time.Sleep(2 * time.Second)
				continue
			}

			if len(pollResp.Jobs) == 0 {
				// No jobs available, back off
				time.Sleep(1 * time.Second)
				continue
			}

			for _, job := range pollResp.Jobs {
				if atomic.LoadInt32(&activeCount) >= int32(concurrencyLimit) {
					break
				}
				inFlight.Add(1)
				atomic.AddInt32(&activeCount, 1)
				go func(j *pb.Job) {
					defer atomic.AddInt32(&activeCount, -1)
					defer inFlight.Done()
					runJob(workerCtx, client, workerID, j)
				}(job)
			}
		}
	}
}

// runJob executes one job while a background goroutine periodically extends
// its lease so long-running tasks are not reclaimed mid-execution. Renewal
// starts when execution starts and is stopped (context cancel) when it
// finishes, success or failure.
func runJob(ctx context.Context, client pb.TaskTideServiceClient, workerID string, job *pb.Job) {
	log.Printf("Worker executing job %s of type workflow-task\n", job.Id)

	renewCtx, cancelRenew := context.WithCancel(ctx)
	defer cancelRenew()
	go renewLease(renewCtx, client, workerID, job.Id)

	workerService.ExecuteJob(ctx, client, workerID, job)
}

// renewLease extends the job's lease every leaseRenewalInterval until its
// context is cancelled. The first failure per job is logged; later ones are
// silenced to keep noise low.
func renewLease(ctx context.Context, client pb.TaskTideServiceClient, workerID, jobID string) {
	ticker := time.NewTicker(leaseRenewalInterval)
	defer ticker.Stop()
	loggedFirstFailure := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := client.ExtendLease(ctx, &pb.ExtendLeaseRequest{
				JobId:    jobID,
				WorkerId: workerID,
				// 0 lets the server apply its default lease duration.
				ExtensionSeconds: 0,
			})
			if err != nil && !loggedFirstFailure {
				loggedFirstFailure = true
				log.Printf("Lease renewal failed for job %s: %v (further renewal failures silenced for this job)\n", jobID, err)
			}
		}
	}
}
