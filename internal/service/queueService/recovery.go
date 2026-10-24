package queueService

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func StartRecoveryLoop(ctx context.Context, queries *db.Queries) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				recoverDeadWorkers(ctx, queries)
				recoverExpiredLeases(ctx, queries)
			}
		}
	}()
}

func recoverDeadWorkers(ctx context.Context, queries *db.Queries) {
	threshold := time.Now().Add(-15 * time.Second)
	thresholdTimestamp := pgtype.Timestamp{Time: threshold, Valid: true}

	deadWorkers, err := queries.GetDeadWorkers(ctx, thresholdTimestamp)
	if err != nil {
		return
	}

	for _, w := range deadWorkers {
		log.Printf("Recovery: Worker %s (%s) has missed heartbeats. Cleaning up leases...\n", w.ID.String(), w.WorkerHost)

		leases, err := queries.GetLeasesByWorker(ctx, w.ID.String())
		if err == nil {
			for _, lease := range leases {
				log.Printf("Recovery: Reclaiming leased job %s held by dead worker %s\n", lease.JobID.String(), w.ID.String())

				// Requeue the job
				_, _ = queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
					Status: "pending",
					ID:     lease.JobID,
				})

				// Mark execution failed
				exec, err := queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
					JobID:    lease.JobID,
					WorkerID: w.ID.String(),
				})
				if err == nil {
					_ = queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
						Status: "failed",
						ID:     exec.ID,
					})
				}

				// Release lease
				_ = queries.ReleaseLease(ctx, lease.JobID)
			}
		}

		// Delete dead worker from registry
		_ = queries.DeleteWorker(ctx, w.ID)
	}
}

func recoverExpiredLeases(ctx context.Context, queries *db.Queries) {
	expiredLeases, err := queries.GetExpiredLeases(ctx)
	if err != nil {
		return
	}

	for _, lease := range expiredLeases {
		log.Printf("Recovery: Lease for job %s expired. Reclaiming...\n", lease.JobID.String())

		// Treat lease expiration as a failure attempt (increment retry count)
		job, err := queries.GetJob(ctx, lease.JobID)
		if err == nil {
			if job.RetryCount < job.MaxRetries {
				nextExec := time.Now() // retry immediately
				_, _ = queries.IncrementJobRetryCount(ctx, db.IncrementJobRetryCountParams{
					ScheduledAt: pgtype.Timestamptz{Time: nextExec, Valid: true},
					ID:          lease.JobID,
				})
				log.Printf("Recovery: Requeued job %s (attempt %d/%d)\n", lease.JobID.String(), job.RetryCount+1, job.MaxRetries)
			} else {
				// Move to DLQ
				_, _ = queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
					Status: "dead_letter",
					ID:     lease.JobID,
				})
				_, _ = queries.CreateDeadLetterJob(ctx, db.CreateDeadLetterJobParams{
					JobID:       lease.JobID,
					ErrorReason: pgtype.Text{String: "visibility timeout exceeded and retries exhausted", Valid: true},
				})
				log.Printf("Recovery: Job %s exhausted retries. Moved to DLQ.\n", lease.JobID.String())
			}
		}

		// Mark execution failed
		exec, err := queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
			JobID:    lease.JobID,
			WorkerID: lease.WorkerID,
		})
		if err == nil {
			_ = queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
				Status: "failed",
				ID:     exec.ID,
			})
		}

		// Release the lease
		_ = queries.ReleaseLease(ctx, lease.JobID)
	}
}
