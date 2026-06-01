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
		var worker models.Worker
		err := json.NewDecoder(r.Body).Decode(&worker)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse worker", http.StatusBadRequest)
			return
		}
		addedWorker, err := queries.AddWorker(ctx, db.AddWorkerParams{
			WorkerHost: worker.WorkerHost,
			WorkerPort: worker.WorkerPort,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not register worker to DB", http.StatusBadRequest)
			return
		}
		go queueService.AssignWorker(queries, ctx)
		addedWorkerModel := models.Worker {
			ID: models.WorkerID{
				Bytes: addedWorker.ID.Bytes,
				Valid: addedWorker.ID.Valid,
			},
		}
		json.NewEncoder(w).Encode(addedWorkerModel)
	}
}

func handleCompletion(queries *db.Queries, ctx context.Context) func( http.ResponseWriter,  *http.Request) {
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
