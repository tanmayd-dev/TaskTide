package workerService

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/tasks"
)

func Execute(task models.WorkerTaskExecution) {
	var taskError error
	switch models.TaskType(task.TaskBase.TaskType) {
	case models.TaskNOOP:
		taskError = tasks.Noop(&task)
	case models.TaskOUTPUT:
		taskError = tasks.Output(&task)
	default:
		taskError = errors.New("not implemented")
		log.Printf("Not implemented for %v\n", task.TaskBase.TaskType)
	}
	success := true
	if taskError != nil {
		success = false
	}
	// else {
	// 	if models.TaskType(taskBase.TaskType) == models.TaskOUTPUT && task.TaskOutput != "" {
	// 		workflowExecutionID, err := utils.StringToUUID(string(task.WorkflowID))
	// 		if err == nil {
	// 			workflowExecution, err := queries.GetWorkflowExecution(ctx, workflowExecutionID)
	// 			if err == nil {
	// 				workflowBaseUUID, err := utils.StringToUUID(workflowExecution.WorkflowBase.String())
	// 				if err == nil {
	// 					taskOutputUUID, err := utils.StringToUUID(string(task.TaskOutput))
	// 					if err == nil {
	// 						err = queries.UpdateWorkflowOutput(ctx, db.UpdateWorkflowOutputParams{
	// 							ID:             workflowBaseUUID,
	// 							WorkflowOutput: taskOutputUUID,
	// 						})
	// 						if err != nil {
	// 							log.Printf("Could not update workflow output: %v\n", err.Error())
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	task.TaskCompleted = true
	task.TaskSucceeded = success
	taskJSON, err := json.Marshal(task)
	if err != nil {
		if err != nil {
			log.Printf("%v\n", err.Error())
			return
		}
	}
	_, err = http.Post(
		fmt.Sprintf("http://%v:%v/worker/completion", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		"application/json",
		bytes.NewReader(taskJSON),
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return
	}
}
