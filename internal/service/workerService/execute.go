package workerService

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
)

func Execute(queries *db.Queries, ctx context.Context, _ *minio.Client, task models.TaskExecution, workerID models.WorkerID) {
	taskBaseUUID, err := uuid.ParseBytes([]byte(task.TaskBase))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskUUID, err := uuid.ParseBytes([]byte(task.ID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	workerUUID, err := uuid.ParseBytes([]byte(workerID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	queries.CreateTaskAttempt(ctx, pgtype.UUID{
		Bytes: taskUUID,
		Valid: true,
	})
	taskBase, err := queries.GetTask(ctx, pgtype.UUID{
		Bytes: taskBaseUUID,
		Valid: true,
	})
	var taskError error
	switch models.TaskType(taskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop()
	default:
		log.Printf("Not implemented for %v\n", taskBase.TaskType)
	}
	err = queries.MarkTaskComplete(ctx, pgtype.UUID{
		Bytes: taskUUID,
		Valid: true,
	})
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	err = queries.MarkWorkerAvailable(ctx, pgtype.UUID{
		Bytes: workerUUID,
		Valid: true,
	})
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	if taskError != nil {
		queries.FailTask(ctx, pgtype.UUID{
			Bytes: taskUUID,
			Valid: true,
		})
	} else {
		queries.SucceedTask(ctx, pgtype.UUID{
			Bytes: taskUUID,
			Valid: true,
		})
	}
	_, err = http.Post(
		fmt.Sprintf("http://%v:%v/worker/completion", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader([]byte{}),
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
}
