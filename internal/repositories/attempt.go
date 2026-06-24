package repositories

import (
	"context"

	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func CreateTaskAttempt(ctx context.Context, queries *db.Queries, completion models.WorkerTaskExecution) error {
	taskExecutionUUID, err := utils.StringToUUID(string(completion.ID))
	if err != nil {
		return err
	}
	taskExecutionDB, err := queries.GetTaskExecution(ctx, taskExecutionUUID)
	if err != nil {
		return err
	}
	_, err = queries.CreateTaskAttempt(ctx, db.CreateTaskAttemptParams{
		TaskExecutionBase: taskExecutionUUID,
		TaskCompleted:     completion.TaskCompleted,
		TaskSucceeded:     completion.TaskSucceeded,
		TaskInput:         taskExecutionDB.TaskInput,
	})
	return err
}

func GetTaskAttemptsCount(ctx context.Context, queries *db.Queries, id models.TaskExecutionID) (int64, error) {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return 0, err
	}
	return queries.GetTaskAttemptsCount(ctx, uuid)
}


