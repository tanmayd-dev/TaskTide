package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func main() {
	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	// Create Workflow
	var workflowID models.WorkflowDefinitionID
	log.Println("Creating Workflow")
	{
		request := models.WorkflowDefinition{
			WorkflowName: "Sample",
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/create", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		var workflow models.WorkflowDefinition
		err = json.NewDecoder(response.Body).Decode(&workflow)
		if err != nil {
			log.Fatalf("could not parse created workflow: %#v", err)
		}
		workflowID = workflow.ID
	}

	// Create Tasks
	var t1, t2, t3 models.TaskDefinitionID
	// t1
	log.Println("Creating taskID 1")
	{
		request := models.TaskDefinition{
			WorkflowID: workflowID,
			TaskType:   models.TaskNOOP,
			TaskName:   "SampleTask",
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/add", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		var task models.TaskDefinition
		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			log.Fatalf("could not parse created workflow: %#v", err)
		}
		t1 = task.ID
	}
	// t2
	log.Println("Creating taskID 2")
	{
		request := models.TaskDefinition{
			WorkflowID: workflowID,
			TaskType:   models.TaskNOOP,
			TaskName:   "SampleTask",
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/add", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		var task models.TaskDefinition
		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			log.Fatalf("could not parse created workflow: %#v", err)
		}
		t2 = task.ID
	}
	// t3
	log.Println("Creating taskID 3")
	{
		request := models.TaskDefinition{
			WorkflowID: workflowID,
			TaskType:   models.TaskNOOP,
			TaskName:   "SampleTask",
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/add", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		var task models.TaskDefinition
		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			log.Fatalf("could not parse created workflow: %#v", err)
		}
		t3 = task.ID
	}

	// Create Task Dependencies
	log.Println("Creating Dependencies")
	{
		request := models.TaskDependency{
			ID:           t1,
			Dependencies: []models.TaskDefinitionID{t2, t3},
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/dependency", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		log.Printf("%#v\n", response)
	}

	// Get Workflow
	log.Println("Getting workflow")
	{
		request := models.WorkflowDefinition{
			ID: workflowID,
		}
		requestBody, err := json.Marshal(request)
		if err != nil {
			log.Fatalf("could not marshal request body: %#v", err)
		}
		apiURL := fmt.Sprintf("http://%s:%s/workflow/", os.Getenv("API_HOST"), os.Getenv("API_PORT"))
		req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer(requestBody))
		if err != nil {
			log.Fatalf("could not create workflow request: %#v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		response, err := client.Do(req)
		if err != nil {
			log.Fatalf("could not execute workflow request: %#v", err)
		}
		defer response.Body.Close()
		var workflow models.FatWorkflowDefinition
		err = json.NewDecoder(response.Body).Decode(&workflow)
		if err != nil {
			log.Fatalf("could not parse created workflow response: %#v", err)
		}
		log.Printf("%#v\n", workflow)
	}

	// Execute Workflow
	log.Println("Running Workflow")
	{
		request := models.WorkflowDefinition{
			ID: workflowID,
		}
		requestBody, _ := json.Marshal(request)
		response, err := http.Post(fmt.Sprintf("http://%v:%v/workflow/", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewBufferString(string(requestBody)))
		if err != nil {
			log.Fatalf("could not create workflow: %#v", err)
		}
		log.Printf("%#v\n", response)
	}
}
