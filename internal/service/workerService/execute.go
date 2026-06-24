package workerService

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
)

func ExecuteJob(ctx context.Context, client pb.TaskTideServiceClient, workerID string, job *pb.Job) {
	var payloadStruct struct {
		Task      models.TaskDefinition `json:"task"`
		InputData string                `json:"input_data"`
	}

	err := json.Unmarshal([]byte(job.Payload), &payloadStruct)
	if err != nil {
		// Fallback to direct unmarshal for standalone jobs
		var taskDef models.TaskDefinition
		errFallback := json.Unmarshal([]byte(job.Payload), &taskDef)
		if errFallback != nil {
			log.Printf("ExecuteJob: failed to parse payload: %v\n", errFallback)
			_, _ = client.CompleteJob(ctx, &pb.CompleteRequest{
				JobId:       job.Id,
				WorkerId:    workerID,
				Success:     false,
				ErrorReason: errFallback.Error(),
			})
			return
		}
		payloadStruct.Task = taskDef
	}

	taskExec := models.WorkerTaskExecution{
		ID:         models.TaskExecutionID(job.Id),
		TaskBase:   payloadStruct.Task,
		TaskWorker: models.WorkerID(workerID),
		TaskInput:  payloadStruct.InputData,
	}

	var taskError error
	switch models.TaskType(payloadStruct.Task.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop(&taskExec)
	case models.TaskOUTPUT:
		taskError = tasks.Output(&taskExec)
	case models.TaskDocRead:
		taskError = tasks.DocRead(&taskExec)
	case models.TaskCSVRead:
		taskError = tasks.CSVRead(&taskExec)
	default:
		taskError = errors.New("task type not implemented")
		log.Printf("ExecuteJob: not implemented for %v\n", payloadStruct.Task.TaskType)
	}

	success := taskError == nil
	var errReason string
	if !success {
		errReason = taskError.Error()
	}

	_, rpcErr := client.CompleteJob(ctx, &pb.CompleteRequest{
		JobId:       job.Id,
		WorkerId:    workerID,
		Success:     success,
		OutputData:  taskExec.TaskOutput,
		ErrorReason: errReason,
	})
	if rpcErr != nil {
		log.Printf("ExecuteJob: failed to complete job via RPC: %v\n", rpcErr)
	}
}
