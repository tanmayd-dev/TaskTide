package tasks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func Output(queries *db.Queries, ctx context.Context, minioClient *minio.Client, task models.TaskExecution, taskAttempt models.TaskAttempt) error {
	var content []byte
	if taskAttempt.TaskInput != "" {
		taskInputUUID, err := utils.StringToUUID(string(taskAttempt.TaskInput))
		if err != nil {
			log.Printf("%v\n", err.Error())
			return err
		}
		object, _, err := repositories.GetFile(ctx, queries, db.InputFile{
			ID: taskInputUUID,
		}, minioClient, os.Getenv("FILES_BUCKET_NAME"))
		if err != nil {
			log.Printf("%v\n", err.Error())
			return err
		}
		content, err = io.ReadAll(object)
		if err != nil {
			log.Printf("%v\n", err.Error())
			return err
		}
		log.Printf("Got OUTPUT task with input content: %s\n", string(content))
	}
	if task.TaskOutput == "" {
		return fmt.Errorf("missing task output for OUTPUT task %s", task.TaskBase)
	}
	taskOutputUUID, err := utils.StringToUUID(string(task.TaskOutput))
	if err != nil {
		log.Printf("%v\n", err.Error())
		return err
	}
	outputRecord, err := queries.GetFile(ctx, taskOutputUUID)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return err
	}
	outputContent := append(content, []byte("\nOUTPUT task completed\n")...)
	_, err = minioClient.PutObject(
		ctx,
		os.Getenv("FILES_BUCKET_NAME"),
		fmt.Sprintf("%v/%v", taskOutputUUID, outputRecord.FileName),
		bytes.NewReader(outputContent),
		int64(len(outputContent)),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return err
	}
	seconds, err := strconv.Atoi(os.Getenv("NOOP_TIME"))
	if err != nil {
		return err
	}
	time.Sleep(time.Second * time.Duration(seconds))
	return nil
}
