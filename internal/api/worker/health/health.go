package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func Router(_ int) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handleHealth)
	}
}
