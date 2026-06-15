package worker

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func registerWorker(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		worker, err := utils.ReadRequest[models.Worker](w, r)
		if utils.HasError(w, r, "could not read worker from request", err) {
			return
		}
		workerModel, err := repositories.AddWorker(ctx, queries, worker)
		if utils.HasError(w, r, "could not add worker", err) {
			return
		}
		go queueService.AssignWorker(queries, ctx, minioClient)
		addedWorkerModel := models.Worker{
			ID: workerModel.ID,
		}
		utils.WriteJSON(w, http.StatusOK, addedWorkerModel)
	}
}

func handleCompletion(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		completion, err := utils.ReadRequest[models.WorkerTaskExecution](w, r)
		if utils.HasError(w, r, "Could not parse completion payload", err) {
			return
		}

		taskExecution, err := repositories.GetTaskExecution(ctx, queries, completion.ID)
		if utils.HasError(w, r, "Could not get task execution details", err) {
			return
		}

		if completion.TaskSucceeded && taskExecution.TaskOutput != "" {
			err = repositories.UpdateFileContent(ctx, queries, minioClient, os.Getenv("FILES_BUCKET_NAME"), taskExecution.TaskOutput, completion.TaskOutput)
			if utils.HasError(w, r, "Could not save output to storage", err) {
				return
			}
		}

		err = repositories.CreateTaskAttempt(ctx, queries, completion)
		if utils.HasError(w, r, "Could not record task attempt", err) {
			return
		}

		err = repositories.MarkTaskComplete(ctx, queries, completion.ID)
		if utils.HasError(w, r, "Could not update task execution status", err) {
			return
		}

		if taskExecution.TaskWorker != "" {
			err = repositories.MarkWorkerAvailable(ctx, queries, taskExecution.TaskWorker)
			if err != nil {
				log.Printf("could not mark worker available: %v\n", err)
			}
		}

		go queueService.AssignWorker(queries, ctx, minioClient)

		w.WriteHeader(http.StatusOK)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", registerWorker(queries, ctx, minioClient))
		r.Post("/completion", handleCompletion(queries, ctx, minioClient))
	}
}
