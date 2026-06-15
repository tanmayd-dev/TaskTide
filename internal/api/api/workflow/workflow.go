package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
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
		createdWorkflow, err := repositories.CreateWorkflow(ctx, queries, workflowDef.WorkflowName)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not add workflow to DB", http.StatusBadRequest)
			return
		}
		utils.WriteJSON(w, http.StatusOK, createdWorkflow)
	}
}

func handleAddTask(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var taskDef models.TaskDefinition
		err := json.NewDecoder(r.Body).Decode(&taskDef)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse task", http.StatusBadRequest)
			return
		}
		createdTask, err := repositories.CreateTask(ctx, queries, taskDef)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not add task to db", http.StatusBadRequest)
			return
		}
		utils.WriteJSON(w, http.StatusOK, createdTask)
	}
}

func addTaskDependency(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.TaskDependency
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse request", http.StatusBadRequest)
			return
		}
		err = repositories.AddTaskDependency(ctx, queries, request.ID, request.Dependencies, request.InputFromDependency)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not Add task dependency", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
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
		workflow, err := repositories.GetWorkflow(ctx, queries, requestWorkflow.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		tasks, err := repositories.GetWorkflowTasks(ctx, queries, requestWorkflow.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get tasks", http.StatusBadRequest)
			return
		}
		fatWorkflow := models.FatWorkflowDefinition{
			ID:            workflow.ID,
			WorkflowName:  workflow.WorkflowName,
			WorkflowTasks: tasks,
		}
		utils.WriteJSON(w, http.StatusOK, fatWorkflow)
	}
}

func handleStart(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var workflowDef models.WorkflowDefinition
		err := json.NewDecoder(r.Body).Decode(&workflowDef)
		if err != nil {
			jsonTask, jsonErr := json.Marshal(workflowDef)
			if jsonErr != nil {
				http.Error(w, "Could not parse Task.", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Could not parse Task. Format is: %s", jsonTask), http.StatusBadRequest)
			return
		}

		executionID, err := apiservice.StartWorkflow(queries, ctx, minioClient, workflowDef.ID)
		if err != nil {
			log.Printf("failed to start workflow: %v", err)
			http.Error(w, "Could not start workflow", http.StatusInternalServerError)
			return
		}

		utils.WriteJSON(w, http.StatusOK, models.WorkflowExecution{ID: executionID})
	}
}

func handleListWorkflows(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		workflows, err := repositories.ListWorkflows(ctx, queries)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not list workflows", http.StatusInternalServerError)
			return
		}
		if workflows == nil {
			workflows = []models.WorkflowDefinition{}
		}
		utils.WriteJSON(w, http.StatusOK, workflows)
	}
}

func handleGetWorkflowByID(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		workflow, err := repositories.GetWorkflow(ctx, queries, models.WorkflowDefinitionID(idStr))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		tasks, err := repositories.GetWorkflowTasks(ctx, queries, models.WorkflowDefinitionID(idStr))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get tasks", http.StatusBadRequest)
			return
		}
		fatWorkflow := models.FatWorkflowDefinition{
			ID:            workflow.ID,
			WorkflowName:  workflow.WorkflowName,
			WorkflowTasks: tasks,
		}
		utils.WriteJSON(w, http.StatusOK, fatWorkflow)
	}
}

func handleGetWorkflowStatus(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestExecution models.WorkflowExecution
		err := json.NewDecoder(r.Body).Decode(&requestExecution)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow execution", http.StatusBadRequest)
			return
		}
		tasks, err := repositories.GetWorkflowTaskExecutions(ctx, queries, requestExecution.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get task executions", http.StatusBadRequest)
			return
		}
		if tasks == nil {
			tasks = []models.TaskExecution{}
		}
		utils.WriteJSON(w, http.StatusOK, tasks)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handleGetWorkflow(queries, ctx, minioClient))
		router.Get("/list", handleListWorkflows(queries, ctx, minioClient))
		router.Get("/{id}", handleGetWorkflowByID(queries, ctx, minioClient))
		router.Post("/", handleStart(queries, ctx, minioClient))
		router.Post("/create", handleCreateWorkflow(queries, ctx, minioClient))
		router.Post("/add", handleAddTask(queries, ctx, minioClient))
		router.Post("/dependency", addTaskDependency(queries, ctx, minioClient))
		router.Post("/status", handleGetWorkflowStatus(queries, ctx, minioClient))
	}
}
