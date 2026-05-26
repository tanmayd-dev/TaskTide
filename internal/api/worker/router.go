package worker

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker/health"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker/task"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Route("/", health.Router())
		router.Route("/task", task.Router(queries, ctx, minioClient))
	}
}
