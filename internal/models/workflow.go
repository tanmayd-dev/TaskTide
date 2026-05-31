package models

import (
	"encoding/json"
	"fmt"
)

type TaskType string
type TaskDefinitionID string
type WorkflowDefinitionID string
type FileID string

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
	ID       string
	FileName string
}

type TaskDefinition struct {
	ID         string
	TaskType   TaskType
	TaskName   string
	WorkflowID string
}

type TaskExecution struct {
	ID            string
	TaskBase      string
	TaskWorker    string
	TaskAssigned  bool
	TaskCompleted bool
}

type Worker struct {
	ID              string
	WorkerHost      string
	WorkerPort      string
	WorkerAvailable bool
}

type WorkflowDefinition struct {
	ID           string
	WorkflowName string
}

type FatWorkflowDefinition struct {
	ID            string
	WorflowName   string
	WorkflowTasks []TaskDefinition
}
