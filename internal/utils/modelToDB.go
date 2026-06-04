package utils

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func InputFileDB(inputFileModel models.InputFile) (db.InputFile, error) {
	UUID, err := StringToUUID(string(inputFileModel.ID))
	if err != nil {
		return db.InputFile{}, err
	}
	return db.InputFile{
		ID:       UUID,
		FileName: inputFileModel.FileName,
	}, nil
}

func TaskDefinitionDB(taskDefinitionDB models.TaskDefinition) (db.TaskDefinition, error) {
	UUID, err := StringToUUID(string(taskDefinitionDB.ID))
	if err != nil {
		return db.TaskDefinition{}, err
	}
	WorkflowID, err := StringToUUID(string(taskDefinitionDB.WorkflowID))
	if err != nil {
		return db.TaskDefinition{}, err
	}
	return db.TaskDefinition{
		ID:       UUID,
		TaskType: string(taskDefinitionDB.TaskType),
		TaskName: pgtype.Text{
			String: taskDefinitionDB.TaskName,
			Valid:  true,
		},
		WorkflowID: WorkflowID,
	}, nil
}

func TaskExecutionDB(taskExecutionDB models.TaskExecution) (db.TaskExecution, error) {
	UUID, err := StringToUUID(string(taskExecutionDB.ID))
	if err != nil {
		return db.TaskExecution{}, err
	}
	taskBaseID, err := StringToUUID(string(taskExecutionDB.TaskBase))
	if err != nil {
		return db.TaskExecution{}, err
	}
	taskWorkerID, err := StringToUUID(string(taskExecutionDB.TaskWorker))
	if err != nil {
		return db.TaskExecution{}, err
	}
	return db.TaskExecution{
		ID:            UUID,
		TaskBase:      taskBaseID,
		TaskWorker:    taskWorkerID,
		TaskAssigned:  taskExecutionDB.TaskAssigned,
		TaskCompleted: taskExecutionDB.TaskCompleted,
	}, nil
}

func WorkerDB(workerDB models.Worker) (db.Worker, error) {
	UUID, err := StringToUUID(string(workerDB.ID))
	if err != nil {
		return db.Worker{}, err
	}
	return db.Worker{
		ID:              UUID,
		WorkerHost:      workerDB.WorkerHost,
		WorkerPort:      workerDB.WorkerPort,
		WorkerAvailable: workerDB.WorkerAvailable,
	}, nil
}

func WorkflowDefinitionDB(workflowDefinitionDB models.WorkflowDefinition) (db.WorkflowDefinition, error) {
	UUID, err := StringToUUID(string(workflowDefinitionDB.ID))
	if err != nil {
		return db.WorkflowDefinition{}, err
	}
	return db.WorkflowDefinition{
		ID: UUID,
		WorkflowName: pgtype.Text{
			String: workflowDefinitionDB.WorkflowName,
			Valid:  true,
		},
	}, nil
}
