package models

import (
	"encoding/json"
	"fmt"
)

type TaskType string
type TaskDefinitionID string
type WorkflowDefinitionID string
type FileID string
type WorkerID string
type InputFileID string
type TaskExecutionID string

const (
	TaskNOOP              TaskType = "NOOP"
	TaskDocRead           TaskType = "doc.read"
	TaskDocCheckSensetive TaskType = "doc.check_sensetive"
	TaskCSVRead           TaskType = "csv.read"
	TaskCSVValidate       TaskType = "csv.validate"
	TaskCSVAggregate      TaskType = "csv.aggregate"
	TaskCSVFilter         TaskType = "csv.filter"
)

var validTaskTypes = map[TaskType]struct{}{
	TaskNOOP:              {},
	TaskDocRead:           {},
	TaskDocCheckSensetive: {},
	TaskCSVRead:           {},
	TaskCSVValidate:       {},
	TaskCSVAggregate:      {},
	TaskCSVFilter:         {},
}

func (taskType *TaskType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	taskTypeS := TaskType(s)
	if _, ok := validTaskTypes[taskTypeS]; !ok {
		return fmt.Errorf("Invalid task type %v.", s)
	}
	*taskType = taskTypeS
	return nil
}

type InputFile struct {
	ID       InputFileID `json:"id,omitempty"`
	FileName string
}

type TaskDefinition struct {
	ID         TaskDefinitionID `json:"id,omitempty"`
	TaskType   TaskType
	TaskName   string
	WorkflowID WorkflowDefinitionID `json:"workflow_id,omitempty"`
}

type TaskExecution struct {
	ID            TaskExecutionID  `json:"id,omitempty"`
	TaskBase      TaskDefinitionID `json:"task_base,omitempty"`
	TaskWorker    WorkerID         `json:"task_worker,omitempty"`
	TaskAssigned  bool
	TaskCompleted bool
}

type Worker struct {
	ID              WorkerID `json:"id,omitempty"`
	WorkerHost      string
	WorkerPort      string
	WorkerAvailable bool
}

type WorkflowDefinition struct {
	ID           WorkflowDefinitionID `json:"id,omitempty"`
	WorkflowName string
}

type FatWorkflowDefinition struct {
	ID            WorkflowDefinitionID `json:"id,omitempty"`
	WorflowName   string
	WorkflowTasks []TaskDefinition
}

type FileDependency struct {
	ID           TaskDefinitionID   `json:"id,omitempty"`
	Dependencies []TaskDefinitionID `json:"dependencies,omitempty"`
}
