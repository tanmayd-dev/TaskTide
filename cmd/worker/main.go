package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
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
	conn, err := grpc.Dial(addr, grpc.DialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	regResp, err := client.RegisterWorker(ctx, &pb.RegisterRequest{
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

	// Start Heartbeat loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{WorkerId: workerID})
				if err != nil {
					log.Printf("Heartbeat failed: %v\n", err)
				}
			}
		}
	}()

	// Semaphore for concurrency control
	sem := make(chan struct{}, concurrencyLimit)

	// Catch OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v. Initiating graceful shutdown...\n", sig)
		cancel() // Stop polling and heartbeats
	}()

	// Polling loop
	log.Println("Worker starting polling loop...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker polling loop stopped. Waiting for active tasks to complete...")
			// Wait for all active task slots to be released
			for i := 0; i < concurrencyLimit; i++ {
				sem <- struct{}{}
			}
			log.Println("All active tasks completed. Exiting gracefully.")
			return
		default:
			// Calculate free slots
			freeSlots := concurrencyLimit - len(sem)
			if freeSlots <= 0 {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			pollResp, err := client.PollJobs(ctx, &pb.PollRequest{
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
				sem <- struct{}{}
				go func(j *pb.Job) {
					defer func() { <-sem }()
					log.Printf("Worker executing job %s of type workflow-task\n", j.Id)
					workerService.ExecuteJob(context.Background(), client, workerID, j)
				}(job)
			}
		}
	}
}
