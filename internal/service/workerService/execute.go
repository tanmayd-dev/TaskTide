package workerService

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func Execute(queries *db.Queries, ctx context.Context, _ *minio.Client, task models.TaskExecution, workerID int) {
	taskBaseUUID, err := uuid.ParseBytes([]byte(task.TaskBase))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskBase, err := queries.GetTask(ctx, pgtype.UUID{
		Bytes: taskBaseUUID,
		Valid: true,
	})
	switch models.TaskType(taskBase.TaskType.String) {
	case models.TaskNOOP:
		time.Sleep(time.Second * 10)
		err = queries.MarkTaskComplete(ctx, int32(task.ID))
		if err != nil {
			log.Printf("Could not Free Task:%v\n", err.Error())
			return
		}
		err = queries.MarkWorkerAvailable(ctx, int32(workerID))
		if err != nil {
			log.Printf("Could not Free Task:%v\n", err.Error())
			return
		}
	default:
		log.Printf("Not implemented for %v\n", taskBase.TaskType)
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
