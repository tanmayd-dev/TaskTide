package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	workflowID := createWorkflow(baseURL, "EndToEndSample")
	log.Println("created workflow", workflowID)

	file1 := uploadFile(baseURL, "tmp/sample1.txt")
	attachFileToWorkflow(baseURL, workflowID, file1)
	log.Printf("uploaded file: %s\n", file1.FileName)

	// Create a simple chain: A -> B -> C -> OUTPUT
	chainTaskA := createTask(baseURL, workflowID, models.TaskNOOP, "Chain Task A", file1.ID, "")
	chainTaskB := createTask(baseURL, workflowID, models.TaskNOOP, "Chain Task B", "", chainTaskA)
	chainTaskC := createTask(baseURL, workflowID, models.TaskNOOP, "Chain Task C", "", chainTaskB)
	outputTask := createTask(baseURL, workflowID, models.TaskOUTPUT, "Chain Task Output", "", chainTaskC)

	log.Printf("created chained tasks %s -> %s -> %s -> %s\n", chainTaskA, chainTaskB, chainTaskC, outputTask)

	// Show workflow and start it
	workflow := getWorkflow(baseURL, workflowID)
	log.Printf("workflow details: %+v\n", workflow)

	executionID := startWorkflow(baseURL, workflowID)
	log.Printf("workflow execution requested: %s", executionID)
	pollWorkflowStatus(baseURL, executionID)

	// workflowNoInputID := createWorkflow(baseURL, "EndToEndNoInput")
	// log.Println("created no-input workflow", workflowNoInputID)
	// taskNoInput := createTask(baseURL, workflowNoInputID, models.TaskNOOP, "No Input Task", "", "")
	// taskDependent := createTask(baseURL, workflowNoInputID, models.TaskNOOP, "Dependent Task", "", "")
	// addDependency(baseURL, taskDependent, []models.TaskDefinitionID{taskNoInput}, taskNoInput)
	// log.Println("created no-input workflow with dependency and default null input branch")

	// startWorkflow(baseURL, workflowNoInputID)
	// log.Println("no-input workflow execution requested")
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
	log.Printf("created task %s (%s) input=%s input_task=%s", taskName, task.ID, taskInput, task.TaskInputTask)
	return task.ID
}

func addDependency(base string, taskID models.TaskDefinitionID, dependencies []models.TaskDefinitionID, inputFromDependency models.TaskDefinitionID) {
	request := models.TaskDependency{ID: taskID, Dependencies: dependencies, InputFromDependency: inputFromDependency}
	response := postJSON(base, "/workflow/dependency", request)
	response.Body.Close()
	log.Printf("added dependency for %s -> %v inputFrom=%s", taskID, dependencies, inputFromDependency)
}

func attachFileToWorkflow(base string, workflowID models.WorkflowDefinitionID, file models.InputFile) {
	request := models.InputFile{ID: file.ID, WorkflowID: workflowID}
	response := postJSON(base, "/file/workflow", request)
	response.Body.Close()
	log.Printf("attached file %s to workflow %s", file.FileName, workflowID)
}

func getWorkflow(base string, workflowID models.WorkflowDefinitionID) models.FatWorkflowDefinition {
	request := models.WorkflowDefinition{ID: workflowID}
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("could not marshal get workflow request: %v", err)
	}

	req, err := http.NewRequest("GET", apiURL(base, "/workflow/"), bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("could not build get workflow request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to get workflow: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		log.Fatalf("get workflow returned %d: %s", response.StatusCode, string(body))
	}
	var workflow models.FatWorkflowDefinition
	if err := json.NewDecoder(response.Body).Decode(&workflow); err != nil {
		log.Fatalf("could not decode workflow: %v", err)
	}
	return workflow
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

func pollWorkflowStatus(base string, executionID models.WorkflowExecutionID) {
	request := models.WorkflowExecution{ID: executionID}
	for i := 0; i < 30; i++ {
		response := postJSON(base, "/workflow/status", request)
		var tasks []models.TaskExecution
		if err := json.NewDecoder(response.Body).Decode(&tasks); err != nil {
			response.Body.Close()
			log.Fatalf("could not decode workflow status response: %v", err)
		}
		response.Body.Close()
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
				log.Printf("workflow execution %s complete", executionID)
				return
			}
			completedCount := 0
			for _, t := range tasks {
				if t.TaskCompleted {
					completedCount++
				}
			}
			log.Printf("workflow execution %s waiting, %d/%d tasks complete", executionID, completedCount, len(tasks))
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
	log.Printf("uploaded file %s as %s", uploaded.FileName, uploaded.ID)
	return uploaded
}
