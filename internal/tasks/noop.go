package tasks

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Noop(task *models.WorkerTaskExecution) error {
	log.Printf("Got Task with input content: %s\n", task.TaskInput)
	output := task.TaskInput + "\nNOOP appended text from noop"
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	task.TaskOutput = output
	return nil
}
