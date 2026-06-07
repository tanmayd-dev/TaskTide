package queueService

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/testutil"
)

func TestAssignWorker_AssignsSingleTask(t *testing.T) {
	taskReceived := make(chan models.TaskExecution, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/task" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var task models.TaskExecution
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		taskReceived <- task
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverHostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(serverHostPort, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected test server URL: %s", server.URL)
	}

	workerID := uuid.New()
	taskID := uuid.New()
	taskBaseID := uuid.New()
	workflowID := uuid.New()

	calls := struct {
		available bool
		task      bool
	}{}

	mockDB := &testutil.MockDB{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			switch {
			case strings.Contains(sql, "SELECT id, worker_host, worker_port, worker_available FROM worker"):
				if !calls.available {
					calls.available = true
					return testutil.RowFromValues(
						pgtype.UUID{Bytes: workerID, Valid: true},
						parts[0],
						parts[1],
						true,
					)
				}
				return testutil.RowFromError(pgx.ErrNoRows)
			case strings.Contains(sql, "SELECT id, task_base, task_worker, task_assigned, task_completed, task_input, task_output, workflow_id FROM task_execution A"):
				if !calls.task {
					calls.task = true
					return testutil.RowFromValues(
						pgtype.UUID{Bytes: taskID, Valid: true},
						pgtype.UUID{Bytes: taskBaseID, Valid: true},
						pgtype.UUID{Valid: false},
						false,
						false,
						pgtype.UUID{Valid: false},
						pgtype.UUID{Valid: false},
						pgtype.UUID{Bytes: workflowID, Valid: true},
					)
				}
				return testutil.RowFromError(pgx.ErrNoRows)
			case strings.Contains(sql, "UPDATE task_execution"):
				return testutil.RowFromValues(
					pgtype.UUID{Bytes: taskID, Valid: true},
					pgtype.UUID{Bytes: taskBaseID, Valid: true},
					pgtype.UUID{Bytes: workerID, Valid: true},
					true,
					false,
					pgtype.UUID{Valid: false},
					pgtype.UUID{Valid: false},
					pgtype.UUID{Bytes: workflowID, Valid: true},
				)
			case strings.Contains(sql, "UPDATE worker"):
				return testutil.RowFromValues(
					pgtype.UUID{Bytes: workerID, Valid: true},
					parts[0],
					parts[1],
					false,
				)
			default:
				return testutil.RowFromError(pgx.ErrNoRows)
			}
		},
	}

	queries := db.New(mockDB)
	AssignWorker(queries, context.Background())

	select {
	case task := <-taskReceived:
		if task.ID != models.TaskExecutionID(taskID.String()) {
			t.Fatalf("unexpected task id: got %s, want %s", task.ID, taskID.String())
		}
		if task.TaskWorker != models.WorkerID(workerID.String()) {
			t.Fatalf("unexpected task worker: got %s, want %s", task.TaskWorker, workerID.String())
		}
		if !task.TaskAssigned {
			t.Fatal("expected task to be assigned")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for task dispatch")
	}
}
