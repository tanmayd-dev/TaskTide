package tasks

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/consts"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func Noop(queries *db.Queries, ctx context.Context, minioClient *minio.Client, taskAttempt models.TaskAttempt) error {
	taskInputUUID, err := utils.StringToUUID(string(taskAttempt.ID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return err
	}
	if taskAttempt.TaskInput != "" {
		object, _, err := repositories.GetFile(ctx, queries, db.InputFile{
			ID: taskInputUUID,
		}, minioClient, os.Getenv("FILES_BUCKET_NAME"))
		if err != nil {
			log.Printf("%v\n", err.Error())
			return err
		}
		bytes := make([]byte, consts.MB*5)
		_, err = object.Read(bytes)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return err
		}
		log.Printf("Got Task with input %#v\n", bytes)
	}
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	return nil
}
