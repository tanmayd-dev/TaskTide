package tasks

import (
	"os"
	"strconv"
	"time"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Output(task *models.WorkerTaskExecution) error {
	output := task.TaskInput + "\nNOOP appended text from output"
	// Default to 0 when NOOP_TIME is unset or empty (mirrors Noop), so the
	// OUTPUT task does not fail in environments without NOOP_TIME.
	seconds := 0
	if value := os.Getenv("NOOP_TIME"); value != "" {
		var err error
		seconds, err = strconv.Atoi(value)
		if err != nil {
			return err
		}
	}
	time.Sleep(time.Second * time.Duration(seconds))
	task.TaskOutput = output
	return nil
}
