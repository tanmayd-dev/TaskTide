package queue

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
)

func enqueue(queries *db.Queries, ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse task", http.StatusBadRequest)
			return
		}
		taskUUID, err := uuid.Parse(task.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		_, err = queries.CreateTaskExecution(ctx, db.CreateTaskExecutionParams{
			TaskBase: pgtype.UUID{
				Bytes: taskUUID,
				Valid: true,
			},
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not create task Execution", http.StatusBadRequest)
			return
		}

		go queueService.AssignWorker(queries, ctx)
	}
}

func poll(queries *db.Queries, ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task db.TaskExecution
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse task", http.StatusBadRequest)
			return
		}
		go queueService.AssignWorker(queries, ctx)
	}
}

func Router(queries *db.Queries, ctx context.Context) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", poll(queries, ctx))
		r.Post("/", enqueue(queries, ctx))
	}
}
