package tasks

import (
	"os"
	"strconv"
	"time"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Output(task *models.WorkerTaskExecution) error {
	output := task.TaskInput + "\nNOOP appended text from output"
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	task.TaskOutput = output
	return nil
}
