package workerService

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func Execute(queries *db.Queries, ctx context.Context, minioClient *minio.Client, task models.TaskExecution, workerID models.WorkerID) {
	taskBaseUUID, err := utils.StringToUUID(string(task.TaskBase))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskUUID, err := utils.StringToUUID(string(task.ID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	inputUUID, err := utils.StringToUUID(string(task.TaskInput))
	if err != nil {
		log.Printf("No input: %v\n", err.Error())
		inputUUID.Valid = false
	}
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskAttempt, err := queries.CreateTaskAttempt(ctx, db.CreateTaskAttemptParams{
		TaskExecutionBase: taskUUID,
		TaskInput:         inputUUID,
	})
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskAttemptModel := utils.TaskAttemptModel(taskAttempt)
	if !inputUUID.Valid {
		taskAttemptModel.ID = ""
	}
	taskBase, err := queries.GetTask(ctx, taskBaseUUID)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	var taskError error
	switch models.TaskType(taskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop(queries, ctx, minioClient, task, taskAttemptModel)
	case models.TaskOUTPUT:
		taskError = tasks.Output(queries, ctx, minioClient, task, taskAttemptModel)
	default:
		log.Printf("Not implemented for %v\n", taskBase.TaskType)
	}
	err = queries.MarkTaskComplete(ctx, taskUUID)
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	err = queries.MarkWorkerAvailable(ctx, workerUUID)
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	if taskError != nil {
		queries.FailTask(ctx, taskUUID)
		queries.MarkTaskAttemptFailure(ctx, taskAttempt.ID)
	} else {
		queries.SucceedTask(ctx, taskUUID)
		queries.MarkTaskAttemptSuccess(ctx, taskAttempt.ID)
		if models.TaskType(taskBase.TaskType) == models.TaskOUTPUT && task.TaskOutput != "" {
			workflowExecutionID, err := utils.StringToUUID(string(task.WorkflowID))
			if err == nil {
				workflowExecution, err := queries.GetWorkflowExecution(ctx, workflowExecutionID)
				if err == nil {
					workflowBaseUUID, err := utils.StringToUUID(workflowExecution.WorkflowBase.String())
					if err == nil {
						taskOutputUUID, err := utils.StringToUUID(string(task.TaskOutput))
						if err == nil {
							err = queries.UpdateWorkflowOutput(ctx, db.UpdateWorkflowOutputParams{
								ID:             workflowBaseUUID,
								WorkflowOutput: taskOutputUUID,
							})
							if err != nil {
								log.Printf("Could not update workflow output: %v\n", err.Error())
							}
						}
					}
				}
			}
		}
	}
	_, err = http.Post(
		fmt.Sprintf("http://%v:%v/worker/completion", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader([]byte{}),
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
}
