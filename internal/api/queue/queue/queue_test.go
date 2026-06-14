package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/testutil"
)

func TestEnqueue_ReturnsAccepted(t *testing.T) {
	workflowID := uuid.New()
	mockDB := &testutil.MockDB{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			switch {
			case strings.Contains(sql, "INSERT INTO workflow_execution"):
				return testutil.RowFromValues(
					pgtype.UUID{Bytes: uuid.New(), Valid: true},
					pgtype.UUID{Bytes: workflowID, Valid: true},
				)
			case strings.Contains(sql, "INSERT INTO task_execution"):
				return testutil.RowFromValues(
					pgtype.UUID{Bytes: uuid.New(), Valid: true},
					args[0].(pgtype.UUID),
					pgtype.UUID{Valid: false},
					false,
					false,
					args[1].(pgtype.UUID),
					pgtype.UUID{Valid: false},
					args[2].(pgtype.UUID),
				)
			default:
				return testutil.RowFromError(pgx.ErrNoRows)
			}
		},
	}

	queries := db.New(mockDB)
	payload := models.TaskDefinition{
		ID:         models.TaskDefinitionID(uuid.New().String()),
		TaskType:   models.TaskNOOP,
		WorkflowID: models.WorkflowDefinitionID(workflowID.String()),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler := enqueue(queries, context.Background())
	handler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}
}

func TestPoll_ReturnsNoContent(t *testing.T) {
	queries := db.New(&testutil.MockDB{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler := poll(queries, context.Background())
	handler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}
