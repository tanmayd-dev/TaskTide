package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	fmt.Println("  list-workers     List active and dead workers")
	fmt.Println("  pause-queue      Pause a queue")
	fmt.Println("  resume-queue     Resume a queue")
	os.Exit(1)
}

func submitJobCmd() {
	cmd := flag.NewFlagSet("submit-job", flag.ExitOnError)
	queueName := cmd.String("queue", "default", "Queue name")
	payload := cmd.String("payload", "{}", "JSON payload")
	retries := cmd.Int("retries", 3, "Max retries")
	delay := cmd.Int("delay", 0, "Delay in seconds")
	priority := cmd.Int("priority", 0, "Job priority")
	idempotencyKey := cmd.String("idempotency-key", "", "Idempotency key")
	addr := cmd.String("addr", "localhost:50051", "gRPC address of the coordinator")

	_ = cmd.Parse(os.Args[2:])

	// Validate payload as JSON
	var js json.RawMessage
	if json.Unmarshal([]byte(*payload), &js) != nil {
		log.Fatalf("Error: payload is not valid JSON")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to coordinator: %v", err)
	}
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

func getDBConn(ctx context.Context) *pgx.Conn {
	dbString := os.Getenv("POSTGRES_DBSTRING")
	if dbString == "" {
		dbString = "postgres://postgres:postgres@localhost:5432/TaskTide?sslmode=disable"
	}
	conn, err := pgx.Connect(ctx, dbString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return conn
}

func inspectQueueCmd() {
	cmd := flag.NewFlagSet("inspect-queue", flag.ExitOnError)
	queueName := cmd.String("queue", "default", "Queue name")
	_ = cmd.Parse(os.Args[2:])

	ctx := context.Background()
	conn := getDBConn(ctx)
	defer conn.Close(ctx)

	// Ensure queue exists in database first
	var qExists bool
	_ = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM queues WHERE name=$1)", *queueName).Scan(&qExists)
	if !qExists {
		fmt.Printf("Queue '%s' does not exist.\n", *queueName)
		return
	}

	var paused bool
	_ = conn.QueryRow(ctx, "SELECT paused FROM queues WHERE name=$1", *queueName).Scan(&paused)

	var pending, running, completed, failed, deadLetter int
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE queue_name=$1 AND status='pending'", *queueName).Scan(&pending)
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE queue_name=$1 AND status='running'", *queueName).Scan(&running)
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE queue_name=$1 AND status='completed'", *queueName).Scan(&completed)
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE queue_name=$1 AND status='failed'", *queueName).Scan(&failed)
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE queue_name=$1 AND status='dead_letter'", *queueName).Scan(&deadLetter)

	fmt.Printf("Queue: %s (Paused: %v)\n", *queueName, paused)
	fmt.Printf("  Pending:     %d\n", pending)
	fmt.Printf("  Running:     %d\n", running)
	fmt.Printf("  Completed:   %d\n", completed)
	fmt.Printf("  Failed:      %d\n", failed)
	fmt.Printf("  Dead-Letter: %d\n", deadLetter)
}

func retryJobCmd() {
	cmd := flag.NewFlagSet("retry-failed-job", flag.ExitOnError)
	jobId := cmd.String("job", "", "Job ID (UUID)")
	_ = cmd.Parse(os.Args[2:])

	if *jobId == "" {
		log.Fatalf("Error: --job argument is required")
	}

	ctx := context.Background()
	conn := getDBConn(ctx)
	defer conn.Close(ctx)

	var status string
	err := conn.QueryRow(ctx, "SELECT status FROM jobs WHERE id=$1", *jobId).Scan(&status)
	if err != nil {
		log.Fatalf("Job '%s' not found: %v", *jobId, err)
	}

	if status != "failed" && status != "dead_letter" {
		log.Fatalf("Job '%s' is in status '%s'. Can only retry failed or dead_letter jobs.", *jobId, status)
	}

	// Move job back to pending
	_, err = conn.Exec(ctx, "UPDATE jobs SET status='pending', retry_count=0, scheduled_at=NOW(), updated_at=NOW() WHERE id=$1", *jobId)
	if err != nil {
		log.Fatalf("Failed to retry job: %v", err)
	}

	// Delete from worker leases and dead letter list if exists
	_, _ = conn.Exec(ctx, "DELETE FROM worker_leases WHERE job_id=$1", *jobId)
	_, _ = conn.Exec(ctx, "DELETE FROM dead_letter_jobs WHERE job_id=$1", *jobId)

	fmt.Printf("Job %s successfully rescheduled to pending status.\n", *jobId)
}

func listWorkersCmd() {
	ctx := context.Background()
	conn := getDBConn(ctx)
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "SELECT id, worker_host, worker_port, concurrency, capabilities, last_heartbeat FROM worker ORDER BY last_heartbeat DESC")
	if err != nil {
		log.Fatalf("Failed to query workers: %v", err)
	}
	defer rows.Close()

	fmt.Printf("%-38s %-20s %-12s %-50s %-20s\n", "WORKER ID", "HOST:PORT", "CONCURRENCY", "CAPABILITIES", "LAST SEEN")
	fmt.Println(fmt.Sprintf("%s", time.Now().Format("2006-01-02 15:04:05")))

	for rows.Next() {
		var id pgtype.UUID
		var host, port string
		var concurrency int32
		var capabilities []string
		var lastHeartbeat pgtype.Timestamp

		err = rows.Scan(&id, &host, &port, &concurrency, &capabilities, &lastHeartbeat)
		if err != nil {
			log.Fatalf("Failed to scan worker: %v", err)
		}

		addr := fmt.Sprintf("%s:%s", host, port)
		statusStr := "ACTIVE"
		if time.Since(lastHeartbeat.Time) > 15*time.Second {
			statusStr = "DEAD"
		}

		fmt.Printf("%-38s %-20s %-12d %-50v %-20s (%s)\n",
			id.String(),
			addr,
			concurrency,
			capabilities,
			lastHeartbeat.Time.Format("15:04:05"),
			statusStr,
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

	ctx := context.Background()
	conn := getDBConn(ctx)
	defer conn.Close(ctx)

	// Ensure queue exists
	var qExists bool
	_ = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM queues WHERE name=$1)", *queueName).Scan(&qExists)
	if !qExists {
		// Create queue if it doesn't exist
		_, err := conn.Exec(ctx, "INSERT INTO queues (name, paused) VALUES ($1, $2)", *queueName, paused)
		if err != nil {
			log.Fatalf("Failed to create queue: %v", err)
		}
	} else {
		_, err := conn.Exec(ctx, "UPDATE queues SET paused=$1 WHERE name=$2", paused, *queueName)
		if err != nil {
			log.Fatalf("Failed to update queue: %v", err)
		}
	}

	action := "paused"
	if !paused {
		action = "resumed"
	}
	fmt.Printf("Queue '%s' successfully %s.\n", *queueName, action)
}
