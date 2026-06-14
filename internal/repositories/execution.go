package repositories

import (
	"context"

	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func CreateWorkflowExecution(ctx context.Context, queries *db.Queries, workflowID models.WorkflowDefinitionID) (models.WorkflowExecution, error) {
	workflowUUID, err := utils.StringToUUID(string(workflowID))
	if err != nil {
		return models.WorkflowExecution{}, err
	}
	we, err := queries.CreateWorkflowExecution(ctx, workflowUUID)
	if err != nil {
		return models.WorkflowExecution{}, err
	}
	return models.WorkflowExecution{
		ID:           models.WorkflowExecutionID(we.ID.String()),
		WorkflowBase: models.WorkflowDefinitionID(we.WorkflowBase.String()),
	}, nil
}

func CreateTaskExecution(ctx context.Context, queries *db.Queries, taskID models.TaskDefinitionID, inputID models.InputFileID, outputID models.InputFileID, workflowExecutionID models.WorkflowExecutionID) (models.TaskExecution, error) {
	taskUUID, err := utils.StringToUUID(string(taskID))
	if err != nil {
		return models.TaskExecution{}, err
	}
	inputUUID, err := utils.StringToUUID(string(inputID))
	if err != nil {
		return models.TaskExecution{}, err
	}
	outputUUID, err := utils.StringToUUID(string(outputID))
	if err != nil {
		return models.TaskExecution{}, err
	}
	weUUID, err := utils.StringToUUID(string(workflowExecutionID))
	if err != nil {
		return models.TaskExecution{}, err
	}

	te, err := queries.CreateTaskExecution(ctx, db.CreateTaskExecutionParams{
		TaskBase:   taskUUID,
		TaskInput:  inputUUID,
		TaskOutput: outputUUID,
		WorkflowID: weUUID,
	})
	if err != nil {
		return models.TaskExecution{}, err
	}
	return utils.TaskExecutionModel(te), nil
}

func GetUnassignedTasks(ctx context.Context, queries *db.Queries) (models.TaskExecution, error) {
	te, err := queries.GetUnassignedTasks(ctx)
	if err != nil {
		return models.TaskExecution{}, err
	}
	// Convert GetUnassignedTasksRow to db.TaskExecution to use TaskExecutionModel helper
	teDB := db.TaskExecution{
		ID:            te.ID,
		TaskBase:      te.TaskBase,
		TaskWorker:    te.TaskWorker,
		TaskAssigned:  te.TaskAssigned,
		TaskCompleted: te.TaskCompleted,
		TaskInput:     te.TaskInput,
		TaskOutput:    te.TaskOutput,
		WorkflowID:    te.WorkflowID,
	}
	return utils.TaskExecutionModel(teDB), nil
}

func GetWorkflowTaskExecutions(ctx context.Context, queries *db.Queries, workflowExecutionID models.WorkflowExecutionID) ([]models.TaskExecution, error) {
	weUUID, err := utils.StringToUUID(string(workflowExecutionID))
	if err != nil {
		return nil, err
	}
	tesDB, err := queries.GetWorkflowTaskExecutions(ctx, weUUID)
	if err != nil {
		return nil, err
	}
	var tes []models.TaskExecution
	for _, te := range tesDB {
		tes = append(tes, utils.TaskExecutionModel(te))
	}
	return tes, nil
}


func MarkTaskComplete(ctx context.Context, queries *db.Queries, id models.TaskExecutionID) error {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return err
	}
	return queries.MarkTaskComplete(ctx, uuid)
}

func GetTaskExecution(ctx context.Context, queries *db.Queries, id models.TaskExecutionID) (models.TaskExecution, error) {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return models.TaskExecution{}, err
	}
	te, err := queries.GetTaskExecution(ctx, uuid)
	if err != nil {
		return models.TaskExecution{}, err
	}
	return utils.TaskExecutionModel(te), nil
}

func UnassignTask(ctx context.Context, queries *db.Queries, id models.TaskExecutionID) error {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return err
	}
	return queries.UnassignTask(ctx, uuid)
}

