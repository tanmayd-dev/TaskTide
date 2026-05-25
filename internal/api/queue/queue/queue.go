package queue

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func enqueue(taskQueue *[]models.TaskDefinition) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var newTask models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&newTask)
		if err != nil {
			http.Error(w, "Could not parse task", http.StatusBadRequest)
		}
		*taskQueue = append(*taskQueue, newTask)
	}
}

func dequeue(taskQueue *[]models.TaskDefinition) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(*taskQueue) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		newTask := (*taskQueue)[0]
		*taskQueue = (*taskQueue)[1:]
		err := json.NewEncoder(w).Encode(newTask)
		if err != nil {
			http.Error(w, "Could not encode Task", http.StatusInternalServerError)
		}
	}
}

func Router(taskQueue *[]models.TaskDefinition) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", dequeue(taskQueue))
		r.Post("/", enqueue(taskQueue))
	}
}
