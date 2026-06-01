package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func Router(_ models.WorkerID) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handleHealth)
	}
}
