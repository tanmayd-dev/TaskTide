package task

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
)

func runTask(queries *db.Queries, ctx context.Context, minioClient *minio.Client, workerID int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.TaskExecution
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse form", http.StatusBadRequest)
			return
		}
		go workerService.Execute(queries, ctx, minioClient, task, workerID)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client, workerID int) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", runTask(queries, ctx, minioClient, workerID))
	}
}
