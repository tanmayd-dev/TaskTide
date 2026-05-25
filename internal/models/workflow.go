package models

import (
	"encoding/json"
	"fmt"
)

type TaskType string
type TaskDefinitionID string
type WorkflowDefinitionID string

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

func (task_type *TaskType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	task_type_s := TaskType(s)
	if _, ok := validTaskTypes[task_type_s]; !ok {
		return fmt.Errorf("Invalid task type %v.", s)
	}
	*task_type = task_type_s
	return nil
}

type TaskDefinition struct {
	ID   TaskDefinitionID `json:"id"`
	Type TaskType `json:"type"`
}

type WorkflowDefinition struct {
	ID    WorkflowDefinitionID `json:"id"`
	Tasks []TaskDefinition `json:"tasks"`
}
