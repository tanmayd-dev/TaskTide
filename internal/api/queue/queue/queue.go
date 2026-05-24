package queue

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func enqueue(task_queue *[]models.TaskDefinition) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var new_task models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&new_task)
		if err != nil {
			http.Error(w, "Could not parse task", http.StatusBadRequest)
		}
		*task_queue = append(*task_queue, new_task)
	}
}

func dequeue(task_queue *[]models.TaskDefinition) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(*task_queue) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		new_task := (*task_queue)[0]
		*task_queue = (*task_queue)[1:]
		err := json.NewEncoder(w).Encode(new_task)
		if err != nil {
			http.Error(w, "Could not encode Task", http.StatusInternalServerError)
		}
	}
}

func Router(task_queue *[]models.TaskDefinition) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", dequeue(task_queue))
		r.Post("/", enqueue(task_queue))
	}
}
