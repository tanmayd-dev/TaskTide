package utils

import (
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func InputFileModel(inputFileDB db.InputFile) models.InputFile {
	return models.InputFile{
		ID: models.InputFileID(inputFileDB.ID.String()),
		FileName: inputFileDB.FileName,
	}
}

func TaskDefinitionModel(taskDefinitionDB db.TaskDefinition) models.TaskDefinition {
	return models.TaskDefinition{
		ID: models.TaskDefinitionID(taskDefinitionDB.ID.String()),
		TaskType: models.TaskType(taskDefinitionDB.TaskName.String),
		TaskName: taskDefinitionDB.TaskName.String,
		WorkflowID: models.WorkflowDefinitionID(taskDefinitionDB.WorkflowID.String()),
	}
}

func TaskExecutionModel(taskExecutionDB db.TaskExecution) models.TaskExecution {
	return models.TaskExecution{
		ID: models.TaskExecutionID(taskExecutionDB.ID.String()),
		TaskBase: models.TaskDefinitionID(taskExecutionDB.TaskBase.String()),
		TaskWorker: models.WorkerID(taskExecutionDB.TaskWorker.String()),
		TaskAssigned: taskExecutionDB.TaskAssigned,
		TaskCompleted: taskExecutionDB.TaskCompleted,
	}
}

func WorkerModel(workerDB db.Worker) models.Worker {
	return models.Worker{
		ID: models.WorkerID(workerDB.ID.String()),
		WorkerHost: workerDB.WorkerHost,
		WorkerPort: workerDB.WorkerPort,
		WorkerAvailable: workerDB.WorkerAvailable,
	}
}

func WorkflowDefinitionModel(workflowDefinitionDB db.WorkflowDefinition) models.WorkflowDefinition {
	return models.WorkflowDefinition{
		ID: models.WorkflowDefinitionID(workflowDefinitionDB.ID.String()),
		WorkflowName: workflowDefinitionDB.WorkflowName.String,
	}
}
