package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	apiservice "github.com/tanmayd-dev/TaskTide/internal/service/apiService"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func handleCreateWorkflow(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var workflowDef models.WorkflowDefinition
		err := json.NewDecoder(r.Body).Decode(&workflowDef)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow", http.StatusBadRequest)
			return
		}
		createdWorkflow, err := queries.CreateWorkflow(ctx, pgtype.Text{
			String: workflowDef.WorkflowName,
			Valid:  true,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not add workflow to DB", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(createdWorkflow)
	}
}

func handleAddTask(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var taskDef models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&taskDef)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow", http.StatusBadRequest)
			return
		}
		workflowUUID, err := utils.StringToUUID(string(taskDef.WorkflowID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow id", http.StatusBadRequest)
			return
		}
		createdTask, err := queries.CreateTask(ctx, db.CreateTaskParams{
			TaskName: pgtype.Text{
				String: taskDef.TaskName,
				Valid:  true,
			},
			TaskType:   string(taskDef.TaskType),
			WorkflowID: workflowUUID,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not add task to db", http.StatusBadRequest)
			return
		}
		err = json.NewEncoder(w).Encode(createdTask)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not return task", http.StatusBadRequest)
			return
		}
	}
}

func addTaskDependency(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.TaskDependency
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse request", http.StatusBadRequest)
			return
		}
		err = repositories.AddTaskDependency(queries, ctx, minioClient, request.ID, request.Dependencies)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not Add task dependency", http.StatusBadRequest)
			return
		}
	}
}

func handleGetWorkflow(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestWorkflow models.WorkflowDefinition
		err := json.NewDecoder(r.Body).Decode(&requestWorkflow)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow", http.StatusBadRequest)
			return
		}
		workflowUUID, err := utils.StringToUUID(string(requestWorkflow.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow id", http.StatusBadRequest)
			return
		}
		workflow, err := queries.GetWorkflow(ctx, workflowUUID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		DBTasks, err := queries.GetWorkflowTasks(ctx, workflowUUID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get tasks", http.StatusBadRequest)
			return
		}
		var tasks []models.TaskDefinition
		for _, task := range DBTasks {
			tasks = append(tasks, models.TaskDefinition{
				ID:         models.TaskDefinitionID(task.ID.String()),
				TaskType:   models.TaskType(task.TaskType),
				TaskName:   task.TaskName.String,
				WorkflowID: models.WorkflowDefinitionID(task.WorkflowID.String()),
			})
		}
		fatWorkflow := models.FatWorkflowDefinition{
			ID:            models.WorkflowDefinitionID(workflow.ID.String()),
			WorkflowName:  workflow.WorkflowName.String,
			WorkflowTasks: tasks,
		}
		err = json.NewEncoder(w).Encode(fatWorkflow)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not return workflow", http.StatusBadRequest)
			return
		}
	}
}

func handleStart(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var workflowDef models.WorkflowDefinition
		err := json.NewDecoder(r.Body).Decode(&workflowDef)
		if err != nil {
			jsonTask, err := json.Marshal(workflowDef)
			if err != nil {
				http.Error(w, "Could not parse Task.", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Could not parse Task. Format is: %s", jsonTask), http.StatusBadRequest)
			return
		}
		go apiservice.StartWorkflow(queries, ctx, minioClient, workflowDef.ID)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handleGetWorkflow(queries, ctx, minioClient))
		router.Post("/", handleStart(queries, ctx, minioClient))
		router.Post("/create", handleCreateWorkflow(queries, ctx, minioClient))
		router.Post("/add", handleAddTask(queries, ctx, minioClient))
		router.Post("/dependency", addTaskDependency(queries, ctx, minioClient))
	}
}
