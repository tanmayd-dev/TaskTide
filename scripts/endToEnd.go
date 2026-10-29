package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Scenarios intentionally NOT covered by this harness (stack-level
// properties, not harness gaps):
//   - lease-extension-on-a->30s-task: requires NOOP_TIME > 30s (the lease
//     window) in the worker env; cannot be staged from the harness itself
//     (the harness must not edit env).
//   - worker-SIGKILL recovery: requires a multi-replica worker topology.
//   - coordinator-restart scheduled_at preservation: disruptive to the
//     stack; indirectly covered by case 4's transient retry flow.

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	host := getEnv("API_HOST", "127.0.0.1")
	port := getEnv("API_PORT", "8080")
	baseURL := fmt.Sprintf("http://%s:%s", host, port)

	// Open DB connection for persistence checks
	dbString := os.Getenv("POSTGRES_DBSTRING")
	if dbString == "" {
		log.Fatalf("POSTGRES_DBSTRING is not set in environment")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbString)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// ----------------------------------------------------
	// Test Scenario 1: Chained Workflow (A -> B -> C -> OUTPUT)
	// ----------------------------------------------------
	log.Println("=== Starting Test Case 1: Chained Tasks ===")
	workflowID1 := createWorkflow(baseURL, "ChainedWorkflow")
	log.Println("Created workflow:", workflowID1)

	file1 := uploadFile(baseURL, "tmp/sample1.txt")
	attachFileToWorkflow(baseURL, workflowID1, file1)

	chainTaskA := createTask(baseURL, workflowID1, models.TaskNOOP, "Chain Task A", file1.ID, "")
	chainTaskB := createTask(baseURL, workflowID1, models.TaskNOOP, "Chain Task B", "", chainTaskA)
	chainTaskC := createTask(baseURL, workflowID1, models.TaskNOOP, "Chain Task C", "", chainTaskB)
	outputTask := createTask(baseURL, workflowID1, models.TaskOUTPUT, "Chain Task Output", "", chainTaskC)

	log.Printf("Created chained tasks: %s -> %s -> %s -> %s\n", chainTaskA, chainTaskB, chainTaskC, outputTask)

	executionID1 := startWorkflow(baseURL, workflowID1)
	log.Printf("Workflow execution started: %s\n", executionID1)

	pollWorkflowStatus(baseURL, executionID1)
	log.Println("Workflow execution completed successfully.")

	// Verify Output Content of Scenario 1 (chained sequential appends)
	tes1 := getWorkflowTaskExecutions(baseURL, executionID1)
	var outputTaskExec models.TaskExecution
	for _, te := range tes1 {
		// Output task base is outputTask
		if te.TaskBase == outputTask {
			outputTaskExec = te
			break
		}
	}
	if outputTaskExec.ID == "" {
		log.Fatalf("Could not find task execution for output task %s", outputTask)
	}

	finalContent := downloadFile(baseURL, outputTaskExec.TaskOutput)
	log.Printf("Downloaded final output file content:\n%s\n", finalContent)

	expectedSuffixes := []string{
		"Hello TaskTide!",
		"NOOP appended text from noop",
		"NOOP appended text from output",
	}
	for _, suffix := range expectedSuffixes {
		if !strings.Contains(finalContent, suffix) {
			log.Fatalf("Output validation failed: final content does not contain expected suffix '%s'", suffix)
		}
	}
	log.Println("Output validation passed: chained execution processed input sequence correctly.")

	// Verify DB Persistence
	verifyDatabasePersistence(ctx, conn, "ChainedWorkflow", executionID1, 4)
	log.Println("=== Test Case 1 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 2: Independent Tasks (A, B, C)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 2: Independent Tasks ===")
	workflowID2 := createWorkflow(baseURL, "IndependentWorkflow")
	log.Println("Created workflow:", workflowID2)

	file2 := uploadFile(baseURL, "tmp/sample1.txt")
	attachFileToWorkflow(baseURL, workflowID2, file2)

	taskA := createTask(baseURL, workflowID2, models.TaskNOOP, "Independent Task A", file2.ID, "")
	taskB := createTask(baseURL, workflowID2, models.TaskNOOP, "Independent Task B", file2.ID, "")
	taskC := createTask(baseURL, workflowID2, models.TaskNOOP, "Independent Task C", file2.ID, "")

	log.Printf("Created independent tasks: %s, %s, %s\n", taskA, taskB, taskC)

	executionID2 := startWorkflow(baseURL, workflowID2)
	log.Printf("Workflow execution started: %s\n", executionID2)

	pollWorkflowStatus(baseURL, executionID2)
	log.Println("Workflow execution completed successfully.")

	// Verify DB Persistence
	verifyDatabasePersistence(ctx, conn, "IndependentWorkflow", executionID2, 3)
	log.Println("=== Test Case 2 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 3: Mixed Workflow (A -> B, C, D -> E)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 3: Mixed Tasks ===")
	workflowID3 := createWorkflow(baseURL, "MixedWorkflow")
	log.Println("Created workflow:", workflowID3)

	file3 := uploadFile(baseURL, "tmp/sample1.txt")
	attachFileToWorkflow(baseURL, workflowID3, file3)

	mixedA := createTask(baseURL, workflowID3, models.TaskNOOP, "Mixed Task A", file3.ID, "")
	mixedB := createTask(baseURL, workflowID3, models.TaskNOOP, "Mixed Task B", "", mixedA)
	mixedC := createTask(baseURL, workflowID3, models.TaskNOOP, "Mixed Task C", file3.ID, "")
	mixedD := createTask(baseURL, workflowID3, models.TaskNOOP, "Mixed Task D", file3.ID, "")
	mixedE := createTask(baseURL, workflowID3, models.TaskOUTPUT, "Mixed Task E", "", mixedD)

	log.Printf("Created mixed tasks: (%s -> %s), %s, (%s -> %s)\n", mixedA, mixedB, mixedC, mixedD, mixedE)

	executionID3 := startWorkflow(baseURL, workflowID3)
	log.Printf("Workflow execution started: %s\n", executionID3)

	pollWorkflowStatus(baseURL, executionID3)
	log.Println("Workflow execution completed successfully.")

	// Verify DB Persistence
	verifyDatabasePersistence(ctx, conn, "MixedWorkflow", executionID3, 5)
	log.Println("=== Test Case 3 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 4: Transient Retry (A fails once, succeeds on retry)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 4: Transient Retry ===")
	workflowID4 := createWorkflow(baseURL, "TransientRetryWorkflow")
	log.Println("Created workflow:", workflowID4)

	// Create a file with content "FAIL_ONCE"
	failOnceFile := "tmp/fail_once.txt"
	err = os.WriteFile(failOnceFile, []byte("FAIL_ONCE"), 0644)
	if err != nil {
		log.Fatalf("failed to write fail_once file: %v", err)
	}
	defer os.Remove(failOnceFile)

	file4 := uploadFile(baseURL, failOnceFile)
	attachFileToWorkflow(baseURL, workflowID4, file4)

	transientTask := createTask(baseURL, workflowID4, models.TaskNOOP, "Transient Task", file4.ID, "")

	log.Printf("Created transient task: %s (will fail once)\n", transientTask)

	executionID4 := startWorkflow(baseURL, workflowID4)
	log.Printf("Workflow execution started: %s\n", executionID4)

	pollWorkflowStatus(baseURL, executionID4)
	log.Println("Workflow execution completed successfully.")

	// Verify DB Persistence and 2 attempts
	verifyDatabasePersistence(ctx, conn, "TransientRetryWorkflow", executionID4, 1)
	
	// Query task attempts count for transientTask
	var teID4 string
	err = conn.QueryRow(ctx, "SELECT id FROM task_execution WHERE workflow_id = $1 AND task_base = $2", executionID4, transientTask).Scan(&teID4)
	if err != nil {
		log.Fatalf("Database validation failed: could not find task execution for %s: %v", transientTask, err)
	}
	var attemptCount4 int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM task_attempt WHERE task_execution_base = $1", teID4).Scan(&attemptCount4)
	if err != nil {
		log.Fatalf("Database validation failed: could not count task attempts for transient task: %v", err)
	}
	if attemptCount4 != 2 {
		log.Fatalf("Database validation failed: expected exactly 2 attempts, got %d", attemptCount4)
	}
	log.Printf("[DB OK] found exactly %d task attempts for transient task (first failed, second succeeded)\n", attemptCount4)
	log.Println("=== Test Case 4 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 5: Permanent Retry Exhaustion (A fails 3 times, marked completed (failed))
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 5: Permanent Failure Retry Exhaustion ===")
	workflowID5 := createWorkflow(baseURL, "PermanentFailureWorkflow")
	log.Println("Created workflow:", workflowID5)

	// Create a file with content "FAIL"
	failFile := "tmp/fail.txt"
	err = os.WriteFile(failFile, []byte("FAIL"), 0644)
	if err != nil {
		log.Fatalf("failed to write fail file: %v", err)
	}
	defer os.Remove(failFile)

	file5 := uploadFile(baseURL, failFile)
	attachFileToWorkflow(baseURL, workflowID5, file5)

	permanentTask := createTask(baseURL, workflowID5, models.TaskNOOP, "Permanent Failure Task", file5.ID, "")

	log.Printf("Created permanent failure task: %s (will fail 3 times)\n", permanentTask)

	executionID5 := startWorkflow(baseURL, workflowID5)
	log.Printf("Workflow execution started: %s\n", executionID5)

	// Phase 3 semantics: on retry exhaustion the job is dead-lettered and the
	// workflow task execution is marked FAILED (task_failed = TRUE) instead of
	// being completed. Poll the database until that terminal state is visible.
	var teID5 string
	err = conn.QueryRow(ctx, "SELECT id FROM task_execution WHERE workflow_id = $1 AND task_base = $2", executionID5, permanentTask).Scan(&teID5)
	if err != nil {
		log.Fatalf("Database validation failed: could not find task execution for %s: %v", permanentTask, err)
	}

	deadline5 := time.Now().Add(60 * time.Second)
	for {
		var teFailed bool
		err = conn.QueryRow(ctx, "SELECT task_failed FROM task_execution WHERE id = $1", teID5).Scan(&teFailed)
		if err != nil {
			log.Fatalf("Database validation failed: could not read task_failed for task execution: %v", err)
		}
		var attemptCount5 int
		err = conn.QueryRow(ctx, "SELECT count(*) FROM task_attempt WHERE task_execution_base = $1", teID5).Scan(&attemptCount5)
		if err != nil {
			log.Fatalf("Database validation failed: could not count task attempts for permanent task: %v", err)
		}
		var jobDeadLettered bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE task_execution_id = $1 AND status = 'dead_letter')", teID5).Scan(&jobDeadLettered)
		if err != nil {
			log.Fatalf("Database validation failed: could not check job status for permanent task: %v", err)
		}

		if teFailed && attemptCount5 >= 3 && jobDeadLettered {
			log.Printf("[DB OK] permanent task terminal state reached: %d attempts, task_failed=TRUE, job dead_lettered\n", attemptCount5)
			break
		}
		if time.Now().After(deadline5) {
			log.Fatalf("Database validation failed: permanent task never reached terminal state within 60s (attempts=%d teFailed=%t deadLetter=%t)", attemptCount5, teFailed, jobDeadLettered)
		}
		time.Sleep(2 * time.Second)
	}
	log.Println("Workflow execution terminal state verified (task marked failed after retry exhaustion).")

	// Verify DB Persistence
	verifyDatabasePersistence(ctx, conn, "PermanentFailureWorkflow", executionID5, 1)
	log.Println("=== Test Case 5 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 6: Concurrent Idempotent Submission (gRPC)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 6: Concurrent Idempotent Submission ===")
	gconn, gclient := dialQueueGRPC()
	defer func() { _ = gconn.Close() }()

	// Unique-per-run key so a previously run harness's leftover job with this
	// key cannot satisfy this race (submit must be exercised fresh).
	idemKey6 := fmt.Sprintf("e2e-idem-race-%d", time.Now().UnixNano())
	type submitResult struct {
		resp *pb.SubmitResponse
		err  error
	}
	const racers6 = 20
	submitResults := make([]submitResult, racers6)
	var wg6 sync.WaitGroup
	for i := 0; i < racers6; i++ {
		wg6.Add(1)
		go func(idx int) {
			defer wg6.Done()
			subCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := gclient.SubmitJob(subCtx, &pb.SubmitRequest{
				QueueName:      "default",
				PayloadJson:    "{}",
				MaxRetries:     3,
				IdempotencyKey: idemKey6,
			})
			submitResults[idx] = submitResult{resp: resp, err: err}
		}(i)
	}
	wg6.Wait()

	expected6 := submitResults[0]
	if expected6.err != nil || expected6.resp.JobId == "" {
		log.Fatalf("First idempotent submission failed: %v (resp=%v)", expected6.err, expected6.resp)
	}
	for i, r := range submitResults[1:] {
		if r.err != nil {
			log.Fatalf("Concurrent idempotent submission %d failed: %v", i+1, r.err)
		}
		if r.resp.JobId != expected6.resp.JobId || r.resp.Status != expected6.resp.Status {
			log.Fatalf("Concurrent idempotent submission %d returned JobId=%s/Status=%s, want JobId=%s/Status=%s",
				i+1, r.resp.JobId, r.resp.Status, expected6.resp.JobId, expected6.resp.Status)
		}
	}
	log.Printf("[gRPC OK] %d concurrent submissions with key %q all resolved to job %s (status %s)\n",
		racers6, idemKey6, expected6.resp.JobId, expected6.resp.Status)

	// Let the coordinator settle before the persistence check.
	time.Sleep(2 * time.Second)
	var idemRowCount int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE idempotency_key = $1", idemKey6).Scan(&idemRowCount)
	if err != nil {
		log.Fatalf("Database validation failed: could not count jobs for idempotency key %s: %v", idemKey6, err)
	}
	if idemRowCount != 1 {
		log.Fatalf("Database validation failed: expected exactly 1 job row for idempotency key %s, got %d", idemKey6, idemRowCount)
	}
	var idemJobID string
	err = conn.QueryRow(ctx, "SELECT id FROM jobs WHERE idempotency_key = $1", idemKey6).Scan(&idemJobID)
	if err != nil {
		log.Fatalf("Database validation failed: could not read job id for idempotency key %s: %v", idemKey6, err)
	}
	if idemJobID != expected6.resp.JobId {
		log.Fatalf("Database validation failed: persisted job %s does not match gRPC response id %s", idemJobID, expected6.resp.JobId)
	}
	log.Printf("[DB OK] exactly %d persisted job row for idempotency key %s\n", idemRowCount, idemKey6)
	log.Println("=== Test Case 6 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 7: DAG Poison Prevention (A dead-letters; B depends on A and must never run)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 7: DAG Poison Prevention ===")
	workflowID7 := createWorkflow(baseURL, "PoisonDAGWorkflow")
	log.Println("Created workflow:", workflowID7)

	poisonFile := writeTmpFile("poison_fail.txt", "FAIL")
	defer os.Remove(poisonFile)
	file7 := uploadFile(baseURL, poisonFile)

	// The workflow API's TaskDefinition has no per-task retry budget field,
	// so A uses the default retry budget before dead-lettering.
	taskA7 := createTask(baseURL, workflowID7, models.TaskNOOP, "Poison Task A", file7.ID, "")
	taskB7 := createTask(baseURL, workflowID7, models.TaskNOOP, "Poison Task B", "", taskA7)
	log.Printf("Created poison DAG: %s -> %s\n", taskA7, taskB7)

	executionID7 := startWorkflow(baseURL, workflowID7)
	log.Printf("Workflow execution started: %s\n", executionID7)

	deadline7 := time.Now().Add(60 * time.Second)
	poisonPrevented := false
	for time.Now().Before(deadline7) {
		var teAID string
		var teAFailed bool
		errA := conn.QueryRow(ctx, "SELECT id, task_failed FROM task_execution WHERE workflow_id = $1 AND task_base = $2", executionID7, taskA7).Scan(&teAID, &teAFailed)
		if errA != nil {
			// Workflow task executions are created at workflow start; poll
			// until they appear.
			time.Sleep(2 * time.Second)
			continue
		}

		var attempts7A int
		if err := conn.QueryRow(ctx, "SELECT count(*) FROM task_attempt WHERE task_execution_base = $1", teAID).Scan(&attempts7A); err != nil {
			log.Fatalf("Database validation failed: could not count attempts for poison task A: %v", err)
		}
		var job7DeadLetter bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE task_execution_id = $1 AND status = 'dead_letter')", teAID).Scan(&job7DeadLetter); err != nil {
			log.Fatalf("Database validation failed: could not check job status for poison task A: %v", err)
		}

		var teBID string
		var teBAssigned, teBCompleted bool
		errB := conn.QueryRow(ctx, "SELECT id, task_assigned, task_completed FROM task_execution WHERE workflow_id = $1 AND task_base = $2", executionID7, taskB7).Scan(&teBID, &teBAssigned, &teBCompleted)
		var attempts7B int
		if errB == nil {
			if err := conn.QueryRow(ctx, "SELECT count(*) FROM task_attempt WHERE task_execution_base = $1", teBID).Scan(&attempts7B); err != nil {
				log.Fatalf("Database validation failed: could not count attempts for poison task B: %v", err)
			}
		}

		if teAFailed && attempts7A >= 3 && job7DeadLetter &&
			errB == nil && !teBAssigned && !teBCompleted && attempts7B == 0 {
			poisonPrevented = true
			log.Printf("[DB OK] poison DAG terminal state: A failed after %d attempts (job dead_lettered); B never executed (task_assigned=%t, task_completed=%t, attempts=%d)\n",
				attempts7A, teBAssigned, teBCompleted, attempts7B)
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !poisonPrevented {
		log.Fatalf("Database validation failed: poison DAG never reached terminal state within 60s (A failed+dead_lettered, B never executed)")
	}
	log.Println("Workflow task B was never executed after A's retry exhaustion (poison prevented).")
	log.Println("=== Test Case 7 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 8: 30-Worker Lease-Claim Race (one job, concurrent polls)
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 8: 30-Worker Lease-Claim Race ===")
	// A standalone job's capability requirement is its queue name, so the
	// job is submitted to a queue named for the racers' capability. The
	// lowercase name is picked deliberately: the stack's real workers
	// register the exact (case-sensitive) capability "NOOP", so they can
	// never steal this queue's job from the racers.
	const raceQueue = "noop"
	jobID8, _ := submitStandaloneJob(gclient, raceQueue, "{}", 3, "")
	log.Printf("Submitted single race job: %s (queue %s)\n", jobID8, raceQueue)

	const racers8 = 30
	pollErrors := make([]error, racers8)
	claimedByID := make(map[string]int) // claimer job id -> count across all racers
	var mu8 sync.Mutex
	var wg8 sync.WaitGroup
	for i := 0; i < racers8; i++ {
		wg8.Add(1)
		go func(idx int) {
			defer wg8.Done()
			pollCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			resp, err := gclient.PollJobs(pollCtx, &pb.PollRequest{
				WorkerId:     randomUUID(),
				Capabilities: []string{raceQueue},
				FreeSlots:    1,
			})
			mu8.Lock()
			defer mu8.Unlock()
			if err != nil {
				pollErrors[idx] = err
				return
			}
			for _, j := range resp.Jobs {
				claimedByID[j.Id]++
			}
		}(i)
	}
	wg8.Wait()

	for i, pollErr := range pollErrors {
		if pollErr != nil {
			log.Fatalf("Poll racer %d failed: %v", i, pollErr)
		}
	}
	// Scoping: only THIS run's race job counts. A leftover race job from a
	// previous suite run (queue "noop", lease expired then requeued by the
	// recovery sweep) may legally be claimed by a racer too.
	totalRaceClaims8 := claimedByID[jobID8]
	if totalRaceClaims8 != 1 {
		log.Fatalf("Expected exactly 1 of %d concurrent pollers to claim job %s, got %d claims", racers8, jobID8, totalRaceClaims8)
	}
	claimsForPollersWithRace8 := 0
	for id, count := range claimedByID {
		if id != jobID8 {
			claimsForPollersWithRace8 += count
		}
	}
	log.Printf("[gRPC OK] %d concurrent pollers raced: exactly 1 winner claimed race job %s (plus %d claims of leftover jobs from earlier runs)\n",
		racers8, jobID8, claimsForPollersWithRace8)

	time.Sleep(2 * time.Second)
	var executionCount8 int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM executions WHERE job_id = $1", jobID8).Scan(&executionCount8)
	if err != nil {
		log.Fatalf("Database validation failed: could not count executions for race job: %v", err)
	}
	if executionCount8 != 1 {
		log.Fatalf("Database validation failed: expected exactly 1 execution row for job %s, got %d", jobID8, executionCount8)
	}
	var leaseCount8 int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM worker_leases WHERE job_id = $1", jobID8).Scan(&leaseCount8)
	if err != nil {
		log.Fatalf("Database validation failed: could not count leases for race job: %v", err)
	}
	if leaseCount8 != 1 {
		log.Fatalf("Database validation failed: expected exactly 1 worker_leases row for job %s, got %d", jobID8, leaseCount8)
	}
	log.Println("[DB OK] single claimed job maps to exactly 1 execution row and 1 worker lease")
	log.Println("=== Test Case 8 Passed ===")

	// ----------------------------------------------------
	// Test Scenario 9: Unified CLI over the gRPC Port
	// ----------------------------------------------------
	log.Println("\n=== Starting Test Case 9: Unified CLI over gRPC Port ===")

	stdout, stderr, err9 := runCLI("pause-queue", "-queue", "default")
	if err9 != nil {
		log.Fatalf("CLI pause-queue failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	if !strings.Contains(stdout, "paused") {
		log.Fatalf("CLI pause-queue output missing 'paused': %s", stdout)
	}
	log.Printf("[CLI OK] pause-queue default: %s\n", strings.TrimSpace(stdout))

	stdout, stderr, err9 = runCLI("inspect-queue", "-queue", "default")
	if err9 != nil {
		log.Fatalf("CLI inspect-queue failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	// Assert structure, not exact wording: the metrics block with its counts.
	if !strings.Contains(stdout, "Queue:") || !strings.Contains(stdout, "Pending:") {
		log.Fatalf("CLI inspect-queue output missing queue metrics: %s", stdout)
	}
	log.Printf("[CLI OK] inspect-queue default:\n%s\n", stdout)

	stdout, stderr, err9 = runCLI("resume-queue", "-queue", "default")
	if err9 != nil {
		log.Fatalf("CLI resume-queue failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	if !strings.Contains(stdout, "resumed") {
		log.Fatalf("CLI resume-queue output missing 'resumed': %s", stdout)
	}
	log.Printf("[CLI OK] resume-queue default: %s\n", strings.TrimSpace(stdout))

	stdout, stderr, err9 = runCLI("list-workers")
	if err9 != nil {
		log.Fatalf("CLI list-workers failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	log.Printf("[CLI OK] list-workers\n")

	// Leave-and-fail job: the payload "FAIL" is not valid JSON, so the
	// worker fails it on every attempt until the coordinator exhausts the
	// retry budget. Submitted to queue "NOOP" since a standalone job's
	// capability requirement is its queue name and "NOOP" is a real worker
	// capability.
	jobID9, status9 := submitStandaloneJob(gclient, "NOOP", `{"input_data":"FAIL"}`, 2, "")
	log.Printf("Submitted CLI-managed failing job: %s (status %s)\n", jobID9, status9)

	deadline9 := time.Now().Add(60 * time.Second)
	for {
		var deadLettered bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM dead_letter_jobs WHERE job_id = $1)", jobID9).Scan(&deadLettered)
		if err != nil {
			log.Fatalf("Database validation failed: could not check dead_letter state for job %s: %v", jobID9, err)
		}
		if deadLettered {
			break
		}
		if time.Now().After(deadline9) {
			log.Fatalf("Database validation failed: job %s never reached dead_letter within 60s", jobID9)
		}
		time.Sleep(2 * time.Second)
	}
	var status9DB string
	err = conn.QueryRow(ctx, "SELECT status FROM jobs WHERE id = $1", jobID9).Scan(&status9DB)
	if err != nil {
		log.Fatalf("Database validation failed: could not read status for job %s: %v", jobID9, err)
	}
	if status9DB != "dead_letter" {
		log.Fatalf("Database validation failed: expected dead_letter status for job %s, got %s", jobID9, status9DB)
	}
	log.Printf("[DB OK] job %s is dead_lettered\n", jobID9)

	// Pause the job's own queue so the retried job stays 'pending' for a
	// stable assertion before the real workers can claim it again.
	stdout, stderr, err9 = runCLI("pause-queue", "-queue", "NOOP")
	if err9 != nil {
		log.Fatalf("CLI pause-queue (NOOP) failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}

	stdout, stderr, err9 = runCLI("retry-failed-job", "--job", jobID9)
	if err9 != nil {
		log.Fatalf("CLI retry-failed-job failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	if !strings.Contains(stdout, jobID9) {
		log.Fatalf("CLI retry-failed-job output missing job id %s:\nstdout: %s\nstderr: %s", jobID9, stdout, stderr)
	}
	log.Printf("[CLI OK] retry-failed-job %s: %s\n", jobID9, strings.TrimSpace(stdout))

	deadlineRetry9 := time.Now().Add(15 * time.Second)
	for {
		var retryStatus9 string
		var retryCount9 int
		err = conn.QueryRow(ctx, "SELECT status, retry_count FROM jobs WHERE id = $1", jobID9).Scan(&retryStatus9, &retryCount9)
		if err != nil {
			log.Fatalf("Database validation failed: could not read retried job state: %v", err)
		}
		var deadLetterRows9 int
		err = conn.QueryRow(ctx, "SELECT count(*) FROM dead_letter_jobs WHERE job_id = $1", jobID9).Scan(&deadLetterRows9)
		if err != nil {
			log.Fatalf("Database validation failed: could not count dead_letter rows for job %s: %v", jobID9, err)
		}
		if retryStatus9 == "pending" && retryCount9 == 0 && deadLetterRows9 == 0 {
			log.Printf("[DB OK] retried job %s is pending (retry_count=0, dead_letter record removed)\n", jobID9)
			break
		}
		if time.Now().After(deadlineRetry9) {
			log.Fatalf("Database validation failed: retried job %s did not settle to pending (status=%s retry_count=%d dead_letter_rows=%d)",
				jobID9, retryStatus9, retryCount9, deadLetterRows9)
		}
		time.Sleep(1 * time.Second)
	}
	stdout, stderr, err9 = runCLI("resume-queue", "-queue", "NOOP")
	if err9 != nil {
		log.Fatalf("CLI resume-queue (NOOP) failed: %v\nstdout: %s\nstderr: %s", err9, stdout, stderr)
	}
	log.Println("=== Test Case 9 Passed ===")

	log.Println("\nAll Test Cases executed and verified successfully!")
}


func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func apiURL(base, path string) string {
	return fmt.Sprintf("%s%s", base, path)
}

func postJSON(base, path string, payload interface{}) *http.Response {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("could not marshal payload for %s: %v", path, err)
	}
	response, err := http.Post(apiURL(base, path), "application/json", bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("request failed for %s: %v", path, err)
	}
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		log.Fatalf("request %s returned %d: %s", path, response.StatusCode, string(body))
	}
	return response
}

func createWorkflow(base, name string) models.WorkflowDefinitionID {
	request := models.WorkflowDefinition{WorkflowName: name}
	response := postJSON(base, "/workflow/create", request)
	defer response.Body.Close()
	var workflow models.WorkflowDefinition
	if err := json.NewDecoder(response.Body).Decode(&workflow); err != nil {
		log.Fatalf("could not decode workflow response: %v", err)
	}
	return workflow.ID
}

func createTask(base string, workflowID models.WorkflowDefinitionID, taskType models.TaskType, taskName string, taskInput models.InputFileID, taskInputTask models.TaskDefinitionID) models.TaskDefinitionID {
	request := models.TaskDefinition{
		WorkflowID:    workflowID,
		TaskType:      taskType,
		TaskName:      taskName,
		TaskInput:     taskInput,
		TaskInputTask: taskInputTask,
	}
	response := postJSON(base, "/workflow/add", request)
	defer response.Body.Close()
	var task models.TaskDefinition
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		log.Fatalf("could not decode task response: %v", err)
	}
	return task.ID
}

func attachFileToWorkflow(base string, workflowID models.WorkflowDefinitionID, file models.InputFile) {
	// No-op: workflows no longer have global inputs/outputs.
}

func startWorkflow(base string, workflowID models.WorkflowDefinitionID) models.WorkflowExecutionID {
	request := models.WorkflowDefinition{ID: workflowID}
	response := postJSON(base, "/workflow/", request)
	defer response.Body.Close()
	var execution models.WorkflowExecution
	if err := json.NewDecoder(response.Body).Decode(&execution); err != nil {
		log.Fatalf("could not decode workflow execution response: %v", err)
	}
	return execution.ID
}

func getWorkflowTaskExecutions(base string, executionID models.WorkflowExecutionID) []models.TaskExecution {
	request := models.WorkflowExecution{ID: executionID}
	response := postJSON(base, "/workflow/status", request)
	defer response.Body.Close()
	var tasks []models.TaskExecution
	if err := json.NewDecoder(response.Body).Decode(&tasks); err != nil {
		log.Fatalf("could not decode task executions: %v", err)
	}
	return tasks
}

func pollWorkflowStatus(base string, executionID models.WorkflowExecutionID) {
	for i := 0; i < 40; i++ {
		tasks := getWorkflowTaskExecutions(base, executionID)
		if len(tasks) == 0 {
			log.Printf("workflow has no task executions yet")
		} else {
			allCompleted := true
			for _, t := range tasks {
				if !t.TaskCompleted {
					allCompleted = false
					break
				}
			}
			if allCompleted {
				log.Printf("workflow execution %s complete\n", executionID)
				return
			}
			completedCount := 0
			for _, t := range tasks {
				if t.TaskCompleted {
					completedCount++
				}
			}
			log.Printf("workflow execution %s waiting, %d/%d tasks complete\n", executionID, completedCount, len(tasks))
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("workflow execution %s did not complete in time", executionID)
}

func uploadFile(base, path string) models.InputFile {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("could not open file %s: %v", path, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		log.Fatalf("could not create form file: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		log.Fatalf("could not copy file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		log.Fatalf("could not close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL(base, "/file"), body)
	if err != nil {
		log.Fatalf("could not create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("file upload failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		log.Fatalf("file upload returned %d: %s", response.StatusCode, string(body))
	}
	var uploaded models.InputFile
	if err := json.NewDecoder(response.Body).Decode(&uploaded); err != nil {
		log.Fatalf("could not decode file upload response: %v", err)
	}
	return uploaded
}

func downloadFile(base string, fileID models.InputFileID) string {
	url := fmt.Sprintf("%s?id=%s", apiURL(base, "/file"), string(fileID))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("could not build get file request: %v", err)
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to get file: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		log.Fatalf("get file returned %d: %s", response.StatusCode, string(body))
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("could not read file content: %v", err)
	}
	return string(content)
}

func verifyDatabasePersistence(ctx context.Context, conn *pgx.Conn, workflowName string, executionID models.WorkflowExecutionID, expectedTaskCount int) {
	var wfID string
	err := conn.QueryRow(ctx, "SELECT id FROM workflow_definition WHERE workflow_name = $1", workflowName).Scan(&wfID)
	if err != nil {
		log.Fatalf("Database validation failed: could not find workflow definition %s: %v", workflowName, err)
	}
	log.Printf("[DB OK] found workflow definition %s (ID: %s)\n", workflowName, wfID)

	var taskCount int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM task_definition WHERE workflow_id = $1", wfID).Scan(&taskCount)
	if err != nil {
		log.Fatalf("Database validation failed: could not count task definitions: %v", err)
	}
	if taskCount != expectedTaskCount {
		log.Fatalf("Database validation failed: expected %d task definitions, got %d", expectedTaskCount, taskCount)
	}
	log.Printf("[DB OK] found %d task definitions for workflow\n", taskCount)

	var execID string
	err = conn.QueryRow(ctx, "SELECT id FROM workflow_execution WHERE id = $1", string(executionID)).Scan(&execID)
	if err != nil {
		log.Fatalf("Database validation failed: could not find workflow execution %s: %v", executionID, err)
	}
	log.Printf("[DB OK] found workflow execution (ID: %s)\n", execID)

	var completeCount int
	// Terminal count = completed OR failed (Phase 3+ semantics: retry
	// exhaustion marks the task failed, not completed).
	err = conn.QueryRow(ctx, "SELECT count(*) FROM task_execution WHERE workflow_id = $1 AND (task_completed = TRUE OR task_failed = TRUE)", execID).Scan(&completeCount)
	if err != nil {
		log.Fatalf("Database validation failed: could not count task executions: %v", err)
	}
	if completeCount != expectedTaskCount {
		log.Fatalf("Database validation failed: expected %d terminal task executions, got %d", expectedTaskCount, completeCount)
	}
	log.Printf("[DB OK] found %d terminal task executions\n", completeCount)

	rows, err := conn.Query(ctx, "SELECT id FROM task_execution WHERE workflow_id = $1", execID)
	if err != nil {
		log.Fatalf("Database validation failed: could not fetch task executions: %v", err)
	}
	
	var teIDs []string
	for rows.Next() {
		var teID string
		if err := rows.Scan(&teID); err != nil {
			rows.Close()
			log.Fatalf("Database scan failed: %v", err)
		}
		teIDs = append(teIDs, teID)
	}
	rows.Close()

	for _, teID := range teIDs {
		var attemptCount int
		err = conn.QueryRow(ctx, "SELECT count(*) FROM task_attempt WHERE task_execution_base = $1", teID).Scan(&attemptCount)
		if err != nil {
			log.Fatalf("Database validation failed: could not count task attempts for task execution %s: %v", teID, err)
		}
		if attemptCount == 0 {
			log.Fatalf("Database validation failed: no task attempts found for task execution %s", teID)
		}
		log.Printf("[DB OK] found %d task attempt(s) for task execution %s\n", attemptCount, teID)
	}
}

// ------------------------------------------------------------
// Helpers for the gRPC-backed cases (6-9)
// ------------------------------------------------------------

// dialQueueGRPC dials the coordinator's gRPC endpoint directly. The compose
// stack publishes port 50051 on the host so host processes (the CLI and this
// harness) can submit/poll over gRPC; QUEUE_HOST/GRPC_PORT override the
// env-agnostic defaults.
func dialQueueGRPC() (*grpc.ClientConn, pb.TaskTideServiceClient) {
	queueHost := getEnv("QUEUE_HOST", "127.0.0.1")
	grpcPort := getEnv("GRPC_PORT", "50051")
	addr := fmt.Sprintf("%s:%s", queueHost, grpcPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not dial coordinator gRPC at %s: %v", addr, err)
	}
	return conn, pb.NewTaskTideServiceClient(conn)
}

// submitStandaloneJob submits a single standalone (non-workflow) job over
// gRPC and fails the harness on submission errors.
func submitStandaloneJob(client pb.TaskTideServiceClient, queue, payload string, maxRetries int32, idempotencyKey string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.SubmitJob(ctx, &pb.SubmitRequest{
		QueueName:      queue,
		PayloadJson:    payload,
		MaxRetries:     maxRetries,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		log.Fatalf("SubmitJob to queue '%s' failed: %v", queue, err)
	}
	return resp.JobId, resp.Status
}

// randomUUID returns a valid format-v4 UUID string for synthetic worker IDs.
func randomUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("could not generate worker id: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// writeTmpFile writes a fixture file under tmp/ and returns its path (same
// pattern as cases 4 and 5).
func writeTmpFile(name, content string) string {
	path := "tmp/" + name
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.Fatalf("failed to write tmp file %s: %v", path, err)
	}
	return path
}

// runCLI runs one CLI subcommand via `go run ./cmd/cli`, pinning the gRPC
// dial env so a container-network QUEUE_HOST in .env cannot leak into the
// subprocess. Returns (stdout, stderr, run error).
func runCLI(args ...string) (string, string, error) {
	cmd := exec.Command("go", append([]string{"run", "./cmd/cli"}, args...)...)
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "QUEUE_HOST=") || strings.HasPrefix(e, "GRPC_PORT=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "QUEUE_HOST=127.0.0.1", "GRPC_PORT=50051")
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
