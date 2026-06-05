package workerService

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func Execute(queries *db.Queries, ctx context.Context, _ *minio.Client, task models.TaskExecution, workerID models.WorkerID) {
	taskBaseUUID, err := utils.StringToUUID(string(task.TaskBase))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	taskUUID, err := utils.StringToUUID(string(task.ID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	queries.CreateTaskAttempt(ctx, taskUUID)
	taskBase, err := queries.GetTask(ctx, taskBaseUUID)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
	var taskError error
	switch models.TaskType(taskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop()
	default:
		log.Printf("Not implemented for %v\n", taskBase.TaskType)
	}
	err = queries.MarkTaskComplete(ctx, taskUUID)
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	err = queries.MarkWorkerAvailable(ctx, workerUUID)
	if err != nil {
		log.Printf("Could not Free Task:%v\n", err.Error())
		return
	}
	if taskError != nil {
		queries.FailTask(ctx, taskUUID)
	} else {
		queries.SucceedTask(ctx, taskUUID)
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
