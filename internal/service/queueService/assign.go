package queueService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func AssignWorker(queries *db.Queries, ctx context.Context) {
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
			TaskWorker: pgtype.Int4{
				Int32: worker.ID,
				Valid: true,
			},
			ID: task.ID,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		assignedWorker, err := queries.MarkWorkerUnavailable(ctx, int32(worker.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		assigningTaskModel := models.TaskExecution {
			ID: int(assigningTask.ID),
			TaskBase: assigningTask.TaskBase.String(),
			TaskWorker: int(assigningTask.TaskWorker.Int32),
			TaskAssigned: assigningTask.TaskAssigned.Bool,
			TaskCompleted: assigningTask.TaskCompleted.Bool,
		}
		assignedTaskJSON, err := json.Marshal(assigningTaskModel)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
		resp, err := http.Post(
			fmt.Sprintf("http://%v:%v/task", assignedWorker.WorkerHost.String, assignedWorker.WorkerPort.String),
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
