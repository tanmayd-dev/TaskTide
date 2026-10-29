package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsageAndExit()
	}

	command := os.Args[1]

	switch command {
	case "submit-job":
		submitJobCmd()
	case "inspect-queue":
		inspectQueueCmd()
	case "retry-failed-job":
		retryJobCmd()
	case "list-workers":
		listWorkersCmd()
	case "pause-queue":
		pauseQueueCmd(true)
	case "resume-queue":
		pauseQueueCmd(false)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsageAndExit()
	}
}

func printUsageAndExit() {
	fmt.Println("Usage: tasktide-cli <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  submit-job       Submit a new job to a queue")
	fmt.Println("  inspect-queue    Show metrics for a queue")
	fmt.Println("  retry-failed-job Retry a failed or dead-letter job")
	fmt.Println("  list-workers     List registered workers")
	fmt.Println("  pause-queue      Pause a queue")
	fmt.Println("  resume-queue     Resume a queue")
	os.Exit(1)
}

// dialCoordinator connects to the TaskTide gRPC coordinator using the same
// connection env knobs as the worker (QUEUE_HOST, GRPC_PORT).
func dialCoordinator() *grpc.ClientConn {
	queueHost := os.Getenv("QUEUE_HOST")
	if queueHost == "" {
		queueHost = "localhost"
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	addr := fmt.Sprintf("%s:%s", queueHost, grpcPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to coordinator at %s: %v", addr, err)
	}
	return conn
}

func submitJobCmd() {
	cmd := flag.NewFlagSet("submit-job", flag.ExitOnError)
	queueName := cmd.String("queue", "default", "Queue name")
	payload := cmd.String("payload", "{}", "JSON payload")
	retries := cmd.Int("retries", 3, "Max retries")
	delay := cmd.Int("delay", 0, "Delay in seconds")
	priority := cmd.Int("priority", 0, "Job priority")
	idempotencyKey := cmd.String("idempotency-key", "", "Idempotency key")

	_ = cmd.Parse(os.Args[2:])

	// Validate payload as JSON
	var js json.RawMessage
	if json.Unmarshal([]byte(*payload), &js) != nil {
		log.Fatalf("Error: payload is not valid JSON")
	}

	conn := dialCoordinator()
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SubmitJob(ctx, &pb.SubmitRequest{
		QueueName:      *queueName,
		PayloadJson:    *payload,
		MaxRetries:     int32(*retries),
		DelaySeconds:   int32(*delay),
		Priority:       int32(*priority),
		IdempotencyKey: *idempotencyKey,
	})
	if err != nil {
		log.Fatalf("Failed to submit job: %v", err)
	}

	fmt.Printf("Job submitted successfully!\nID: %s\nStatus: %s\n", resp.JobId, resp.Status)
}

func inspectQueueCmd() {
	cmd := flag.NewFlagSet("inspect-queue", flag.ExitOnError)
	queueName := cmd.String("queue", "default", "Queue name")
	_ = cmd.Parse(os.Args[2:])

	conn := dialCoordinator()
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.InspectQueue(ctx, &pb.InspectQueueRequest{QueueName: *queueName})
	if err != nil {
		log.Fatalf("Failed to inspect queue '%s': %v", *queueName, err)
	}

	fmt.Printf("Queue: %s\n", resp.QueueName)
	fmt.Printf("  Pending:     %d\n", resp.Pending)
	fmt.Printf("  Running:     %d\n", resp.Running)
	fmt.Printf("  Completed:   %d\n", resp.Completed)
	fmt.Printf("  Failed:      %d\n", resp.Failed)
	fmt.Printf("  Dead-Letter: %d\n", resp.DeadLetter)
}

func retryJobCmd() {
	cmd := flag.NewFlagSet("retry-failed-job", flag.ExitOnError)
	jobId := cmd.String("job", "", "Job ID (UUID)")
	_ = cmd.Parse(os.Args[2:])

	if *jobId == "" {
		log.Fatalf("Error: --job argument is required")
	}

	conn := dialCoordinator()
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RetryJob(ctx, &pb.RetryJobRequest{JobId: *jobId})
	if err != nil {
		log.Fatalf("Failed to retry job '%s': %v", *jobId, err)
	}

	fmt.Printf("Job %s successfully rescheduled to status '%s'.\n", resp.JobId, resp.Status)
}

func listWorkersCmd() {
	conn := dialCoordinator()
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ListWorkers(ctx, &pb.ListWorkersRequest{})
	if err != nil {
		log.Fatalf("Failed to list workers: %v", err)
	}

	fmt.Printf("%-38s %-25s %-10s %-20s\n", "WORKER ID", "HOST", "STATUS", "LAST SEEN")
	fmt.Println(fmt.Sprintf("%s", time.Now().Format("2006-01-02 15:04:05")))

	for _, w := range resp.Workers {
		statusStr := "ACTIVE"
		if !w.Active {
			statusStr = "DEAD"
		}
		fmt.Printf("%-38s %-25s %-10s %-20s\n",
			w.WorkerId,
			w.Host,
			statusStr,
			time.Unix(w.LastHeartbeat, 0).Format("15:04:05"),
		)
	}
}

func pauseQueueCmd(paused bool) {
	cmdName := "pause-queue"
	if !paused {
		cmdName = "resume-queue"
	}
	cmd := flag.NewFlagSet(cmdName, flag.ExitOnError)
	queueName := cmd.String("queue", "default", "Queue name")
	_ = cmd.Parse(os.Args[2:])

	conn := dialCoordinator()
	defer conn.Close()

	client := pb.NewTaskTideServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.PauseQueue(ctx, &pb.PauseQueueRequest{QueueName: *queueName, Paused: paused})
	if err != nil {
		log.Fatalf("Failed to set paused=%t on queue '%s': %v", paused, *queueName, err)
	}

	action := "paused"
	if !paused {
		action = "resumed"
	}
	fmt.Printf("Queue '%s' successfully %s (paused: %t).\n", resp.QueueName, action, resp.Paused)
}
