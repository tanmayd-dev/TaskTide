package report

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func getCompletion(_ *db.Queries, _ context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task db.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Printf("Could not decode %v. Got %v\n", r.Body, err)
			return
		}
		log.Printf("Completed %#v\n", task)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", getCompletion(queries, ctx, minioClient))
	}
}
