package workerService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func RegisterSelf(port int) (models.WorkerID, error) {
	worker := models.Worker{
		WorkerHost: "localhost",
		WorkerPort: fmt.Sprintf("%v", port),
	}
	workerJSON, err := json.Marshal(worker)
	registration, err := http.Post(
		fmt.Sprintf("http://%v:%v/worker", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader(workerJSON),
	)
	if err != nil {
		log.Printf("Registeration erred: %v\n", err.Error())
		return models.WorkerID{}, err
	}
	if registration.StatusCode != 200 {
		log.Printf("Registeration failed: %v\n", registration.StatusCode)
		return models.WorkerID{}, err
	}
	log.Printf("Added self, got %#v", registration)
	var registeredWorker models.Worker
	json.NewDecoder(registration.Body).Decode(&registeredWorker)
	return registeredWorker.ID, nil
}
