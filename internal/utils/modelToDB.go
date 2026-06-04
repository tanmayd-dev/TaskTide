package utils

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func InputFileDB(inputFileModel models.InputFile) (db.InputFile, error) {
	UUID, err := uuid.Parse(inputFileModel.ID)
	if err != nil {
		return db.InputFile{}, err
	}
	return db.InputFile{
		ID: pgtype.UUID{
			Bytes: UUID,
			Valid: true,
		},
		FileName: pgtype.Text{
			String: inputFileModel.FileName,
			Valid:  true,
		},
	}, nil
}

func TaskDefinitionDB(taskDefinitionDB models.TaskDefinition) (db.TaskDefinition, error) {
	UUID, err := uuid.Parse(taskDefinitionDB.ID)
	if err != nil {
		return db.TaskDefinition{}, err
	}
	WorkflowID, err := uuid.Parse(taskDefinitionDB.WorkflowID)
	if err != nil {
		return db.TaskDefinition{}, err
	}
	return db.TaskDefinition{
		ID: pgtype.UUID{
			Bytes: UUID,
			Valid: true,
		},
		TaskType: pgtype.Text{
			String: string(taskDefinitionDB.TaskType),
			Valid:  true,
		},
		TaskName: pgtype.Text{
			String: taskDefinitionDB.TaskName,
			Valid:  true,
		},
		WorkflowID: pgtype.UUID{
			Bytes: WorkflowID,
			Valid: true,
		},
	}, nil
}

func TaskExecutionDB(taskExecutionDB models.TaskExecution) (db.TaskExecution, error) {
	UUID, err := uuid.Parse(taskExecutionDB.ID)
	if err != nil {
		return db.TaskExecution{}, err
	}
	taskBaseID, err := uuid.Parse(taskExecutionDB.TaskBase)
	if err != nil {
		return db.TaskExecution{}, err
	}
	taskWorkerID, err := uuid.Parse(taskExecutionDB.TaskWorker)
	if err != nil {
		return db.TaskExecution{}, err
	}
	return db.TaskExecution{
		ID: pgtype.UUID{
			Bytes: UUID,
			Valid: true,
		},
		TaskBase: pgtype.UUID{
			Bytes: taskBaseID,
			Valid: true,
		},
		TaskWorker: pgtype.UUID{
			Bytes: taskWorkerID,
			Valid: true,
		},
		TaskAssigned: pgtype.Bool{
			Bool:  taskExecutionDB.TaskAssigned,
			Valid: true,
		},
		TaskCompleted: pgtype.Bool{
			Bool:  taskExecutionDB.TaskCompleted,
			Valid: true,
		},
	}, nil
}

func WorkerDB(workerDB models.Worker) (db.Worker, error) {
	UUID, err := uuid.Parse(workerDB.ID)
	if err != nil {
		return db.Worker{}, err
	}
	return db.Worker{
		ID: pgtype.UUID{
			Bytes: UUID,
			Valid: true,
		},
		WorkerHost: pgtype.Text{
			String: workerDB.WorkerHost,
			Valid:  true,
		},
		WorkerPort: pgtype.Text{
			String: workerDB.WorkerPort,
			Valid:  true,
		},
		WorkerAvailable: pgtype.Bool{
			Bool:  workerDB.WorkerAvailable,
			Valid: true,
		},
	}, nil
}

func WorkflowDefinitionDB(workflowDefinitionDB models.WorkflowDefinition) (db.WorkflowDefinition, error) {
	UUID, err := uuid.Parse(workflowDefinitionDB.ID)
	if err != nil {
		return db.WorkflowDefinition{}, err
	}
	return db.WorkflowDefinition{
		ID: pgtype.UUID{
			Bytes: UUID,
			Valid: true,
		},
		WorkflowName: pgtype.Text{
			String: workflowDefinitionDB.WorkflowName,
			Valid:  true,
		},
	}, nil
}
