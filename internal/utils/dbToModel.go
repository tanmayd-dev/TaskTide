package utils

import (
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func InputFileModel(inputFileDB db.InputFile) models.InputFile {
	return models.InputFile{
		ID: inputFileDB.ID.String(),
		FileName: inputFileDB.FileName.String,
	}
}

func TaskDefinitionModel(taskDefinitionDB db.TaskDefinition) models.TaskDefinition {
	return models.TaskDefinition{
		ID: taskDefinitionDB.ID.String(),
		TaskType: models.TaskType(taskDefinitionDB.TaskName.String),
		TaskName: taskDefinitionDB.TaskName.String,
		WorkflowID: taskDefinitionDB.WorkflowID.String(),
	}
}

func TaskExecutionModel(taskExecutionDB db.TaskExecution) models.TaskExecution {
	return models.TaskExecution{
		ID: taskExecutionDB.ID.String(),
		TaskBase: taskExecutionDB.TaskBase.String(),
		TaskWorker: taskExecutionDB.TaskWorker.String(),
		TaskAssigned: taskExecutionDB.TaskAssigned.Bool,
		TaskCompleted: taskExecutionDB.TaskCompleted.Bool,
	}
}

func WorkerModel(workerDB db.Worker) models.Worker {
	return models.Worker{
		ID: workerDB.ID.String(),
		WorkerHost: workerDB.WorkerHost.String,
		WorkerPort: workerDB.WorkerPort.String,
		WorkerAvailable: workerDB.WorkerAvailable.Bool,
	}
}

func WorkflowDefinitionModel(workflowDefinitionDB db.WorkflowDefinition) models.WorkflowDefinition {
	return models.WorkflowDefinition{
		ID: workflowDefinitionDB.ID.String(),
		WorkflowName: workflowDefinitionDB.WorkflowName.String,
	}
}
