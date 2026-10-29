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

// failedTasksCap bounds the test-only failedTasks map so it cannot grow
// without limit (production-safety bounding of test-only state).
const failedTasksCap = 10000

func Noop(task *models.WorkerTaskExecution) error {
	log.Printf("Got Task with input content: %s\n", task.TaskInput)

	// Test-only failure simulators (E2E tests); never enabled in production.
	if os.Getenv("ENABLE_TEST_SIMULATORS") == "true" {
		// Simulate failures for E2E tests
		if task.TaskInput == "FAIL" {
			return errors.New("simulated permanent failure")
		}

		if task.TaskInput == "FAIL_ONCE" {
			failedMu.Lock()
			alreadyFailed := failedTasks[string(task.ID)]
			if !alreadyFailed {
				if len(failedTasks) >= failedTasksCap {
					failedTasks = make(map[string]bool)
				}
				failedTasks[string(task.ID)] = true
				failedMu.Unlock()
				log.Printf("Simulating transient failure for task %s\n", task.ID)
				return errors.New("simulated transient failure")
			}
			failedMu.Unlock()
			log.Printf("Transient failure already simulated for task %s; succeeding on retry\n", task.ID)
		}
	}

	output := task.TaskInput + "\nNOOP appended text from noop"
	// Default to 0 when NOOP_TIME is unset or empty.
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
