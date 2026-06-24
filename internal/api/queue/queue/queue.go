package queue

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
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
		taskExec, err := repositories.CreateTaskExecution(ctx, queries, task.ID, task.TaskInput, task.TaskOutput, workflowExecutionID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not create task Execution", http.StatusBadRequest)
			return
		}

		payloadBytes, err := json.Marshal(task)
		if err == nil {
			taskExecUUID, _ := utils.StringToUUID(string(taskExec.ID))
			_, err = queries.CreateJob(ctx, db.CreateJobParams{
				QueueName:       "default",
				Payload:         payloadBytes,
				MaxRetries:      3,
				Delay:           0,
				ScheduledAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
				Priority:        0,
				IdempotencyKey:  pgtype.Text{String: "", Valid: false},
				TaskExecutionID: taskExecUUID,
			})
			if err != nil {
				log.Printf("Failed to create Job for task execution %s: %v\n", taskExec.ID, err)
			} else {
				log.Printf("Successfully created Job for task execution %s\n", taskExec.ID)
			}
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func poll(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", poll(queries, ctx, minioClient))
		r.Post("/", enqueue(queries, ctx, minioClient))
	}
}
