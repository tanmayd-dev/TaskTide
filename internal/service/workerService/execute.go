package workerService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
)

func Execute(task models.WorkerTaskExecution) {
	var taskError error
	switch models.TaskType(task.TaskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop(&task)
	case models.TaskOUTPUT:
		taskError = tasks.Output(&task)
	default:
		taskError = errors.New("not implemented")
		log.Printf("Not implemented for %v\n", task.TaskBase.TaskType)
	}
	success := taskError == nil
	task.TaskCompleted = true
	task.TaskSucceeded = success
	taskJSON, err := json.Marshal(task)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	_, err = http.Post(
		fmt.Sprintf("http://%v:%v/worker/completion", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader(taskJSON),
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
}
