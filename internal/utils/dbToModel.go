package utils

import (
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func InputFileModel(inputFileDB db.InputFile) models.InputFile {
	return models.InputFile{
		ID:       models.InputFileID(inputFileDB.ID.String()),
		FileName: inputFileDB.FileName,
	}
}

func TaskDefinitionModel(taskDefinitionDB db.TaskDefinition) models.TaskDefinition {
	taskInput := models.InputFileID("")
	if taskDefinitionDB.TaskInputFile.Valid {
		taskInput = models.InputFileID(taskDefinitionDB.TaskInputFile.String())
	}
	taskInputTask := models.TaskDefinitionID("")
	if taskDefinitionDB.TaskInputTask.Valid {
		taskInputTask = models.TaskDefinitionID(taskDefinitionDB.TaskInputTask.String())
	}
	taskOutput := models.InputFileID("")
	if taskDefinitionDB.TaskOutput.Valid {
		taskOutput = models.InputFileID(taskDefinitionDB.TaskOutput.String())
	}
	return models.TaskDefinition{
		ID:            models.TaskDefinitionID(taskDefinitionDB.ID.String()),
		TaskType:      models.TaskType(taskDefinitionDB.TaskType),
		TaskName:      taskDefinitionDB.TaskName.String,
		WorkflowID:    models.WorkflowDefinitionID(taskDefinitionDB.WorkflowID.String()),
		TaskInput:     taskInput,
		TaskInputTask: taskInputTask,
		TaskOutput:    taskOutput,
	}
}

func TaskExecutionModel(taskExecutionDB db.TaskExecution) models.TaskExecution {
	taskInput := models.InputFileID("")
	if taskExecutionDB.TaskInput.Valid {
		taskInput = models.InputFileID(taskExecutionDB.TaskInput.String())
	}
	taskOutput := models.InputFileID("")
	if taskExecutionDB.TaskOutput.Valid {
		taskOutput = models.InputFileID(taskExecutionDB.TaskOutput.String())
	}
	return models.TaskExecution{
		ID:            models.TaskExecutionID(taskExecutionDB.ID.String()),
		TaskBase:      models.TaskDefinitionID(taskExecutionDB.TaskBase.String()),
		TaskWorker:    models.WorkerID(taskExecutionDB.TaskWorker.String()),
		TaskAssigned:  taskExecutionDB.TaskAssigned,
		TaskCompleted: taskExecutionDB.TaskCompleted,
		TaskInput:     taskInput,
		TaskOutput:    taskOutput,
	}
}

func WorkerModel(workerDB db.Worker) models.Worker {
	return models.Worker{
		ID:              models.WorkerID(workerDB.ID.String()),
		WorkerHost:      workerDB.WorkerHost,
		WorkerPort:      workerDB.WorkerPort,
		WorkerAvailable: workerDB.WorkerAvailable,
	}
}

func WorkflowDefinitionModel(workflowDefinitionDB db.WorkflowDefinition) models.WorkflowDefinition {
	return models.WorkflowDefinition{
		ID:           models.WorkflowDefinitionID(workflowDefinitionDB.ID.String()),
		WorkflowName: workflowDefinitionDB.WorkflowName.String,
	}
}

func TaskAttemptModel(taskAttemptDB db.TaskAttempt) models.TaskAttempt {
	taskInput := models.InputFileID("")
	if taskAttemptDB.TaskInput.Valid {
		taskInput = models.InputFileID(taskAttemptDB.TaskInput.String())
	}
	return models.TaskAttempt{
		ID:                models.TaskAttemptID(taskAttemptDB.ID.String()),
		TaskExecutionBase: models.TaskExecutionID(taskAttemptDB.TaskExecutionBase.String()),
		TaskCompleted:     taskAttemptDB.TaskCompleted,
		TaskSucceeded:     taskAttemptDB.TaskSucceeded,
		TaskInput:         taskInput,
	}
}
