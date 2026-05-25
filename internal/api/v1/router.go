package v1

import (
	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/api/v1/health"
	"github.com/tanmayd-dev/TaskTide/internal/api/v1/report"
	"github.com/tanmayd-dev/TaskTide/internal/api/v1/workflow"
)

func Router() func(chi.Router) {
	return func(router chi.Router) {
		router.Route("/", health.Health())
		router.Route("/workflow", workflow.Route())
		router.Route("/report", report.Router(queries, ctx, minioClient))
		router.Route("/file", file.Router(queries, ctx, minioClient))
	}
}