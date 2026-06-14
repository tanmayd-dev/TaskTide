package repositories

import (
	"context"
	"log"
	"net/http"

	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func CreateTaskAttempt(ctx context.Context, completion models.WorkerTaskExecution, w http.ResponseWriter, queries *db.Queries) (error) {
	taskExecutionUUID, err := utils.StringToUUID(string(completion.ID))
	if err != nil {
		log.Printf("%v\n", err.Error())
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return err
	}
	taskExecutionDB, err := queries.GetTaskExecution(ctx, taskExecutionUUID)
	_, err = queries.CreateTaskAttempt(ctx, db.CreateTaskAttemptParams{
		TaskExecutionBase: taskExecutionUUID,
		TaskCompleted:     completion.TaskCompleted,
		TaskSucceeded:     completion.TaskSucceeded,
		TaskInput:         taskExecutionDB.TaskInput,
	})
	return err
}
