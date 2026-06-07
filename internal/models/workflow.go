package models

import (
	"encoding/json"
	"fmt"
)

type TaskType string
type TaskDefinitionID string
type WorkflowDefinitionID string
type WorkerID string
type InputFileID string
type TaskExecutionID string
type WorkflowExecutionID string
type TaskAttemptID string

const (
	TaskNOOP              TaskType = "NOOP"
	TaskDocRead           TaskType = "doc.read"
	TaskDocCheckSensitive TaskType = "doc.check_sensitive"
	TaskCSVRead           TaskType = "csv.read"
	TaskCSVValidate       TaskType = "csv.validate"
	TaskCSVAggregate      TaskType = "csv.aggregate"
	TaskCSVFilter         TaskType = "csv.filter"
)

var validTaskTypes = map[TaskType]struct{}{
	TaskNOOP:              {},
	TaskDocRead:           {},
	TaskDocCheckSensitive: {},
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
		return fmt.Errorf("invalid task type %v", s)
	}
	*taskType = taskTypeS
	return nil
}

type InputFile struct {
	ID         InputFileID          `json:"id,omitempty"`
	FileName   string               `json:"file_name,omitempty"`
	WorkflowID WorkflowDefinitionID `json:"workflow_id,omitempty"`
}

type TaskDefinition struct {
	ID                  TaskDefinitionID     `json:"id,omitempty"`
	TaskType            TaskType             `json:"task_type,omitempty"`
	TaskName            string               `json:"task_name,omitempty"`
	WorkflowID          WorkflowDefinitionID `json:"workflow_id,omitempty"`
	TaskInput           InputFileID          `json:"task_input,omitempty"`
	TaskInputTask       TaskDefinitionID     `json:"task_input_task,omitempty"`
	TaskOutput          InputFileID          `json:"task_output,omitempty"`
	WorkflowExecutionID WorkflowExecutionID  `json:"workflow_execution_id,omitempty"`
}

type TaskExecution struct {
	ID            TaskExecutionID     `json:"id,omitempty"`
	TaskBase      TaskDefinitionID    `json:"task_base,omitempty"`
	TaskWorker    WorkerID            `json:"task_worker,omitempty"`
	TaskAssigned  bool                `json:"task_assigned,omitempty"`
	TaskCompleted bool                `json:"task_completed,omitempty"`
	TaskInput     InputFileID         `json:"task_input,omitempty"`
	WorkflowID    WorkflowExecutionID `json:"workflow_id,omitempty"`
}

type Worker struct {
	ID              WorkerID `json:"id,omitempty"`
	WorkerHost      string   `json:"worker_host,omitempty"`
	WorkerPort      string   `json:"worker_port,omitempty"`
	WorkerAvailable bool     `json:"worker_available,omitempty"`
}

type WorkflowDefinition struct {
	ID           WorkflowDefinitionID `json:"id,omitempty"`
	WorkflowName string               `json:"workflow_name,omitempty"`
}

type WorkflowExecution struct {
	ID           WorkflowExecutionID  `json:"id,omitempty"`
	WorkflowBase WorkflowDefinitionID `json:"workflow_base,omitempty"`
}

type FatWorkflowDefinition struct {
	ID            WorkflowDefinitionID `json:"id,omitempty"`
	WorkflowName  string               `json:"workflow_name,omitempty"`
	WorkflowTasks []TaskDefinition     `json:"workflow_tasks,omitempty"`
}

type TaskDependency struct {
	ID                  TaskDefinitionID   `json:"id,omitempty"`
	Dependencies        []TaskDefinitionID `json:"dependencies,omitempty"`
	InputFromDependency TaskDefinitionID   `json:"input_from_dependency,omitempty"`
}

type TaskAttempt struct {
	ID                TaskAttemptID
	TaskExecutionBase TaskExecutionID
	TaskCompleted     bool
	TaskSucceeded     bool
	TaskInput         InputFileID
}
