package queueService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
)

var assignMutex sync.Mutex

func AssignWorker(queries *db.Queries, ctx context.Context, minioClient *minio.Client) {
	assignMutex.Lock()
	defer assignMutex.Unlock()

	worker, err := repositories.GetAvailableWorker(ctx, queries)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	task, err := repositories.GetUnassignedTasks(ctx, queries)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	for {
		assigningTask, err := repositories.AssignWorkerToTask(ctx, queries, worker.ID, task.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		assignedWorker, err := repositories.MarkWorkerUnavailable(ctx, queries, worker.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		object, _, err := repositories.GetFile(ctx, queries, assigningTask.TaskInput, minioClient, os.Getenv("FILES_BUCKET_NAME"))
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		content, err := io.ReadAll(object)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		taskBase, err := repositories.GetTask(ctx, queries, assigningTask.TaskBase)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		log.Printf("Got Task with input content: %s\n", string(content))
		assigningTaskModel := models.WorkerTaskExecution{
			ID:         assigningTask.ID,
			TaskBase:   taskBase,
			TaskWorker: assigningTask.TaskWorker,
			TaskInput:  string(content),
		}
		assignedTaskJSON, err := json.Marshal(assigningTaskModel)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		resp, err := http.Post(
			fmt.Sprintf("http://%v:%v/task", assignedWorker.WorkerHost, assignedWorker.WorkerPort),
			"application/json",
			bytes.NewBuffer(assignedTaskJSON),
		)
		if err != nil || resp.StatusCode != 200 {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = fmt.Sprintf("HTTP status %d", resp.StatusCode)
				resp.Body.Close()
			}
			log.Printf("Failed to dispatch task %v to worker %v (%v:%v): %v\n", task.ID, worker.ID, assignedWorker.WorkerHost, assignedWorker.WorkerPort, errMsg)
			
			unassignErr := repositories.UnassignTask(ctx, queries, assigningTask.ID)
			if unassignErr != nil {
				log.Printf("Failed to unassign task %v: %v\n", assigningTask.ID, unassignErr.Error())
			}
			deleteErr := repositories.DeleteWorker(ctx, queries, worker.ID)
			if deleteErr != nil {
				log.Printf("Failed to delete dead worker %v: %v\n", worker.ID, deleteErr.Error())
			}

			worker, err = repositories.GetAvailableWorker(ctx, queries)
			if errors.Is(err, pgx.ErrNoRows) {
				return
			}
			if err != nil {
				log.Printf("%v\n", err.Error())
				return
			}
			continue
		}
		resp.Body.Close()
		log.Printf("Assigned %v to %v\n", task.ID, worker.ID)
		worker, err = repositories.GetAvailableWorker(ctx, queries)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("workers: %v\n", err.Error())
			return
		}
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		task, err = repositories.GetUnassignedTasks(ctx, queries)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("tasks: %v\n", err.Error())
			return
		}
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
	}
}
