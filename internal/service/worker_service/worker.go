package worker_service

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
			log.Default().Printf("Failed to make request, Got %v, with error %v.\n", response, err)
			continue
		}
		if response.StatusCode == http.StatusNoContent {
			continue
		}
		var task models.TaskDefinition
		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			log.Default().Printf("Failed to decode %v, with %v.\n", task, err)
			continue
		}
		log.Default().Printf("Got %v.", task)
		task_bytes, err := json.Marshal(task)
		if err != nil {
			log.Default().Printf("Failed to marshal %v, with %v.\n", task_bytes, err)
			continue
		}
		report_response, err := http.Post(fmt.Sprintf("http://%v:%v/report", os.Getenv("API_HOST"), os.Getenv("API_PORT")), "application/json", bytes.NewReader(task_bytes))
		if err != nil {
			log.Default().Printf("Failed to report completion %v, with %v\n", report_response, err)
			continue
		}
		log.Default().Printf("Reported %#v and got %v.\n", task, report_response)
	}
}
