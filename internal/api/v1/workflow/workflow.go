package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func handle_start(w http.ResponseWriter, r *http.Request) {
	var task models.TaskDefinition
	err := json.NewDecoder(r.Body).Decode(&task)
		if err != nil {
		json_task, err := json.Marshal(task)
		if err != nil {
			http.Error(w, "Could not parse Task.", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Could not parse Task. Format is: %s", json_task), http.StatusBadRequest)
	}
	json_task, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "Could not parse Task.", http.StatusBadRequest)
		return
	}
	url := fmt.Sprintf("http://%v:%v/", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(json_task),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not request Task from %v. Got %v", url, err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("Got response: %v", resp)))
}

func Route() func(chi.Router) {
	return func(router chi.Router) {
		router.Post("/", handle_start)
	}
}