package apiservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func StartWorkflow(queries *db.Queries, ctx context.Context, _ *minio.Client, workflowID models.WorkflowDefinitionID) error {
	workflowUUID, err := utils.StringToUUID(string(workflowID))
	if err != nil {
		return err
	}
	taskDBs, err := queries.GetWorkflowTasks(ctx, workflowUUID)
	if err != nil {
		return err
	}
	dependencies := make(map[models.TaskDefinitionID]map[models.TaskDefinitionID]struct{})
	dependants := make(map[models.TaskDefinitionID][]models.TaskDefinitionID)
	taskIDToModel := make(map[models.TaskDefinitionID]models.TaskDefinition)
	for _, taskDB := range taskDBs {
		dependencies[models.TaskDefinitionID(taskDB.ID.String())] = make(map[models.TaskDefinitionID]struct{})
		dependencyTaskDBs, err := queries.GetTaskDependencies(ctx, taskDB.ID)
		if err != nil {
			return err
		}
		taskModel := models.TaskDefinitionID(taskDB.ID.String())
		for _, dependencyTaskDB := range dependencyTaskDBs {
			dependencyModel := models.TaskDefinitionID(dependencyTaskDB.TaskDependsOn.String())
			dependencies[taskModel][dependencyModel] = struct{}{}
			dependants[dependencyModel] = append(dependants[dependencyModel], taskModel)
		}
		taskIDToModel[taskModel] = utils.TaskDefinitionModel(taskDB)
	}
	got := true
	for got {
		got = false
		for taskID, deps := range dependencies {
			if len(deps) > 0 {
				continue
			}
			got = true
			delete(dependencies, taskID)
			for _, dependant := range dependants[taskID] {
				delete(dependencies[dependant], taskID)
			}
			delete(dependants, taskID)
			task := taskIDToModel[taskID]
			jsonTask, err := json.Marshal(task)
			if err != nil {
				return err
			}
			log.Printf("sending: %#v\n", task)
			url := fmt.Sprintf("http://%v:%v/", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
			resp, err := http.Post(
				url,
				"application/json",
				bytes.NewBuffer(jsonTask),
			)
			if err != nil || (200 <= resp.StatusCode && resp.StatusCode <= 300) {
				return err
			}
		}
	}
	return nil
}
