package workerService

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
)

func Execute(queries *db.Queries, ctx context.Context, _ *minio.Client, task models.TaskExecution, workerUUID models.WorkerID) {
	queries.CreateTaskAttempt(ctx, pgtype.UUID{
		Bytes: task.ID.Bytes,
		Valid: task.ID.Valid,
	})
	taskBase, err := queries.GetTask(ctx, pgtype.UUID{
		Bytes: task.TaskBase.Bytes,
		Valid: task.TaskBase.Valid,
	})
	var taskError error
	switch models.TaskType(taskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop()
	default:
		log.Printf("Not implemented for %v\n", taskBase.TaskType)
	}
	err = queries.MarkTaskComplete(ctx, pgtype.UUID{
		Bytes: task.ID.Bytes,
		Valid: task.ID.Valid,
	})
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	err = queries.MarkWorkerAvailable(ctx, pgtype.UUID{
		Bytes: workerUUID.Bytes,
		Valid: workerUUID.Valid,
	})
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	if taskError != nil {
		queries.FailTask(ctx, pgtype.UUID{
			Bytes: task.ID.Bytes,
			Valid: task.ID.Valid,
		})
	} else {
		queries.SucceedTask(ctx, pgtype.UUID{
			Bytes: task.ID.Bytes,
			Valid: task.ID.Valid,
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
