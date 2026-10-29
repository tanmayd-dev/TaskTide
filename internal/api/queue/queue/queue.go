package queue

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

// Router no longer serves the HTTP push-model enqueue/poll handlers: the
// queue is pull-model over gRPC (SubmitJob/PollJobs) and ready workflow
// tasks are claimed by the coordinator sweep. It is kept mountable with an
// empty route set for cmd/queue/main.go.
func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {}
}
