package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func handleStart(_ *db.Queries, _ context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			jsonTask, err := json.Marshal(task)
			if err != nil {
				http.Error(w, "Could not parse Task.", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Could not parse Task. Format is: %s", jsonTask), http.StatusBadRequest)
		}
		jsonTask, err := json.Marshal(task)
		if err != nil {
			http.Error(w, "Could not parse Task.", http.StatusBadRequest)
			return
		}
		url := fmt.Sprintf("http://%v:%v/", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
		resp, err := http.Post(
			url,
			"application/json",
			bytes.NewBuffer(jsonTask),
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not request Task from %v. Got %v", url, err), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf("Got response: %v", resp)))
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Post("/", handleStart(queries, ctx, minioClient))
	}
}
