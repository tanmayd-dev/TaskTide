package queueService

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/observability"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/testutil"
	"go.opentelemetry.io/otel"
)

// SubmitJob traces through the package-level tracer, which is nil until the
// service initializes it; point it at a tracer from the default provider so
// unit tests stay hermetic.
func init() {
	if observability.Tracer == nil {
		observability.Tracer = otel.Tracer("queueService-test")
	}
}

// idempotencyRecorder routes SubmitJob's sqlc calls through the MockDB
// without a real database.
type idempotencyRecorder struct {
	createCalls   int
	createErr     error
	createResult  db.Job
	createID      pgtype.UUID
	existingJobs  map[string]db.Job
	getKeyCalls   int
	getKeyErr     error
	queriedKey    pgtype.Text
	ensureQueues  []string
	lastKey       pgtype.Text
	maxRetriesLog []int32
}

func newIdempotencyRecorder() *idempotencyRecorder {
	r := &idempotencyRecorder{existingJobs: make(map[string]db.Job)}
	r.createID = testUUID(1)
	return r
}

func (r *idempotencyRecorder) queries() *db.Queries {
	mock := &testutil.MockDB{
		QueryRowFunc: r.queryRow,
		ExecFunc:     r.exec,
	}
	return db.New(mock)
}

func (r *idempotencyRecorder) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "INSERT INTO jobs") && strings.Contains(sql, "ON CONFLICT (idempotency_key) DO NOTHING"): // CreateJobWithIdempotency
		r.createCalls++
		r.lastKey = args[6].(pgtype.Text)
		r.maxRetriesLog = append(r.maxRetriesLog, args[2].(int32))
		if r.createErr != nil {
			return testutil.RowFromError(r.createErr)
		}
		return testutil.RowFromValues(jobRowValues(r.createResult)...)
	case strings.Contains(sql, "FROM jobs WHERE idempotency_key = $1"): // GetJobByIdempotencyKey
		r.getKeyCalls++
		r.queriedKey = args[0].(pgtype.Text)
		if r.getKeyErr != nil {
			return testutil.RowFromError(r.getKeyErr)
		}
		job, ok := r.existingJobs[r.queriedKey.String]
		if !ok {
			return testutil.RowFromError(pgx.ErrNoRows)
		}
		return testutil.RowFromValues(jobRowValues(job)...)
	}
	return testutil.RowFromError(pgx.ErrNoRows)
}

func (r *idempotencyRecorder) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO queues") { // EnsureQueueExists
		r.ensureQueues = append(r.ensureQueues, args[0].(string))
	}
	return pgconn.CommandTag{}, nil
}

func submitRequest(queue, payload, idempotencyKey string, maxRetries int32) *pb.SubmitRequest {
	return &pb.SubmitRequest{
		QueueName:      queue,
		PayloadJson:    payload,
		MaxRetries:     maxRetries,
		IdempotencyKey: idempotencyKey,
	}
}

func TestSubmitJob_IdempotencyFirstSubmit(t *testing.T) {
	ctx := context.Background()
	rec := newIdempotencyRecorder()
	rec.createResult = db.Job{
		ID:             rec.createID,
		QueueName:      "default",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "race-key", Valid: true},
	}
	handler := &GrpcHandler{Queries: rec.queries()}

	req := submitRequest("default", "{}", "race-key", 3)
	resp, err := handler.SubmitJob(ctx, req)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}
	if resp.JobId != rec.createID.String() || resp.Status != "pending" {
		t.Fatalf("SubmitJob() = (JobId=%s, Status=%s), want (JobId=%s, Status=pending)", resp.JobId, resp.Status, rec.createID.String())
	}
	if rec.createCalls != 1 {
		t.Fatalf("CreateJobWithIdempotency calls = %d, want 1", rec.createCalls)
	}
	if rec.getKeyCalls != 0 {
		t.Fatalf("GetJobByIdempotencyKey calls = %d, want 0 on a clean submit", rec.getKeyCalls)
	}
}

func TestSubmitJob_IdempotencyConflict(t *testing.T) {
	ctx := context.Background()
	rec := newIdempotencyRecorder()
	// First submission with the key creates the job.
	rec.createResult = db.Job{
		ID:             rec.createID,
		QueueName:      "default",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "race-key", Valid: true},
	}
	handler := &GrpcHandler{Queries: rec.queries()}

	first, err := handler.SubmitJob(ctx, submitRequest("default", "{}", "race-key", 3))
	if err != nil {
		t.Fatalf("first SubmitJob() error = %v", err)
	}

	// Second submission with the same key: the INSERT resolves as a no-op
	// (pgx.ErrNoRows) and falls back to returning the existing job.
	rec.createErr = pgx.ErrNoRows
	// The stored job is the one the first submission created.
	existingID := rec.createID
	rec.existingJobs["race-key"] = db.Job{
		ID:             existingID,
		QueueName:      "default",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "race-key", Valid: true},
	}

	second, err := handler.SubmitJob(ctx, submitRequest("default", "{}", "race-key", 3))
	if err != nil {
		t.Fatalf("duplicate SubmitJob() error = %v", err)
	}
	if second.JobId != existingID.String() || second.Status != "pending" {
		t.Fatalf("duplicate SubmitJob() = (JobId=%s, Status=%s), want (JobId=%s, Status=pending)", second.JobId, second.Status, existingID.String())
	}
	if second.JobId != first.JobId {
		t.Fatalf("duplicate SubmitJob() JobId = %s, want the original %s", second.JobId, first.JobId)
	}
	if !rec.queriedKey.Valid || rec.queriedKey.String != "race-key" {
		t.Fatalf("GetJobByIdempotencyKey key = %+v, want {race-key, valid}", rec.queriedKey)
	}
	if rec.getKeyCalls != 1 {
		t.Fatalf("GetJobByIdempotencyKey calls = %d, want 1", rec.getKeyCalls)
	}
	if !rec.lastKey.Valid || rec.lastKey.String != "race-key" {
		t.Fatalf("CreateJobWithIdempotency idempotency_key = %+v, want {race-key, valid}", rec.lastKey)
	}
}

func TestSubmitJob_IdempotencyMaxRetriesClamp(t *testing.T) {
	cases := []struct {
		name  string
		input int32
		want  int32
	}{
		{"negative retries clamped to 3", -1, 3},
		{"above 20 clamped to 3", 21, 3},
		{"valid budget kept", 5, 5},
		{"upper bound kept", 20, 20},
		{"zero kept", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := newIdempotencyRecorder()
			rec.createResult = db.Job{
				ID:        rec.createID,
				QueueName: "default",
				Status:    "pending",
				MaxRetries: tc.input,
			}
			handler := &GrpcHandler{Queries: rec.queries()}

			if _, err := handler.SubmitJob(context.Background(), submitRequest("default", "{}", "", tc.input)); err != nil {
				t.Fatalf("SubmitJob(max_retries=%d) error = %v", tc.input, err)
			}
			if got := rec.maxRetriesLog[0]; got != tc.want {
				t.Fatalf("SubmitJob passed max_retries = %d, want %d", got, tc.want)
			}
		})
	}
}
