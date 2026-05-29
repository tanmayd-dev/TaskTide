package queue

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue/queue"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue/worker"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func Router(queries *db.Queries, ctx context.Context) func(chi.Router) {
	return func(r chi.Router) {
		r.Route(
			"/", queue.Router(queries, ctx),
		)
		r.Route(
			"/worker", worker.Router(queries, ctx),
		)
	}
}
