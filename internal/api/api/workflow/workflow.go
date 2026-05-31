package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func handleCreateWorflow(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
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
		workflowUUID, err := uuid.ParseBytes([]byte(taskDef.WorkflowID))
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
			TaskType: string(taskDef.TaskType),
			WorkflowID: pgtype.UUID{
				Bytes: workflowUUID,
				Valid: true,
			},
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

func handleGetWorkflow(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestWorkflow models.WorkflowDefinition
		err := json.NewDecoder(r.Body).Decode(&requestWorkflow)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow", http.StatusBadRequest)
			return
		}
		workflowUUID, err := uuid.ParseBytes([]byte(requestWorkflow.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow id", http.StatusBadRequest)
			return
		}
		workflow, err := queries.GetWorkflow(ctx, pgtype.UUID{
			Bytes: workflowUUID,
			Valid: true,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse ID", http.StatusBadRequest)
			return
		}
		DBTasks, err := queries.GetWorkflowTasks(ctx, pgtype.UUID{
			Bytes: workflowUUID,
			Valid: true,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get tasks", http.StatusBadRequest)
			return
		}
		var tasks []models.TaskDefinition
		for _, task := range DBTasks {
			tasks = append(tasks, models.TaskDefinition{
				ID:         task.ID.String(),
				TaskType:   models.TaskType(task.TaskType),
				TaskName:   task.TaskName.String,
				WorkflowID: task.WorkflowID.String(),
			})
		}
		fatWorkflow := models.FatWorkflowDefinition{
			ID:            workflow.ID.String(),
			WorflowName:   workflow.WorkflowName.String,
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

func handleStart(queries *db.Queries, ctx context.Context, _ *minio.Client) func(http.ResponseWriter, *http.Request) {
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
		log.Printf("%#v\n", workflowDef)
		workflowUUID, err := uuid.ParseBytes([]byte(workflowDef.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse workflow id", http.StatusBadRequest)
			return
		}
		tasks, err := queries.GetWorkflowTasks(ctx, pgtype.UUID{
			Bytes: workflowUUID,
			Valid: true,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not get tasks for workflow", http.StatusBadRequest)
			return
		}
		var resps []string
		for _, task := range tasks {
			jsonTask, err := json.Marshal(task)
			if err != nil {
				http.Error(w, "Could not parse Task.", http.StatusBadRequest)
				return
			}
			log.Printf("sending: %#v\n", task)
			url := fmt.Sprintf("http://%v:%v/", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
			resp, err := http.Post(
				url,
				"application/json",
				bytes.NewBuffer(jsonTask),
			)
			if err != nil {
				http.Error(w, fmt.Sprintf("Could not request Task from %v. Got %v", url, err), http.StatusInternalServerError)
				return
			}
			respString, err := json.Marshal(resp.Body)
			if err != nil {
				log.Printf("%v\n", err.Error())
				http.Error(w, "Could not parse form", http.StatusBadRequest)
				return
			}
			resps = append(resps, string(respString))
		}
		respsJSON, err := json.Marshal(resps)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not create response", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "%#v", respsJSON)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handleGetWorkflow(queries, ctx, minioClient))
		router.Post("/", handleStart(queries, ctx, minioClient))
		router.Post("/create", handleCreateWorflow(queries, ctx, minioClient))
		router.Post("/add", handleAddTask(queries, ctx, minioClient))
	}
}
