package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func AddTaskDependency(queries *db.Queries, ctx context.Context, minioClient *minio.Client, task models.TaskDefinitionID, dependencies []models.TaskDefinitionID, inputFromDependency models.TaskDefinitionID) error {
	taskUUID, err := utils.StringToUUID(string(task))
	if err != nil {
		return errors.New("could not parse task")
	}
	for _, dependency := range dependencies {
		dependencyUUID, err := utils.StringToUUID(string(dependency))
		if err != nil {
			return fmt.Errorf("could not parse dependency: %#v", dependency)
		}
		queries.AddTaskDependency(ctx, db.AddTaskDependencyParams{
			TaskID:        taskUUID,
			TaskDependsOn: dependencyUUID,
		})
	}
	if inputFromDependency != "" {
		inputDependencyUUID, err := utils.StringToUUID(string(inputFromDependency))
		if err != nil {
			return fmt.Errorf("could not parse input dependency: %#v", inputFromDependency)
		}
		queries.AddTaskDependency(ctx, db.AddTaskDependencyParams{
			TaskID:        taskUUID,
			TaskDependsOn: inputDependencyUUID,
		})
		err = queries.UpdateTaskInputTask(ctx, db.UpdateTaskInputTaskParams{
			ID:            taskUUID,
			TaskInputTask: inputDependencyUUID,
		})
		if err != nil {
			return fmt.Errorf("could not update task input task: %w", err)
		}
	}
	return nil
}
