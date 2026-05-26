package queue

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue/queue"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue/worker"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Router(taskQueue *[]models.TaskDefinition, queries *db.Queries, ctx context.Context) func(chi.Router) {
	return func(r chi.Router) {
		r.Route(
			"/", queue.Router(taskQueue),
		)
		r.Route(
			"/worker", worker.Router(queries, ctx),
		)
	}
}
