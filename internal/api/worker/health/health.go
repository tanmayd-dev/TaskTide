package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func handle_health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func Health() func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handle_health)
	}
}
