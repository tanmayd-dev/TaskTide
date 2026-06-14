package queue

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func enqueue(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse task", http.StatusBadRequest)
			return
		}
		taskUUID, err := utils.StringToUUID(string(task.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		taskInputUUID, err := utils.StringToUUID(string(task.TaskInput))
		if err != nil {
			log.Printf("No task input: %v\n", err.Error())
		}
		taskOutputUUID, err := utils.StringToUUID(string(task.TaskOutput))
		if err != nil {
			log.Printf("No task output: %v\n", err.Error())
		}
		workflowUUID, err := utils.StringToUUID(string(task.WorkflowID))
		if err != nil {
			log.Printf("could not parse workflow ID: %v\n", err.Error())
			http.Error(w, "Could not parse workflow ID", http.StatusBadRequest)
			return
		}
		var workflowExecutionID pgtype.UUID
		if task.WorkflowExecutionID != "" {
			workflowExecutionID, err = utils.StringToUUID(string(task.WorkflowExecutionID))
			if err != nil {
				log.Printf("could not parse workflow execution ID: %v\n", err.Error())
				http.Error(w, "Could not parse workflow execution ID", http.StatusBadRequest)
				return
			}
		} else {
			workflowExecution, err := queries.CreateWorkflowExecution(ctx, workflowUUID)
			if err != nil {
				log.Printf("could not create workflow execution: %v\n", err.Error())
				http.Error(w, "Could not create workflow execution", http.StatusInternalServerError)
				return
			}
			workflowExecutionID = workflowExecution.ID
		}
		_, err = queries.CreateTaskExecution(ctx, db.CreateTaskExecutionParams{
			TaskBase:   taskUUID,
			TaskInput:  taskInputUUID,
			TaskOutput: taskOutputUUID,
			WorkflowID: workflowExecutionID,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not create task Execution", http.StatusBadRequest)
			return
		}

		go queueService.AssignWorker(queries, ctx, minioClient)
		w.WriteHeader(http.StatusAccepted)
	}
}

func poll(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go queueService.AssignWorker(queries, ctx, minioClient)
		w.WriteHeader(http.StatusNoContent)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", poll(queries, ctx, minioClient))
		r.Post("/", enqueue(queries, ctx, minioClient))
	}
}
