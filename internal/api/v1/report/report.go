package report

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func get_completion() func( http.ResponseWriter,  *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
			log.Default().Printf("Could not decode %v. Got %v\n", r.Body, err)
			return
		}
		log.Default().Printf("Completed %#v\n", task)
	}
}

func Router() func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", get_completion())
	}
}