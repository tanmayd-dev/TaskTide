package worker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
)

func registerWorker(queries *db.Queries, ctx context.Context) func(http.ResponseWriter, *http.Request) {
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

func handleCompletion(queries *db.Queries, ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(_ http.ResponseWriter, _ *http.Request) {
		go queueService.AssignWorker(queries, ctx)
	}
}

func Router(queries *db.Queries, ctx context.Context) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", registerWorker(queries, ctx))
		r.Post("/completion", handleCompletion(queries, ctx))
	}
}
