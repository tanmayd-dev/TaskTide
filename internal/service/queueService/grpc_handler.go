package queueService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/observability"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const LeaseDuration = 30 * time.Second

func durationToInterval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{
		Microseconds: d.Microseconds(),
		Valid:        true,
	}
}

type GrpcHandler struct {
	pb.UnimplementedTaskTideServiceServer
	Pool        *pgxpool.Pool
	Queries     *db.Queries
	MinioClient *minio.Client
}

func (h *GrpcHandler) RegisterWorker(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("gRPC: Registering worker %s:%s with concurrency=%d and capabilities=%v\n", req.Hostname, req.Port, req.Concurrency, req.Capabilities)
	w, err := h.Queries.RegisterWorkerWithMetadata(ctx, db.RegisterWorkerWithMetadataParams{
		WorkerHost:   req.Hostname,
		WorkerPort:   req.Port,
		Concurrency:  req.Concurrency,
		Capabilities: req.Capabilities,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register worker: %v", err)
	}
	return &pb.RegisterResponse{WorkerId: w.ID.String()}, nil
}

func (h *GrpcHandler) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	workerUUID, err := utils.StringToUUID(req.WorkerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID: %v", err)
	}
	err = h.Queries.UpdateWorkerHeartbeat(ctx, workerUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update heartbeat: %v", err)
	}
	return &pb.HeartbeatResponse{Success: true}, nil
}

func (h *GrpcHandler) PollJobs(ctx context.Context, req *pb.PollRequest) (*pb.PollResponse, error) {
	workerUUID, err := utils.StringToUUID(req.WorkerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID: %v", err)
	}

	// Update worker heartbeat on poll
	if err := h.Queries.UpdateWorkerHeartbeat(ctx, workerUUID); err != nil {
		log.Printf("PollJobs: failed to update heartbeat for worker %s: %v\n", workerUUID.String(), err)
	}

	if req.FreeSlots <= 0 {
		return &pb.PollResponse{}, nil
	}
	limit := req.FreeSlots
	if limit <= 0 {
		limit = 1
	}

	if h.Pool == nil {
		return nil, status.Errorf(codes.Internal, "database pool is not configured")
	}

	tx, err := h.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)
	pendingJobs, err := qtx.ClaimPendingJobs(ctx, db.ClaimPendingJobsParams{
		Column1: req.Capabilities,
		Limit:   limit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to claim pending jobs: %v", err)
	}

	var leasedJobs []*pb.Job
	slotsRemaining := limit

	for _, j := range pendingJobs {
		if slotsRemaining <= 0 {
			break
		}

		// Capability matching
		matched := false
		var requiredCapability string
		var finalPayload []byte = j.Payload

		if j.TaskExecutionID.Valid {
			// Workflow task - get its task definition type
			taskExec, err := qtx.GetTaskExecution(ctx, j.TaskExecutionID)
			if err == nil {
				taskDef, err := qtx.GetTask(ctx, taskExec.TaskBase)
				if err == nil {
					requiredCapability = taskDef.TaskType

					// Read file content from MinIO
					var inputContent string
					if taskExec.TaskInput.Valid {
						bucketName := os.Getenv("FILES_BUCKET_NAME")
						if bucketName == "" {
							bucketName = "files"
						}
						object, _, err := repositories.GetFile(ctx, qtx, models.InputFileID(taskExec.TaskInput.String()), h.MinioClient, bucketName)
						if err == nil {
							var contentBytes []byte
							buf := new(bytes.Buffer)
							_, err = buf.ReadFrom(object)
							if err == nil {
								contentBytes = buf.Bytes()
								inputContent = string(contentBytes)
							}
						}
					}

					taskDefModel := utils.TaskDefinitionModel(taskDef)
					payloadStruct := struct {
						Task      models.TaskDefinition `json:"task"`
						InputData string                `json:"input_data"`
					}{
						Task:      taskDefModel,
						InputData: inputContent,
					}
					finalPayload, _ = json.Marshal(payloadStruct)
				}
			}
		} else {
			// Standalone job uses queue_name as capability requirement
			requiredCapability = j.QueueName
		}

		if len(req.Capabilities) == 0 {
			matched = true
		} else {
			for _, cap := range req.Capabilities {
				if cap == requiredCapability {
					matched = true
					break
				}
			}
		}

		if !matched {
			continue
		}

		// Try to lease the job
		_, err = qtx.AcquireLease(ctx, db.AcquireLeaseParams{
			JobID:         j.ID,
			WorkerID:      req.WorkerId,
			LeaseDuration: durationToInterval(LeaseDuration),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to acquire lease: %v", err)
		}

		// Update job status to 'running'
		_, err = qtx.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
			Status: "running",
			ID:     j.ID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update job status: %v", err)
		}

		// Create execution record
		_, err = qtx.CreateExecution(ctx, db.CreateExecutionParams{
			JobID:    j.ID,
			WorkerID: req.WorkerId,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create execution: %v", err)
		}

		leasedJobs = append(leasedJobs, &pb.Job{
			Id:              j.ID.String(),
			QueueName:       j.QueueName,
			Payload:         string(finalPayload),
			RetryCount:      j.RetryCount,
			Priority:        j.Priority,
			TaskExecutionId: j.TaskExecutionID.String(),
		})

		slotsRemaining--
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &pb.PollResponse{Jobs: leasedJobs}, nil
}

func (h *GrpcHandler) CompleteJob(ctx context.Context, req *pb.CompleteRequest) (*pb.CompleteResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "CompleteJob")
	defer span.End()

	jobUUID, err := utils.StringToUUID(req.JobId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid job ID: %v", err)
	}

	lease, err := h.Queries.GetActiveLease(ctx, jobUUID)
	if err != nil || lease.WorkerID != req.WorkerId || lease.Completed {
		return nil, status.Errorf(codes.FailedPrecondition, "worker does not hold active lease for job")
	}

	job, err := h.Queries.GetJob(ctx, jobUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get job: %v", err)
	}
	queueName := job.QueueName

	// Mark lease completed and release it
	if err := h.Queries.MarkLeaseCompleted(ctx, jobUUID); err != nil {
		log.Printf("CompleteJob: failed to mark lease completed for job %s: %v\n", jobUUID.String(), err)
	}
	if err := h.Queries.ReleaseLease(ctx, jobUUID); err != nil {
		log.Printf("CompleteJob: failed to release lease for job %s: %v\n", jobUUID.String(), err)
	}

	// Update execution status
	execStatus := "succeeded"
	if !req.Success {
		execStatus = "failed"
	}

	exec, err := h.Queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
		JobID:    jobUUID,
		WorkerID: req.WorkerId,
	})
	if err != nil {
		// No matchable execution record (e.g. expired/re-acquired lease) —
		// skip the execution update and attempt log below; the retry
		// schedule itself must NOT be dropped.
		log.Printf("CompleteJob: no active execution found for job %s (worker %s): %v\n", jobUUID.String(), req.WorkerId, err)
	} else {
		if err := h.Queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
			Status: execStatus,
			ID:     exec.ID,
		}); err != nil {
			log.Printf("CompleteJob: failed to update execution %s for job %s: %v\n", exec.ID.String(), jobUUID.String(), err)
		}

		// Record latency
		duration := time.Since(exec.StartedAt.Time).Seconds()
		observability.JobDurationHistogram.WithLabelValues(queueName, execStatus).Observe(duration)
	}

	// Determine if this job is in a terminal state for workflow purposes.
	// A job is terminal when it succeeds OR when retries are exhausted (going to DLQ).
	// An intermediate failure that will be retried is NOT terminal.
	isTerminal := false

	if req.Success {
		if _, err := h.Queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
			Status: "completed",
			ID:     jobUUID,
		}); err != nil {
			log.Printf("CompleteJob: failed to mark job %s completed: %v\n", jobUUID.String(), err)
		}
		observability.JobExecutionCounter.WithLabelValues(queueName, "succeeded").Inc()
		isTerminal = true
	} else {
		if job.RetryCount < job.MaxRetries {
			// Calculate exponential backoff duration with jitter
			nextExec := time.Now().Add(nextBackoff(job.RetryCount))
			if _, err := h.Queries.IncrementJobRetryCount(ctx, db.IncrementJobRetryCountParams{
				ScheduledAt: pgtype.Timestamptz{Time: nextExec, Valid: true},
				ID:          jobUUID,
			}); err != nil {
				log.Printf("CompleteJob: failed to schedule retry for job %s: %v\n", jobUUID.String(), err)
			}
			log.Printf("CompleteJob: Job %s failed. Retry count incremented to %d. Scheduled for %v.\n", jobUUID.String(), job.RetryCount+1, nextExec)

			observability.JobExecutionCounter.WithLabelValues(queueName, "failed_retry").Inc()
			observability.RetryCounter.WithLabelValues(queueName).Inc()

			// Log retry attempt in retries table — but do NOT update workflow state yet.
			if exec.ID.Valid {
				if _, err := h.Queries.CreateRetry(ctx, db.CreateRetryParams{
					JobID:         jobUUID,
					ExecutionID:   exec.ID,
					AttemptNumber: job.RetryCount + 1,
					ErrorReason:   pgtype.Text{String: req.ErrorReason, Valid: req.ErrorReason != ""},
				}); err != nil {
					log.Printf("CompleteJob: failed to record retry attempt for job %s: %v\n", jobUUID.String(), err)
				}
			}
			// isTerminal remains false — workflow task_execution is NOT marked done yet.
		} else {
			// Max retries exceeded - move to DLQ
			if _, err := h.Queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
				Status: "dead_letter",
				ID:     jobUUID,
			}); err != nil {
				log.Printf("CompleteJob: failed to move job %s to dead_letter: %v\n", jobUUID.String(), err)
			}
			if _, err := h.Queries.CreateDeadLetterJob(ctx, db.CreateDeadLetterJobParams{
				JobID:       jobUUID,
				ErrorReason: pgtype.Text{String: req.ErrorReason, Valid: req.ErrorReason != ""},
			}); err != nil {
				log.Printf("CompleteJob: failed to create dead letter record for job %s: %v\n", jobUUID.String(), err)
			}
			// Mark the workflow task as failed so downstream work is never fanned out.
			if job.TaskExecutionID.Valid {
				if err := h.Queries.MarkTaskFailed(ctx, job.TaskExecutionID); err != nil {
					log.Printf("CompleteJob: failed to mark task %s failed for job %s: %v\n", job.TaskExecutionID.String(), jobUUID.String(), err)
				}
			}
			log.Printf("CompleteJob: Job %s failed and retries exhausted. Moved to DLQ.\n", jobUUID.String())
			observability.JobExecutionCounter.WithLabelValues(queueName, "exhausted_dlq").Inc()
			isTerminal = true
		}
	}

	// Always record an attempt for every execution so that the task_attempt table
	// provides a full audit trail (e.g., 1 failed + 1 succeeded = 2 attempts).
	// Marking the task_execution as "complete" and dispatching downstream jobs
	// only happens at a terminal state to avoid premature workflow progression.
	if job.TaskExecutionID.Valid {
		taskExec, err := h.Queries.GetTaskExecution(ctx, job.TaskExecutionID)
		if err != nil {
			log.Printf("CompleteJob: failed to fetch task execution for job %s: %v\n", jobUUID.String(), err)
		} else {
			taskExecModel := models.WorkerTaskExecution{
				ID:         models.TaskExecutionID(taskExec.ID.String()),
				TaskWorker: models.WorkerID(taskExec.TaskWorker.String()),
				// TaskCompleted is deliberately false for intermediate retries —
				// it is set true only when isTerminal (below).
				TaskCompleted: isTerminal,
				TaskSucceeded: req.Success,
				TaskOutput:    req.OutputData,
			}

			// Record this execution attempt regardless of outcome
			if err := repositories.CreateTaskAttempt(ctx, h.Queries, taskExecModel); err != nil {
				log.Printf("CompleteJob: failed to create task attempt for task %s (job %s): %v\n", taskExecModel.ID, jobUUID, err)
			}

			// Only finalise the workflow state on a terminal success. A
			// terminal failure was already handled by MarkTaskFailed in the
			// failure branch above and must NEVER mark the task complete
			// or fan out downstream work.
			if isTerminal && req.Success {
				// If task execution has output file defined, update the content in storage
				if taskExec.TaskOutput.Valid {
					bucketName := os.Getenv("FILES_BUCKET_NAME")
					if bucketName == "" {
						bucketName = "files"
					}
					err = repositories.UpdateFileContent(ctx, h.Queries, h.MinioClient, bucketName, models.InputFileID(taskExec.TaskOutput.String()), req.OutputData)
					if err != nil {
						log.Printf("CompleteJob: failed to update file content: %v\n", err)
					}
				}

				// Mark task completed in workflow system
				if err := repositories.MarkTaskComplete(ctx, h.Queries, taskExecModel.ID); err != nil {
					log.Printf("CompleteJob: failed to mark task %s complete: %v\n", taskExecModel.ID, err)
				}

				// Workflow task succeeded: submit any downstream tasks that are
				// now ready. Synchronous call — bounded because gRPC handler
				// processing serializes callers; the recovery loop sweeps as
				// a backstop.
				if err := claimAndSubmitReadyTasks(ctx, h.Pool); err != nil {
					log.Printf("CompleteJob: failed to submit ready workflow tasks: %v\n", err)
				}
			}
		}
	}

	return &pb.CompleteResponse{Success: true}, nil
}

func (h *GrpcHandler) ExtendLease(ctx context.Context, req *pb.ExtendLeaseRequest) (*pb.ExtendLeaseResponse, error) {
	jobUUID, err := utils.StringToUUID(req.JobId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid job ID: %v", err)
	}

	extensionSeconds := req.ExtensionSeconds
	if extensionSeconds <= 0 {
		extensionSeconds = int32(LeaseDuration.Seconds())
	}

	_, err = h.Queries.ExtendLease(ctx, db.ExtendLeaseParams{
		ExtensionInterval: durationToInterval(time.Duration(extensionSeconds) * time.Second),
		JobID:             jobUUID,
		WorkerID:          req.WorkerId,
	})
	if err != nil {
		return &pb.ExtendLeaseResponse{Success: false}, nil
	}

	return &pb.ExtendLeaseResponse{Success: true}, nil
}

func (h *GrpcHandler) SubmitJob(ctx context.Context, req *pb.SubmitRequest) (*pb.SubmitResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "SubmitJob")
	defer span.End()

	// Clamp retry budget: values outside [0, 20] are rejected in favour of 3.
	maxRetries := req.MaxRetries
	if maxRetries < 0 || maxRetries > 20 {
		maxRetries = 3
	}

	if err := h.Queries.EnsureQueueExists(ctx, req.QueueName); err != nil {
		log.Printf("SubmitJob: failed to ensure queue %s exists: %v\n", req.QueueName, err)
	}
	// CreateJobWithIdempotency resolves the submit race atomically: on an
	// idempotency-key conflict the insert is a no-op and we fall back to
	// returning the existing job.
	j, err := h.Queries.CreateJobWithIdempotency(ctx, db.CreateJobWithIdempotencyParams{
		QueueName:       req.QueueName,
		Payload:         []byte(req.PayloadJson),
		MaxRetries:      maxRetries,
		Delay:           req.DelaySeconds,
		ScheduledAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Duration(req.DelaySeconds) * time.Second), Valid: true},
		Priority:        req.Priority,
		IdempotencyKey:  pgtype.Text{String: req.IdempotencyKey, Valid: req.IdempotencyKey != ""},
		TaskExecutionID: pgtype.UUID{},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict + DO NOTHING: a job with this idempotency key already exists.
			if req.IdempotencyKey == "" {
				return nil, status.Errorf(codes.Internal, "failed to create job: %v", err)
			}
			existingJob, fetchErr := h.Queries.GetJobByIdempotencyKey(ctx, pgtype.Text{String: req.IdempotencyKey, Valid: true})
			if fetchErr != nil {
				return nil, status.Errorf(codes.Internal, "failed to fetch existing job by idempotency key: %v", fetchErr)
			}
			log.Printf("SubmitJob: Idempotency conflict! Returning existing job ID: %s\n", existingJob.ID.String())
			return &pb.SubmitResponse{JobId: existingJob.ID.String(), Status: existingJob.Status}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to create job: %v", err)
	}

	observability.JobSubmissionCounter.WithLabelValues(req.QueueName).Inc()

	return &pb.SubmitResponse{JobId: j.ID.String(), Status: j.Status}, nil
}

func (h *GrpcHandler) InspectQueue(ctx context.Context, req *pb.InspectQueueRequest) (*pb.InspectQueueResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "InspectQueue")
	defer span.End()

	queue, err := h.Queries.GetQueue(ctx, req.QueueName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "queue '%s' not found", req.QueueName)
		}
		return nil, status.Errorf(codes.Internal, "failed to get queue: %v", err)
	}
	_ = queue

	counts, err := h.Queries.GetQueueJobCounts(ctx, req.QueueName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get queue job counts: %v", err)
	}

	return &pb.InspectQueueResponse{
		QueueName:  req.QueueName,
		Pending:    counts.PendingCount,
		Running:    counts.RunningCount,
		Completed:  counts.CompletedCount,
		Failed:     counts.FailedCount,
		DeadLetter: counts.DeadLetterCount,
	}, nil
}

func (h *GrpcHandler) RetryJob(ctx context.Context, req *pb.RetryJobRequest) (*pb.RetryJobResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "RetryJob")
	defer span.End()

	jobUUID, err := utils.StringToUUID(req.JobId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid job ID: %v", err)
	}

	if h.Pool == nil {
		return nil, status.Errorf(codes.Internal, "database pool is not configured")
	}

	tx, err := h.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)

	job, err := qtx.GetJob(ctx, jobUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "job '%s' not found", req.JobId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get job: %v", err)
	}

	// Guard against an active, unexpired lease: a live worker still owns the
	// job. An expired (or completed) lease is released so the retried job can
	// be claimed afresh.
	lease, err := qtx.GetActiveLease(ctx, jobUUID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "failed to get lease for job: %v", err)
	}
	if err == nil && lease.ExpiresAt.Time.After(time.Now()) {
		return nil, status.Errorf(codes.FailedPrecondition, "job '%s' is leased by an active worker", req.JobId)
	}
	if err == nil {
		if err := qtx.ReleaseLease(ctx, jobUUID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to release expired lease for job: %v", err)
		}
	}

	// The status guard lives in the query's WHERE clause: 0 rows means the job
	// is not in a retryable state ('failed' or 'dead_letter').
	retried, err := qtx.RetryJob(ctx, jobUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.InvalidArgument, "job '%s' is in status '%s'; can only retry failed or dead_letter jobs", req.JobId, job.Status)
		}
		return nil, status.Errorf(codes.Internal, "failed to retry job: %v", err)
	}

	// The job was dead-lettered — remove the dead-letter record now that it is
	// back to 'pending'.
	if job.Status == "dead_letter" {
		if err := qtx.DeleteDeadLetterJob(ctx, jobUUID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to delete dead letter record for job: %v", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	log.Printf("RetryJob: Job %s moved from '%s' back to '%s'\n", retried.ID.String(), job.Status, retried.Status)
	return &pb.RetryJobResponse{JobId: retried.ID.String(), Status: retried.Status}, nil
}

func (h *GrpcHandler) ListWorkers(ctx context.Context, req *pb.ListWorkersRequest) (*pb.ListWorkersResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "ListWorkers")
	defer span.End()

	workers, err := h.Queries.ListWorkersWithStatus(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workers: %v", err)
	}

	result := make([]*pb.WorkerStatus, 0, len(workers))
	for _, w := range workers {
		result = append(result, &pb.WorkerStatus{
			WorkerId:      w.ID.String(),
			Host:          w.WorkerHost,
			Active:        w.Active,
			LastHeartbeat: w.LastHeartbeat.Time.Unix(),
		})
	}
	return &pb.ListWorkersResponse{Workers: result}, nil
}

func (h *GrpcHandler) PauseQueue(ctx context.Context, req *pb.PauseQueueRequest) (*pb.PauseQueueResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "PauseQueue")
	defer span.End()

	if _, err := h.Queries.GetQueue(ctx, req.QueueName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "queue '%s' not found", req.QueueName)
		}
		return nil, status.Errorf(codes.Internal, "failed to get queue: %v", err)
	}

	if err := h.Queries.SetQueuePaused(ctx, db.SetQueuePausedParams{
		Name:   req.QueueName,
		Paused: req.Paused,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update queue paused state: %v", err)
	}

	return &pb.PauseQueueResponse{QueueName: req.QueueName, Paused: req.Paused}, nil
}

// claimAndSubmitReadyTasks claims ready, unassigned workflow tasks and
// submits each one as a job. Claiming runs in a per-task transaction that
// takes GetReadyUnassignedTasks' FOR UPDATE SKIP LOCKED row locks, so a
// failed task rolls back unassigned and is picked up by a later sweep
// without blocking other ready tasks.
func claimAndSubmitReadyTasks(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("database pool is not configured")
	}
	for {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		qtx := db.New(tx)

		readyTasks, err := qtx.GetReadyUnassignedTasks(ctx)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if len(readyTasks) == 0 {
			_ = tx.Rollback(ctx)
			return nil
		}
		task := readyTasks[0]

		if err := qtx.MarkTaskAssigned(ctx, task.ID); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		taskBase, err := qtx.GetTask(ctx, task.TaskBase)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		payloadBytes, err := json.Marshal(utils.TaskDefinitionModel(taskBase))
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if err := qtx.EnsureQueueExists(ctx, "default"); err != nil {
			log.Printf("claimAndSubmitReadyTasks: EnsureQueueExists erred: %v\n", err)
		}
		if _, err := qtx.CreateJob(ctx, db.CreateJobParams{
			QueueName:       "default",
			Payload:         payloadBytes,
			MaxRetries:      2,
			Delay:           0,
			ScheduledAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			Priority:        0,
			IdempotencyKey:  pgtype.Text{String: "", Valid: false},
			TaskExecutionID: task.ID,
		}); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		observability.JobSubmissionCounter.WithLabelValues("default").Inc()
		log.Printf("claimAndSubmitReadyTasks: Successfully submitted workflow task %s as a job\n", task.ID.String())

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
}
