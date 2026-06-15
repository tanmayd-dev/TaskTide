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

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func AssignWorker(queries *db.Queries, ctx context.Context, minioClient *minio.Client) {
	worker, err := queries.GetAvailableWorker(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	task, err := queries.GetUnassignedTasks(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	for {
		assigningTask, err := queries.AssignWorkerToTask(ctx, db.AssignWorkerToTaskParams{
			TaskWorker: worker.ID,
			ID:         task.ID,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		assignedWorker, err := queries.MarkWorkerUnavailable(ctx, worker.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		object, _, err := repositories.GetFile(
			ctx,
			queries,
			models.InputFileID(assigningTask.TaskInput.String()),
			minioClient,
			os.Getenv("FILES_BUCKET_NAME"),
		)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		content, err := io.ReadAll(object)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		taskBase, err := queries.GetTask(ctx, assigningTask.TaskBase)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		taskBaseModel := utils.TaskDefinitionModel(taskBase)
		log.Printf("Got Task with input content: %s\n", string(content))
		assigningTaskModel := models.WorkerTaskExecution{
			ID:         models.TaskExecutionID(assigningTask.ID.String()),
			TaskBase:   taskBaseModel,
			TaskWorker: models.WorkerID(assigningTask.TaskWorker.String()),
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
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		log.Printf("got %#v\n", resp)
		log.Printf("Assigned %v to %v\n", task.ID, worker.ID)
		worker, err = queries.GetAvailableWorker(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("workers: %v\n", err.Error())
			return
		}
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		task, err = queries.GetUnassignedTasks(ctx)
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
