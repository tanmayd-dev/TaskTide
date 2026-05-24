package models

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

type TaskDefinition struct {
	ID TaskDefinitionID
	Type TaskType
}

type WorkflowDefinition struct {
	ID WorkflowDefinitionID
	Tasks []TaskDefinition
}
