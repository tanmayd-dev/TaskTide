package worker

import (
	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker/health"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker/task"
)

func Router() func(chi.Router) {
	return func(router chi.Router) {
		router.Route("/", health.Router())
		router.Route("/task", task.Router())
	}
}
