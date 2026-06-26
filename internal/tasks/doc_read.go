package tasks

import (
	"log"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func DocRead(task *models.WorkerTaskExecution) error {
	log.Printf("[DocRead] Reading text document task execution: %s\n", task.ID)
	task.TaskOutput = task.TaskInput
	return nil
}
