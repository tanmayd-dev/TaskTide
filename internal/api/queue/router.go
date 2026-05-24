package queue

import (
	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue/queue"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Router(task_queue *[]models.TaskDefinition) func(chi.Router) {
	return func(r chi.Router) {
		r.Route(
			"/", queue.Router(task_queue),
		)
	}
}