package queue

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func enqueue(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		task, err := utils.ReadRequest[models.TaskDefinition](w, r)
		if utils.HasError(w, r, "could not read request", err) {
			return
		}
		var workflowExecutionID models.WorkflowExecutionID
		if task.WorkflowExecutionID != "" {
			workflowExecutionID = task.WorkflowExecutionID
		} else {
			we, err := repositories.CreateWorkflowExecution(ctx, queries, task.WorkflowID)
			if utils.HasError(w, r, "could not create workflow execution", err) {
				return
			}
			workflowExecutionID = we.ID
		}
		_, err = repositories.CreateTaskExecution(ctx, queries, task.ID, task.TaskInput, task.TaskOutput, workflowExecutionID)
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
