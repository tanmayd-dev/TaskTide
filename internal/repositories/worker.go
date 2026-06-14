package repositories

import (
	"context"

	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func AddWorker(ctx context.Context, queries *db.Queries, worker models.Worker) (db.Worker, error) {
	addedWorker, err := queries.AddWorker(ctx, db.AddWorkerParams{
		WorkerHost: worker.WorkerHost,
		WorkerPort: worker.WorkerPort,
	})
	return addedWorker, err
}
