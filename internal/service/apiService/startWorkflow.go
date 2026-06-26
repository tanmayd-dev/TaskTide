package apiservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
)

func StartWorkflow(queries *db.Queries, ctx context.Context, minioClient *minio.Client, workflowID models.WorkflowDefinitionID) (models.WorkflowExecutionID, error) {
	_, err := repositories.GetWorkflow(ctx, queries, workflowID)
	if err != nil {
		return "", err
	}

	workflowExecution, err := repositories.CreateWorkflowExecution(ctx, queries, workflowID)
	if err != nil {
		return "", err
	}
	taskModels, err := repositories.GetWorkflowTasks(ctx, queries, workflowID)
	if err != nil {
		return "", err
	}
	dependencies := make(map[models.TaskDefinitionID]map[models.TaskDefinitionID]struct{})
	dependents := make(map[models.TaskDefinitionID][]models.TaskDefinitionID)
	taskIDToModel := make(map[models.TaskDefinitionID]models.TaskDefinition)
	for _, taskModel := range taskModels {
		dependencies[taskModel.ID] = make(map[models.TaskDefinitionID]struct{})
		dependencyTaskDBs, err := repositories.GetTaskDependencies(ctx, queries, taskModel.ID)
		if err != nil {
			return "", err
		}
		taskModel.WorkflowExecutionID = workflowExecution.ID
		for _, dependencyModel := range dependencyTaskDBs {
			dependencies[taskModel.ID][dependencyModel] = struct{}{}
			dependents[dependencyModel] = append(dependents[dependencyModel], taskModel.ID)
		}
		if taskModel.TaskInputTask != "" {
			dependencies[taskModel.ID][taskModel.TaskInputTask] = struct{}{}
			dependents[taskModel.TaskInputTask] = append(dependents[taskModel.TaskInputTask], taskModel.ID)
		}
		taskIDToModel[taskModel.ID] = taskModel
	}
	for taskID, task := range taskIDToModel {
		if task.TaskOutput == "" {
			outputFile, err := repositories.CreateEmptyFile(ctx, queries, minioClient, os.Getenv("FILES_BUCKET_NAME"), fmt.Sprintf("%s-output", task.TaskName))
			if err != nil {
				return "", err
			}
			task.TaskOutput = outputFile.ID
			taskIDToModel[taskID] = task
			err = repositories.UpdateTaskOutput(ctx, queries, taskID, outputFile.ID)
			if err != nil {
				return "", err
			}
		}
	}
	for taskID, task := range taskIDToModel {
		if task.TaskInput == "" {
			if task.TaskInputTask != "" {
				if dependencyTask, ok := taskIDToModel[task.TaskInputTask]; ok && dependencyTask.TaskOutput != "" {
					task.TaskInput = dependencyTask.TaskOutput
				}
			}
			if task.TaskInput == "" {
				inputFile, err := repositories.CreateDefaultNullFile(ctx, queries, minioClient, os.Getenv("FILES_BUCKET_NAME"))
				if err != nil {
					return "", err
				}
				task.TaskInput = inputFile.ID
			}
			taskIDToModel[taskID] = task
		}
	}
	got := true
	for got {
		got = false
		for taskID, deps := range dependencies {
			if len(deps) > 0 {
				continue
			}
			got = true
			delete(dependencies, taskID)
			for _, dependant := range dependents[taskID] {
				delete(dependencies[dependant], taskID)
			}
			delete(dependents, taskID)
			task := taskIDToModel[taskID]
			jsonTask, err := json.Marshal(task)
			if err != nil {
				fmt.Printf("%#v", err)
				return "", err
			}
			log.Printf("sending: %#v\n", task)
			url := fmt.Sprintf("http://%v:%v/", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
			resp, err := http.Post(
				url,
				"application/json",
				bytes.NewBuffer(jsonTask),
			)
			if err != nil {
				fmt.Printf("%#v", err)
				return "", err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", fmt.Errorf("queue returned status %d", resp.StatusCode)
			}
		}
	}
	return workflowExecution.ID, nil
}
