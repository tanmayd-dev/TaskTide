package task

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func runTask(queries *db.Queries, ctx context.Context, workerID models.WorkerID) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		workerTaskExecution, err := utils.ReadRequest[models.WorkerTaskExecution](w, r)
		if utils.HasError(w, r, "could not read request", err) {
			return
		}
		go workerService.Execute(workerTaskExecution)
	}
}

func Router(queries *db.Queries, ctx context.Context, workerID models.WorkerID) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", runTask(queries, ctx, workerID))
	}
}
