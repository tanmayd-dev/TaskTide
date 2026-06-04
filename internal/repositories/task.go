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

func AddTaskDependency(queries *db.Queries, ctx context.Context, minioClient *minio.Client, task models.TaskDefinitionID, dependencies []models.TaskDefinitionID) error {
	taskUUID, err := utils.StringToUUID(string(task))
	if err != nil {
		return errors.New("Could not parse task")
	}
	for _, dependency := range dependencies {
		dependencyUUID, err := utils.StringToUUID(string(dependency))
		if err != nil {
			return fmt.Errorf("Could not parse dependency: %#v", dependency)
		}
		queries.AddTaskDependency(ctx, db.AddTaskDependencyParams{
			TaskID:        taskUUID,
			TaskDependsOn: dependencyUUID,
		})
	}
	return nil
}
