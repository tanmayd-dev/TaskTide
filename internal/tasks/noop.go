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

var attemptMap sync.Map

func Noop(task *models.WorkerTaskExecution) error {
	log.Printf("Got Task with input content: %s\n", task.TaskInput)
	
	if task.TaskInput == "FAIL" {
		return errors.New("simulated permanent failure")
	}
	
	if task.TaskInput == "FAIL_ONCE" {
		val, _ := attemptMap.LoadOrStore(string(task.ID), 0)
		attempts := val.(int)
		if attempts == 0 {
			attemptMap.Store(string(task.ID), 1)
			return errors.New("simulated transient failure (first attempt)")
		}
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

