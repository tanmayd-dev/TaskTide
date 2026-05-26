package worker

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func registerWorker(queries *db.Queries, ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		registeredWorker, err := queries.AddWorker(ctx, db.AddWorkerParams{
			WorkerHost: pgtype.Text{
				String: r.Host,
				Valid:  true,
			},
			WorkerPort: pgtype.Text{
				String: r.URL.Port(),
				Valid:  true,
			},
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not register worker to DB", http.StatusBadRequest)
			return
		}
		log.Printf("%#v\n", registeredWorker)
	}
}

func Router(queries *db.Queries, ctx context.Context) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", registerWorker(queries, ctx))
	}
}
