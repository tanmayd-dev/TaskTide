package workerService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func PollForTask() {
	for {
		time.Sleep(time.Second * time.Duration(5))
		response, err := http.Get(fmt.Sprintf("http://%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")))
		if err != nil {
			log.Printf("Failed to make request, Got %v, with error %v.\n", response, err)
			continue
		}
		if response.StatusCode == http.StatusNoContent {
			continue
		}
		var task models.TaskDefinition
		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			log.Printf("Failed to decode %v, with %v.\n", task, err)
			continue
		}
		log.Printf("Got %v.", task)
		taskBytes, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal %v, with %v.\n", taskBytes, err)
			continue
		}
		reportResponse, err := http.Post(fmt.Sprintf("http://%v:%v/report", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewReader(taskBytes))
		if err != nil {
			log.Printf("Failed to report completion %v, with %v\n", reportResponse, err)
			continue
		}
		log.Printf("Reported %#v and got %v.\n", task, reportResponse)
	}
}
