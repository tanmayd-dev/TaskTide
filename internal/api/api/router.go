package api

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/api/api/file"
	"github.com/tanmayd-dev/TaskTide/internal/api/api/health"
	"github.com/tanmayd-dev/TaskTide/internal/api/api/report"
	"github.com/tanmayd-dev/TaskTide/internal/api/api/workflow"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Route("/", health.Router(queries, ctx, minioClient))
		router.Route("/workflow", workflow.Router(queries, ctx, minioClient))
		router.Route("/report", report.Router(queries, ctx, minioClient))
		router.Route("/file", file.Router(queries, ctx, minioClient))
	}
}
