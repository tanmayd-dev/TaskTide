package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func CreateWorkflow(ctx context.Context, queries *db.Queries, name string) (models.WorkflowDefinition, error) {
	wd, err := queries.CreateWorkflow(ctx, pgtype.Text{String: name, Valid: true})
	if err != nil {
		return models.WorkflowDefinition{}, err
	}
	return utils.WorkflowDefinitionModel(wd), nil
}

func GetWorkflow(ctx context.Context, queries *db.Queries, id models.WorkflowDefinitionID) (models.WorkflowDefinition, error) {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return models.WorkflowDefinition{}, err
	}
	wd, err := queries.GetWorkflow(ctx, uuid)
	if err != nil {
		return models.WorkflowDefinition{}, err
	}
	return utils.WorkflowDefinitionModel(wd), nil
}

func ListWorkflows(ctx context.Context, queries *db.Queries) ([]models.WorkflowDefinition, error) {
	wds, err := queries.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	var workflows []models.WorkflowDefinition
	for _, wd := range wds {
		workflows = append(workflows, utils.WorkflowDefinitionModel(wd))
	}
	return workflows, nil
}

func CreateTask(ctx context.Context, queries *db.Queries, task models.TaskDefinition) (models.TaskDefinition, error) {
	workflowUUID, err := utils.StringToUUID(string(task.WorkflowID))
	if err != nil {
		return models.TaskDefinition{}, err
	}
	var taskInputFile pgtype.UUID
	var taskInputTask pgtype.UUID
	if task.TaskInput != "" {
		taskInputFile, err = utils.StringToUUID(string(task.TaskInput))
		if err != nil {
			return models.TaskDefinition{}, err
		}
	}
	if task.TaskInputTask != "" {
		taskInputTask, err = utils.StringToUUID(string(task.TaskInputTask))
		if err != nil {
			return models.TaskDefinition{}, err
		}
	}
	td, err := queries.CreateTask(ctx, db.CreateTaskParams{
		TaskName:      pgtype.Text{String: task.TaskName, Valid: true},
		TaskType:      string(task.TaskType),
		WorkflowID:    workflowUUID,
		TaskInputFile: taskInputFile,
		TaskInputTask: taskInputTask,
	})
	if err != nil {
		return models.TaskDefinition{}, err
	}
	createdTask := utils.TaskDefinitionModel(td)

	// Automatically record dependency if taskInputTask is set
	if task.TaskInputTask != "" {
		err = AddTaskDependency(ctx, queries, createdTask.ID, []models.TaskDefinitionID{task.TaskInputTask}, "")
		if err != nil {
			return models.TaskDefinition{}, err
		}
	}
	return createdTask, nil
}

func GetWorkflowTasks(ctx context.Context, queries *db.Queries, workflowID models.WorkflowDefinitionID) ([]models.TaskDefinition, error) {
	uuid, err := utils.StringToUUID(string(workflowID))
	if err != nil {
		return nil, err
	}
	tasksDB, err := queries.GetWorkflowTasks(ctx, uuid)
	if err != nil {
		return nil, err
	}
	var tasks []models.TaskDefinition
	for _, t := range tasksDB {
		tasks = append(tasks, utils.TaskDefinitionModel(t))
	}
	return tasks, nil
}

func GetTaskDependencies(ctx context.Context, queries *db.Queries, taskID models.TaskDefinitionID) ([]models.TaskDefinitionID, error) {
	uuid, err := utils.StringToUUID(string(taskID))
	if err != nil {
		return nil, err
	}
	depsDB, err := queries.GetTaskDependencies(ctx, uuid)
	if err != nil {
		return nil, err
	}
	var deps []models.TaskDefinitionID
	for _, d := range depsDB {
		deps = append(deps, models.TaskDefinitionID(d.TaskDependsOn.String()))
	}
	return deps, nil
}

func UpdateTaskOutput(ctx context.Context, queries *db.Queries, taskID models.TaskDefinitionID, outputFileID models.InputFileID) error {
	taskUUID, err := utils.StringToUUID(string(taskID))
	if err != nil {
		return err
	}
	fileUUID, err := utils.StringToUUID(string(outputFileID))
	if err != nil {
		return err
	}
	return queries.UpdateTaskOutput(ctx, db.UpdateTaskOutputParams{
		ID:         taskUUID,
		TaskOutput: fileUUID,
	})
}

func AddTaskDependency(ctx context.Context, queries *db.Queries, task models.TaskDefinitionID, dependencies []models.TaskDefinitionID, inputFromDependency models.TaskDefinitionID) error {
	taskUUID, err := utils.StringToUUID(string(task))
	if err != nil {
		return errors.New("could not parse task")
	}
	for _, dependency := range dependencies {
		dependencyUUID, err := utils.StringToUUID(string(dependency))
		if err != nil {
			return fmt.Errorf("could not parse dependency: %#v", dependency)
		}
		_, err = queries.AddTaskDependency(ctx, db.AddTaskDependencyParams{
			TaskID:        taskUUID,
			TaskDependsOn: dependencyUUID,
		})
		if err != nil {
			return err
		}
	}
	if inputFromDependency != "" {
		inputDependencyUUID, err := utils.StringToUUID(string(inputFromDependency))
		if err != nil {
			return fmt.Errorf("could not parse input dependency: %#v", inputFromDependency)
		}
		_, err = queries.AddTaskDependency(ctx, db.AddTaskDependencyParams{
			TaskID:        taskUUID,
			TaskDependsOn: inputDependencyUUID,
		})
		if err != nil {
			return err
		}
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

func GetTask(ctx context.Context, queries *db.Queries, id models.TaskDefinitionID) (models.TaskDefinition, error) {
	uuid, err := utils.StringToUUID(string(id))
	if err != nil {
		return models.TaskDefinition{}, err
	}
	t, err := queries.GetTask(ctx, uuid)
	if err != nil {
		return models.TaskDefinition{}, err
	}
	return utils.TaskDefinitionModel(t), nil
}


