package tasks

import (
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

var (
	failedTasks = make(map[string]bool)
	failedMu    sync.Mutex
)

func Noop(task *models.WorkerTaskExecution) error {
	log.Printf("Got Task with input content: %s\n", task.TaskInput)

	// Simulate failures for E2E tests
	if task.TaskInput == "FAIL" {
		return errors.New("simulated permanent failure")
	}

	if task.TaskInput == "FAIL_ONCE" {
		failedMu.Lock()
		alreadyFailed := failedTasks[string(task.ID)]
		if !alreadyFailed {
			failedTasks[string(task.ID)] = true
			failedMu.Unlock()
			log.Printf("Simulating transient failure for task %s\n", task.ID)
			return errors.New("simulated transient failure")
		}
		failedMu.Unlock()
		log.Printf("Transient failure already simulated for task %s; succeeding on retry\n", task.ID)
	}

	output := task.TaskInput + "\nNOOP appended text from noop"
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	task.TaskOutput = output
	return nil
}
