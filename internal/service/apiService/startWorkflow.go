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
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func StartWorkflow(queries *db.Queries, ctx context.Context, minioClient *minio.Client, workflowID models.WorkflowDefinitionID) error {
	workflowUUID, err := utils.StringToUUID(string(workflowID))
	if err != nil {
		return err
	}
	workflow, err := queries.GetWorkflow(ctx, workflowUUID)
	if err != nil {
		return err
	}
	workflowInput := models.InputFileID("")
	if workflow.WorkflowInput.Valid {
		workflowInput = models.InputFileID(workflow.WorkflowInput.String())
	}
	workflowExecution, err := queries.CreateWorkflowExecution(ctx, workflowUUID)
	if err != nil {
		return err
	}
	taskDBs, err := queries.GetWorkflowTasks(ctx, workflowUUID)
	if err != nil {
		return err
	}
	dependencies := make(map[models.TaskDefinitionID]map[models.TaskDefinitionID]struct{})
	dependents := make(map[models.TaskDefinitionID][]models.TaskDefinitionID)
	taskIDToModel := make(map[models.TaskDefinitionID]models.TaskDefinition)
	for _, taskDB := range taskDBs {
		dependencies[models.TaskDefinitionID(taskDB.ID.String())] = make(map[models.TaskDefinitionID]struct{})
		dependencyTaskDBs, err := queries.GetTaskDependencies(ctx, taskDB.ID)
		if err != nil {
			return err
		}
		taskModel := utils.TaskDefinitionModel(taskDB)
		taskModel.WorkflowExecutionID = models.WorkflowExecutionID(workflowExecution.ID.String())
		for _, dependencyTaskDB := range dependencyTaskDBs {
			dependencyModel := models.TaskDefinitionID(dependencyTaskDB.TaskDependsOn.String())
			dependencies[models.TaskDefinitionID(taskDB.ID.String())][dependencyModel] = struct{}{}
			dependents[dependencyModel] = append(dependents[dependencyModel], models.TaskDefinitionID(taskDB.ID.String()))
		}
		taskIDToModel[models.TaskDefinitionID(taskDB.ID.String())] = taskModel
	}
	for taskID, task := range taskIDToModel {
		if task.TaskInput == "" {
			if task.TaskInputTask != "" {
				if dependencyTask, ok := taskIDToModel[task.TaskInputTask]; ok && dependencyTask.TaskOutput != "" {
					task.TaskInput = dependencyTask.TaskOutput
				}
			}
			if task.TaskInput == "" && workflowInput != "" {
				task.TaskInput = workflowInput
			}
			if task.TaskInput == "" {
				inputFile, err := repositories.CreateDefaultNullFile(ctx, queries, minioClient, os.Getenv("FILES_BUCKET_NAME"))
				if err != nil {
					return err
				}
				task.TaskInput = models.InputFileID(inputFile.ID.String())
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
				return err
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
				return err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("queue returned status %d", resp.StatusCode)
			}
		}
	}
	return nil
}
