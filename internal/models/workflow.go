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

type TaskDefinition struct {
	ID   TaskDefinitionID `json:"id"`
	Type TaskType         `json:"type"`
}

type WorkflowDefinition struct {
	ID    WorkflowDefinitionID `json:"id"`
	Tasks []TaskDefinition     `json:"tasks"`
}

type File struct {
	ID   FileID `json:"id"`
	Name string `json:"name"`
}
