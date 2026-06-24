package queueService

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

type GrpcHandler struct {
	pb.UnimplementedTaskTideServiceServer
	Queries     *db.Queries
	MinioClient *minio.Client
}

var workflowSubmitMutex sync.Mutex

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
	_ = h.Queries.UpdateWorkerHeartbeat(ctx, workerUUID)

	pendingJobs, err := h.Queries.GetPendingJobs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get pending jobs: %v", err)
	}

	var leasedJobs []*pb.Job
	slotsRemaining := req.FreeSlots
	if slotsRemaining <= 0 {
		slotsRemaining = 1
	}

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
			taskExec, err := h.Queries.GetTaskExecution(ctx, j.TaskExecutionID)
			if err == nil {
				taskDef, err := h.Queries.GetTask(ctx, taskExec.TaskBase)
				if err == nil {
					requiredCapability = taskDef.TaskType
					
					// Read file content from MinIO
					var inputContent string
					if taskExec.TaskInput.Valid {
						bucketName := "tasktide-files"
						object, _, err := repositories.GetFile(ctx, h.Queries, models.InputFileID(taskExec.TaskInput.String()), h.MinioClient, bucketName)
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
		leaseDuration := 30 * time.Second
		expiresAt := pgtype.Timestamp{Time: time.Now().Add(leaseDuration), Valid: true}
		_, err = h.Queries.AcquireLease(ctx, db.AcquireLeaseParams{
			JobID:     j.ID,
			WorkerID:  req.WorkerId,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			// Already leased by another worker
			continue
		}

		// Update job status to 'running'
		_, _ = h.Queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
			Status: "running",
			ID:     j.ID,
		})

		// Create execution record
		_, _ = h.Queries.CreateExecution(ctx, db.CreateExecutionParams{
			JobID:    j.ID,
			WorkerID: req.WorkerId,
		})

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

	return &pb.PollResponse{Jobs: leasedJobs}, nil
}

func (h *GrpcHandler) CompleteJob(ctx context.Context, req *pb.CompleteRequest) (*pb.CompleteResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "CompleteJob")
	defer span.End()

	jobUUID, err := utils.StringToUUID(req.JobId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid job ID: %v", err)
	}

	var queueName string = "default"
	job, err := h.Queries.GetJob(ctx, jobUUID)
	if err == nil {
		queueName = job.QueueName
	}

	// Mark lease completed and release it
	_ = h.Queries.MarkLeaseCompleted(ctx, jobUUID)
	_ = h.Queries.ReleaseLease(ctx, jobUUID)

	// Update execution status
	execStatus := "succeeded"
	if !req.Success {
		execStatus = "failed"
	}

	exec, err := h.Queries.GetActiveExecution(ctx, db.GetActiveExecutionParams{
		JobID:    jobUUID,
		WorkerID: req.WorkerId,
	})
	if err == nil {
		_ = h.Queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
			Status: execStatus,
			ID:     exec.ID,
		})

		// Record latency
		duration := time.Since(exec.StartedAt.Time).Seconds()
		observability.JobDurationHistogram.WithLabelValues(queueName, execStatus).Observe(duration)
	}

	if req.Success {
		_, _ = h.Queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
			Status: "completed",
			ID:     jobUUID,
		})
		observability.JobExecutionCounter.WithLabelValues(queueName, "succeeded").Inc()
	} else {
		if err == nil {
			if job.RetryCount < job.MaxRetries {
				// Calculate exponential backoff duration: 2^(retry_count) * 2 seconds
				backoffSec := 1 << (job.RetryCount) * 2
				nextExec := time.Now().Add(time.Duration(backoffSec) * time.Second)
				_, _ = h.Queries.IncrementJobRetryCount(ctx, db.IncrementJobRetryCountParams{
					ScheduledAt: pgtype.Timestamp{Time: nextExec, Valid: true},
					ID:          jobUUID,
				})
				log.Printf("CompleteJob: Job %s failed. Retry count incremented to %d. Scheduled for %v.\n", jobUUID.String(), job.RetryCount+1, nextExec)

				observability.JobExecutionCounter.WithLabelValues(queueName, "failed_retry").Inc()
				observability.RetryCounter.WithLabelValues(queueName).Inc()

				// Log retry attempt
				if exec.ID.Valid {
					_, _ = h.Queries.CreateRetry(ctx, db.CreateRetryParams{
						JobID:         jobUUID,
						ExecutionID:   exec.ID,
						AttemptNumber: job.RetryCount + 1,
						ErrorReason:   pgtype.Text{String: req.ErrorReason, Valid: req.ErrorReason != ""},
					})
				}
			} else {
				// Max retries exceeded - move to DLQ
				_, _ = h.Queries.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
					Status: "dead_letter",
					ID:     jobUUID,
				})
				_, _ = h.Queries.CreateDeadLetterJob(ctx, db.CreateDeadLetterJobParams{
					JobID:       jobUUID,
					ErrorReason: pgtype.Text{String: req.ErrorReason, Valid: req.ErrorReason != ""},
				})
				log.Printf("CompleteJob: Job %s failed and retries exhausted. Moved to DLQ.\n", jobUUID.String())
				observability.JobExecutionCounter.WithLabelValues(queueName, "exhausted_dlq").Inc()
			}
		}
	}

	// Handle workflow task execution updates
	if job.TaskExecutionID.Valid {
		taskExec, err := h.Queries.GetTaskExecution(ctx, job.TaskExecutionID)
		if err == nil {
			taskExecModel := models.WorkerTaskExecution{
				ID:            models.TaskExecutionID(taskExec.ID.String()),
				TaskWorker:    models.WorkerID(taskExec.TaskWorker.String()),
				TaskCompleted: true,
				TaskSucceeded: req.Success,
				TaskOutput:    req.OutputData,
			}

			// If success and task execution has output file defined, update the content in storage
			if req.Success && taskExec.TaskOutput.Valid {
				bucketName := "tasktide-files"
				err = repositories.UpdateFileContent(ctx, h.Queries, h.MinioClient, bucketName, models.InputFileID(taskExec.TaskOutput.String()), req.OutputData)
				if err != nil {
					log.Printf("CompleteJob: failed to update file content: %v\n", err)
				}
			}

			// Create task attempt
			_ = repositories.CreateTaskAttempt(ctx, h.Queries, taskExecModel)

			// Mark task completed in workflow system
			_ = repositories.MarkTaskComplete(ctx, h.Queries, taskExecModel.ID)

			// If workflow task succeeded, submit any downstream tasks that are now ready
			if req.Success {
				go h.checkAndSubmitWorkflowJobs(context.Background())
			}
		}
	}

	return &pb.CompleteResponse{Success: true}, nil
}

func (h *GrpcHandler) SubmitJob(ctx context.Context, req *pb.SubmitRequest) (*pb.SubmitResponse, error) {
	ctx, span := observability.Tracer.Start(ctx, "SubmitJob")
	defer span.End()

	// Check Idempotency Key
	if req.IdempotencyKey != "" {
		existingJob, err := h.Queries.GetJobByIdempotencyKey(ctx, pgtype.Text{String: req.IdempotencyKey, Valid: true})
		if err == nil {
			log.Printf("SubmitJob: Idempotency hit! Returning existing job ID: %s\n", existingJob.ID.String())
			return &pb.SubmitResponse{JobId: existingJob.ID.String(), Status: existingJob.Status}, nil
		}
	}

	j, err := h.Queries.CreateJob(ctx, db.CreateJobParams{
		QueueName:      req.QueueName,
		Payload:        []byte(req.PayloadJson),
		MaxRetries:     req.MaxRetries,
		Delay:          req.DelaySeconds,
		ScheduledAt:    pgtype.Timestamp{Time: time.Now().Add(time.Duration(req.DelaySeconds) * time.Second), Valid: true},
		Priority:       req.Priority,
		IdempotencyKey: pgtype.Text{String: req.IdempotencyKey, Valid: req.IdempotencyKey != ""},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create job: %v", err)
	}

	observability.JobExecutionCounter.WithLabelValues(req.QueueName, "submitted").Inc()

	return &pb.SubmitResponse{JobId: j.ID.String(), Status: j.Status}, nil
}

func (h *GrpcHandler) checkAndSubmitWorkflowJobs(ctx context.Context) {
	workflowSubmitMutex.Lock()
	defer workflowSubmitMutex.Unlock()

	for {
		task, err := h.Queries.GetUnassignedTasks(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			log.Printf("checkAndSubmitWorkflowJobs: GetUnassignedTasks erred: %v\n", err)
			break
		}

		err = h.Queries.MarkTaskAssigned(ctx, task.ID)
		if err != nil {
			log.Printf("checkAndSubmitWorkflowJobs: MarkTaskAssigned erred: %v\n", err)
			break
		}

		taskBase, err := h.Queries.GetTask(ctx, task.TaskBase)
		if err != nil {
			log.Printf("checkAndSubmitWorkflowJobs: GetTask erred: %v\n", err)
			continue
		}

		taskBaseModel := utils.TaskDefinitionModel(taskBase)
		payloadBytes, err := json.Marshal(taskBaseModel)
		if err != nil {
			log.Printf("checkAndSubmitWorkflowJobs: Marshal task base erred: %v\n", err)
			continue
		}

		_, err = h.Queries.CreateJob(ctx, db.CreateJobParams{
			QueueName:       "default",
			Payload:         payloadBytes,
			MaxRetries:      3,
			Delay:           0,
			ScheduledAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			Priority:        0,
			IdempotencyKey:  pgtype.Text{String: "", Valid: false},
			TaskExecutionID: pgtype.UUID{Bytes: task.ID.Bytes, Valid: true},
		})
		if err != nil {
			log.Printf("checkAndSubmitWorkflowJobs: CreateJob failed: %v\n", err)
		} else {
			log.Printf("checkAndSubmitWorkflowJobs: Successfully submitted workflow task %s as a job\n", task.ID.String())
		}
	}
}
