package queueService

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/testutil"
)

// testUUID returns a deterministic valid UUID for test identifiers.
func testUUID(b byte) pgtype.UUID {
	var bytes [16]byte
	for i := range bytes {
		bytes[i] = b
	}
	return pgtype.UUID{Bytes: bytes, Valid: true}
}

// jobRowValues returns a jobs row's scanned column values in sqlc's SELECT
// order, for testutil.RowFromValues.
func jobRowValues(job db.Job) []any {
	return []any{
		job.ID,
		job.QueueName,
		job.Payload,
		job.MaxRetries,
		job.RetryCount,
		job.Delay,
		job.ScheduledAt,
		job.Priority,
		job.Status,
		job.IdempotencyKey,
		job.TaskExecutionID,
		job.CreatedAt,
		job.UpdatedAt,
	}
}

// recoveryRecorder captures the side effects of requeueOrDeadLetter by
// dispatching on the sqlc SQL text, without a real database.
type recoveryRecorder struct {
	jobs         map[pgtype.UUID]db.Job
	getJobErr    error
	incrementErr error

	incrementCount       int
	incrementID          pgtype.UUID
	incrementScheduledAt pgtype.Timestamptz

	statusUpdates  []db.UpdateJobStatusParams
	deadLetterJobs []pgtype.UUID
	tasksFailed    []pgtype.UUID
}

func newRecoveryRecorder() *recoveryRecorder {
	return &recoveryRecorder{jobs: make(map[pgtype.UUID]db.Job)}
}

func (r *recoveryRecorder) queries() *db.Queries {
	mock := &testutil.MockDB{
		QueryRowFunc: r.queryRow,
		ExecFunc:     r.exec,
	}
	return db.New(mock)
}

func (r *recoveryRecorder) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM jobs WHERE id = $1"): // GetJob
		if r.getJobErr != nil {
			return testutil.RowFromError(r.getJobErr)
		}
		job, ok := r.jobs[args[0].(pgtype.UUID)]
		if !ok {
			return testutil.RowFromError(pgx.ErrNoRows)
		}
		return testutil.RowFromValues(jobRowValues(job)...)
	case strings.Contains(sql, "SET retry_count = retry_count + 1"): // IncrementJobRetryCount
		if r.incrementErr != nil {
			return testutil.RowFromError(r.incrementErr)
		}
		r.incrementCount++
		r.incrementScheduledAt = args[0].(pgtype.Timestamptz)
		r.incrementID = args[1].(pgtype.UUID)
		job := r.jobs[args[1].(pgtype.UUID)]
		job.RetryCount = job.RetryCount + 1
		return testutil.RowFromValues(jobRowValues(job)...)
	case strings.Contains(sql, "FROM jobs WHERE idempotency_key = $1"): // GetJobByIdempotencyKey
		return testutil.RowFromError(pgx.ErrNoRows)
	case strings.Contains(sql, "UPDATE jobs\nSET status = $1"): // UpdateJobStatus
		r.statusUpdates = append(r.statusUpdates, db.UpdateJobStatusParams{
			Status: args[0].(string),
			ID:     args[1].(pgtype.UUID),
		})
		job := r.jobs[args[1].(pgtype.UUID)]
		job.Status = args[0].(string)
		return testutil.RowFromValues(jobRowValues(job)...)
	case strings.Contains(sql, "INSERT INTO dead_letter_jobs"): // CreateDeadLetterJob
		r.deadLetterJobs = append(r.deadLetterJobs, args[0].(pgtype.UUID))
		return testutil.RowFromValues(
			testUUID(9),
			args[0].(pgtype.UUID),
			pgtype.Timestamptz{Time: time.Now(), Valid: true},
			pgtype.Text{String: "reason", Valid: true},
		)
	}
	return testutil.RowFromError(pgx.ErrNoRows)
}

func (r *recoveryRecorder) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "task_failed = TRUE") { // MarkTaskFailed
		r.tasksFailed = append(r.tasksFailed, args[0].(pgtype.UUID))
	}
	return pgconn.CommandTag{}, nil
}

func TestRequeueOrDeadLetter_RetriesRemaining(t *testing.T) {
	ctx := context.Background()
	rec := newRecoveryRecorder()
	jobID := testUUID(1)
	rec.jobs[jobID] = db.Job{
		ID:              jobID,
		QueueName:       "default",
		Status:          "running",
		RetryCount:      0,
		MaxRetries:      3,
		TaskExecutionID: testUUID(2),
	}

	before := time.Now()
	if err := requeueOrDeadLetter(ctx, rec.queries(), jobID, "lease expired"); err != nil {
		t.Fatalf("requeueOrDeadLetter() error = %v", err)
	}

	if rec.incrementCount != 1 {
		t.Fatalf("IncrementJobRetryCount call count = %d, want 1", rec.incrementCount)
	}
	if rec.incrementID != jobID {
		t.Fatalf("IncrementJobRetryCount ID = %v, want %v", rec.incrementID, jobID)
	}
	if !rec.incrementScheduledAt.Valid {
		t.Fatalf("IncrementJobRetryCount scheduled_at must be set (not zero)")
	}
	scheduled := rec.incrementScheduledAt.Time
	if !scheduled.After(before) {
		t.Fatalf("IncrementJobRetryCount scheduled_at = %v, want in the future (after %v)", scheduled, before)
	}
	if scheduled.After(before.Add(301 * time.Second)) {
		t.Fatalf("IncrementJobRetryCount scheduled_at = %v, want within the backoff window (<= %v)", scheduled, before.Add(301*time.Second))
	}

	// While retries remain the job must neither be dead-lettered nor its
	// workflow task marked failed.
	if len(rec.statusUpdates) != 0 {
		t.Fatalf("UpdateJobStatus calls = %d, want 0", len(rec.statusUpdates))
	}
	if len(rec.deadLetterJobs) != 0 {
		t.Fatalf("CreateDeadLetterJob calls = %d, want 0", len(rec.deadLetterJobs))
	}
	if len(rec.tasksFailed) != 0 {
		t.Fatalf("MarkTaskFailed calls = %d, want 0", len(rec.tasksFailed))
	}
}

func TestRequeueOrDeadLetter_DeadLettersExhaustedJob(t *testing.T) {
	cases := []struct {
		name               string
		taskExecutionValid bool
		wantMarkTaskFailed bool
	}{
		{"with task execution", true, true},
		{"without task execution", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			rec := newRecoveryRecorder()
			jobID := testUUID(1)
			teID := pgtype.UUID{}
			if tc.taskExecutionValid {
				teID = testUUID(2)
			}
			rec.jobs[jobID] = db.Job{
				ID:              jobID,
				QueueName:       "default",
				Status:          "running",
				RetryCount:      3,
				MaxRetries:      3,
				TaskExecutionID: teID,
			}

			if err := requeueOrDeadLetter(ctx, rec.queries(), jobID, "retries exhausted"); err != nil {
				t.Fatalf("requeueOrDeadLetter() error = %v", err)
			}

			if rec.incrementCount != 0 {
				t.Fatalf("IncrementJobRetryCount call count = %d, want 0", rec.incrementCount)
			}
			if len(rec.statusUpdates) != 1 || rec.statusUpdates[0].ID != jobID || rec.statusUpdates[0].Status != "dead_letter" {
				t.Fatalf("UpdateJobStatus calls = %+v, want exactly [%s -> %s]", rec.statusUpdates, jobID, "dead_letter")
			}
			if len(rec.deadLetterJobs) != 1 || rec.deadLetterJobs[0] != jobID {
				t.Fatalf("CreateDeadLetterJob calls = %v, want exactly [%v]", rec.deadLetterJobs, jobID)
			}

			if tc.wantMarkTaskFailed {
				if len(rec.tasksFailed) != 1 || rec.tasksFailed[0] != teID {
					t.Fatalf("MarkTaskFailed calls = %v, want exactly [%v]", rec.tasksFailed, teID)
				}
			} else if len(rec.tasksFailed) != 0 {
				t.Fatalf("MarkTaskFailed calls = %v, want 0 for invalid task execution", rec.tasksFailed)
			}
		})
	}
}

func TestRequeueOrDeadLetter_PropagatesErrors(t *testing.T) {
	cases := []struct {
		name string
		mock func(rec *recoveryRecorder)
	}{
		{"GetJob failure", func(rec *recoveryRecorder) {
			jobID := testUUID(1)
			rec.jobs[jobID] = db.Job{ID: jobID, RetryCount: 0, MaxRetries: 3}
			rec.getJobErr = errors.New("db outage")
		}},
		{"IncrementJobRetryCount failure", func(rec *recoveryRecorder) {
			jobID := testUUID(1)
			rec.jobs[jobID] = db.Job{ID: jobID, RetryCount: 0, MaxRetries: 3}
			rec.incrementErr = errors.New("db outage")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := newRecoveryRecorder()
			tc.mock(rec)
			jobID := testUUID(1)
			if err := requeueOrDeadLetter(context.Background(), rec.queries(), jobID, "reason"); err == nil {
				t.Fatalf("requeueOrDeadLetter() expected error for %s, got nil", tc.name)
			}
		})
	}
}
