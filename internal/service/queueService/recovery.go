package queueService

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

// nextBackoff returns the delay before a failed job is retried at the given
// retry count: exponential at 2s base (capped at shift 10), bounded at 300s,
// with up to 50% jitter.
func nextBackoff(retryCount int32) time.Duration {
	shift := min(retryCount, 10)
	backoff := min(2*time.Second*(1<<uint(shift)), 300*time.Second)
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
	return backoff/2 + jitter
}

func StartRecoveryLoop(ctx context.Context, queries *db.Queries) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		iteration := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				recoverDeadWorkers(ctx, queries)
				recoverExpiredLeases(ctx, queries)
				// Reconcile jobs stuck in 'running' without an active lease
				// on every 2nd tick.
				iteration++
				if iteration%2 == 0 {
					reconcileOrphanedJobs(ctx, queries)
				}
			}
		}
	}()
}

// reconcileOrphanedJobs finds jobs left in 'running' with no active lease
// (e.g. lost mid-transition between a poll claim and lease acquisition) and
// requeues or dead-letters them.
func reconcileOrphanedJobs(ctx context.Context, queries *db.Queries) {
	orphans, err := queries.GetOrphanedRunningJobs(ctx)
	if err != nil {
		log.Printf("Recovery: failed to list orphaned running jobs: %v\n", err)
		return
	}
	for _, job := range orphans {
		log.Printf("Recovery: Job %s is running without an active lease. Reconciling...\n", job.ID.String())
		if err := requeueOrDeadLetter(ctx, queries, job.ID, "orphaned running job without active lease"); err != nil {
			log.Printf("Recovery: failed to reconcile orphaned job %s: %v\n", job.ID.String(), err)
		}
	}
}

// requeueOrDeadLetter requeues a failed/abandoned job for another attempt
// (scheduled after the next backoff window) if retries remain, otherwise
// moves it to the DLQ and marks its workflow task failed.
func requeueOrDeadLetter(ctx context.Context, queries *db.Queries, jobID pgtype.UUID, reason string) error {
	job, err := queries.GetJob(ctx, jobID)
	if err != nil {
		log.Printf("Recovery: failed to fetch job %s (%s): %v\n", jobID.String(), reason, err)
		return err
	}

	if job.RetryCount < job.MaxRetries {
		if _, err := queries.IncrementJobRetryCount(ctx, db.IncrementJobRetryCountParams{
			ScheduledAt: pgtype.Timestamptz{Time: time.Now().Add(nextBackoff(job.RetryCount)), Valid: true},
			ID:          jobID,
		}); err != nil {
			log.Printf("Recovery: failed to requeue job %s (%s) for retry: %v\n", jobID.String(), reason, err)
			return err
		}
		log.Printf("Recovery: Requeued job %s (%s, attempt %d/%d)\n", jobID.String(), reason, job.RetryCount+1, job.MaxRetries)
		return nil
	}

	if _, err := queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		Status: "dead_letter",
		ID:     jobID,
	}); err != nil {
		log.Printf("Recovery: failed to move job %s (%s) to dead_letter: %v\n", jobID.String(), reason, err)
		return err
	}
	if _, err := queries.CreateDeadLetterJob(ctx, db.CreateDeadLetterJobParams{
		JobID:       jobID,
		ErrorReason: pgtype.Text{String: reason, Valid: reason != ""},
	}); err != nil {
		log.Printf("Recovery: failed to create dead letter record for job %s (%s): %v\n", jobID.String(), reason, err)
		return err
	}
	if job.TaskExecutionID.Valid {
		if err := queries.MarkTaskFailed(ctx, job.TaskExecutionID); err != nil {
			log.Printf("Recovery: failed to mark task %s failed for job %s (%s): %v\n", job.TaskExecutionID.String(), jobID.String(), reason, err)
			return err
		}
	}
	log.Printf("Recovery: Job %s (%s) exhausted retries. Moved to DLQ.\n", jobID.String(), reason)
	return nil
}

func recoverDeadWorkers(ctx context.Context, queries *db.Queries) {
	threshold := time.Now().Add(-15 * time.Second)
	thresholdTimestamp := pgtype.Timestamp{Time: threshold, Valid: true}

	deadWorkers, err := queries.GetDeadWorkers(ctx, thresholdTimestamp)
	if err != nil {
		log.Printf("Recovery: failed to list dead workers: %v\n", err)
		return
	}

	for _, w := range deadWorkers {
		log.Printf("Recovery: Worker %s (%s) has missed heartbeats. Cleaning up leases...\n", w.ID.String(), w.WorkerHost)

		leases, err := queries.GetLeasesByWorker(ctx, w.ID.String())
		if err != nil {
			log.Printf("Recovery: failed to list leases for dead worker %s: %v\n", w.ID.String(), err)
		} else {
			for _, lease := range leases {
				log.Printf("Recovery: Reclaiming leased job %s held by dead worker %s\n", lease.JobID.String(), w.ID.String())

				// Requeue the job (or dead-letter it if retries are exhausted)
				if err := requeueOrDeadLetter(ctx, queries, lease.JobID, "reclaimed from dead worker"); err != nil {
					log.Printf("Recovery: failed to reclaim job %s from dead worker %s: %v\n", lease.JobID.String(), w.ID.String(), err)
				}

				// Mark execution failed
				exec, err := queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
					JobID:    lease.JobID,
					WorkerID: w.ID.String(),
				})
				if err == nil {
					if err := queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
						Status: "failed",
						ID:     exec.ID,
					}); err != nil {
						log.Printf("Recovery: failed to mark execution %s failed for job %s: %v\n", exec.ID.String(), lease.JobID.String(), err)
					}
				}

				// Release lease
				if err := queries.ReleaseLease(ctx, lease.JobID); err != nil {
					log.Printf("Recovery: failed to release lease for job %s: %v\n", lease.JobID.String(), err)
				}
			}
		}

		// Delete dead worker from registry
		if err := queries.DeleteWorker(ctx, w.ID); err != nil {
			log.Printf("Recovery: failed to delete dead worker %s: %v\n", w.ID.String(), err)
		}
	}
}

func recoverExpiredLeases(ctx context.Context, queries *db.Queries) {
	expiredLeases, err := queries.GetExpiredLeases(ctx)
	if err != nil {
		log.Printf("Recovery: failed to list expired leases: %v\n", err)
		return
	}

	for _, lease := range expiredLeases {
		log.Printf("Recovery: Lease for job %s expired. Reclaiming...\n", lease.JobID.String())

		// Re-read the lease (it now lives in worker_leases as completed /
		// renewed) so we never reclaim a job a worker still owns.
		currentLease, err := queries.GetLease(ctx, lease.JobID)
		if err != nil {
			log.Printf("Recovery: failed to re-read lease for job %s: %v\n", lease.JobID.String(), err)
		} else {
			// TODO(exactly-once): replace with atomic conditional UPDATE after lease-query rework
			if currentLease.Completed || currentLease.ExpiresAt.Time.After(time.Now()) {
				log.Printf("Recovery: Lease for job %s was completed or renewed by worker %s. Skipping.\n", lease.JobID.String(), lease.WorkerID)
				continue
			}
		}

		// Treat lease expiration as a failure attempt (increment retry count)
		if err := requeueOrDeadLetter(ctx, queries, lease.JobID, "visibility timeout exceeded and retries exhausted"); err != nil {
			log.Printf("Recovery: failed to reclaim job %s after lease expiry: %v\n", lease.JobID.String(), err)
		}

		// Mark execution failed
		exec, err := queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
			JobID:    lease.JobID,
			WorkerID: lease.WorkerID,
		})
		if err == nil {
			if err := queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
				Status: "failed",
				ID:     exec.ID,
			}); err != nil {
				log.Printf("Recovery: failed to mark execution %s failed for job %s: %v\n", exec.ID.String(), lease.JobID.String(), err)
			}
		}

		// Release the lease
		if err := queries.ReleaseLease(ctx, lease.JobID); err != nil {
			log.Printf("Recovery: failed to release lease for job %s: %v\n", lease.JobID.String(), err)
		}
	}
}
