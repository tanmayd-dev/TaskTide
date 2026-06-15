package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

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
	verifyDatabasePersistence(ctx, conn, "ChainedWorkflow", 4)
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
	verifyDatabasePersistence(ctx, conn, "IndependentWorkflow", 3)
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
	verifyDatabasePersistence(ctx, conn, "MixedWorkflow", 5)
	log.Println("=== Test Case 3 Passed ===")

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
	request := models.InputFile{ID: file.ID, WorkflowID: workflowID}
	response := postJSON(base, "/file/workflow", request)
	response.Body.Close()
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
	request := models.InputFile{ID: fileID}
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("could not marshal get file request: %v", err)
	}
	req, err := http.NewRequest("GET", apiURL(base, "/file"), bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("could not build get file request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
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

func verifyDatabasePersistence(ctx context.Context, conn *pgx.Conn, workflowName string, expectedTaskCount int) {
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
	err = conn.QueryRow(ctx, "SELECT id FROM workflow_execution WHERE workflow_base = $1 ORDER BY id DESC LIMIT 1", wfID).Scan(&execID)
	if err != nil {
		log.Fatalf("Database validation failed: could not find workflow execution: %v", err)
	}
	log.Printf("[DB OK] found workflow execution (ID: %s)\n", execID)

	var completeCount int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM task_execution WHERE workflow_id = $1 AND task_completed = TRUE", execID).Scan(&completeCount)
	if err != nil {
		log.Fatalf("Database validation failed: could not count task executions: %v", err)
	}
	if completeCount != expectedTaskCount {
		log.Fatalf("Database validation failed: expected %d completed task executions, got %d", expectedTaskCount, completeCount)
	}
	log.Printf("[DB OK] found %d completed task executions\n", completeCount)

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
