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

func RegisterSelf(port int) error {
	worker := models.Worker{
		WorkerHost: "localhost",
		WorkerPort: fmt.Sprintf("%v", port),
	}
	workerJSON, err := json.Marshal(worker)
	if err != nil {
		log.Printf("Worker marshal erred: %v\n", err.Error())
		return err
	}
	log.Printf("%#v\n", string(workerJSON))
	registration, err := http.Post(
		fmt.Sprintf("http://%v:%v/worker", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader(workerJSON),
	)
	if err != nil {
		log.Printf("registration erred: %v\n", err.Error())
		return err
	}
	if registration.StatusCode != 200 {
		log.Printf("registration failed: %v\n", registration.StatusCode)
		return err
	}
	log.Printf("Added self, got %#v", registration)
	var registeredWorker models.Worker
	json.NewDecoder(registration.Body).Decode(&registeredWorker)
	return nil
}
