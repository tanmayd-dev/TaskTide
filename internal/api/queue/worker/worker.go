package worker

import (
	"context"
	"encoding/json"
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

func registerWorker(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		worker, err := utils.ReadRequest[models.Worker](w, r)
		workerDB, err := repositories.AddWorker(ctx, queries, worker)
		if utils.HasError(w, r, "could not add worker", err) {
			return
		}
		workerModel := utils.WorkerModel(workerDB)
		go queueService.AssignWorker(queries, ctx, minioClient)
		addedWorkerModel := models.Worker{
			ID: models.WorkerID(workerModel.ID),
		}
		json.NewEncoder(w).Encode(addedWorkerModel)
	}
}

func handleCompletion(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		completion, err := utils.ReadRequest[models.WorkerTaskExecution](w, r)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse form", http.StatusBadRequest)
			return
		}
		err = repositories.CreateTaskAttempt(ctx, completion, w, queries)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse form", http.StatusBadRequest)
			return
		}
		go queueService.AssignWorker(queries, ctx, minioClient)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", registerWorker(queries, ctx, minioClient))
		r.Post("/completion", handleCompletion(queries, ctx, minioClient))
	}
}
