-- name: AddQueue :one
INSERT INTO queues (
    name, paused
) VALUES (
    $1, $2
) ON CONFLICT (name) DO UPDATE SET paused = EXCLUDED.paused RETURNING *;

-- name: EnsureQueueExists :exec
INSERT INTO queues (
    name, paused
) VALUES (
    $1, FALSE
) ON CONFLICT (name) DO NOTHING;

-- name: GetQueue :one
SELECT * FROM queues WHERE name = $1;

-- name: SetQueuePaused :exec
UPDATE queues SET paused = $2 WHERE name = $1;

-- name: CreateJob :one
INSERT INTO jobs (
    queue_name, payload, max_retries, retry_count, delay, scheduled_at, priority, status, idempotency_key, task_execution_id
) VALUES (
    $1, $2, $3, 0, $4, $5, $6, 'pending', $7, $8
) RETURNING *;

-- name: CreateJobWithIdempotency :one
INSERT INTO jobs (
    queue_name, payload, max_retries, retry_count, delay, scheduled_at, priority, status, idempotency_key, task_execution_id
) VALUES (
    $1, $2, $3, 0, $4, $5, $6, 'pending', $7, $8
) ON CONFLICT (idempotency_key) DO NOTHING
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = $1;

-- name: GetJobByIdempotencyKey :one
SELECT * FROM jobs WHERE idempotency_key = $1;

-- name: UpdateJobStatus :one
UPDATE jobs
SET status = $1, updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: IncrementJobRetryCount :one
UPDATE jobs
SET retry_count = retry_count + 1,
    scheduled_at = sqlc.arg(scheduled_at)::TIMESTAMPTZ,
    status = 'pending',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetOrphanedRunningJobs :many
SELECT j.* FROM jobs j
WHERE j.status = 'running'
  AND NOT EXISTS (
    SELECT 1 FROM worker_leases l
    WHERE l.job_id = j.id AND l.completed = FALSE
  );

-- name: GetPendingJobs :many
SELECT j.* FROM jobs j
JOIN queues q ON j.queue_name = q.name
WHERE j.status = 'pending'
  AND j.scheduled_at <= NOW()
  AND q.paused = FALSE
ORDER BY j.priority DESC, j.scheduled_at ASC;

-- name: ClaimPendingJobs :many
SELECT j.* FROM jobs j
JOIN queues q ON j.queue_name = q.name
LEFT JOIN task_execution te ON te.id = j.task_execution_id
LEFT JOIN task_definition td ON td.id = te.task_base
WHERE j.status = 'pending' AND j.scheduled_at <= NOW() AND q.paused = FALSE
  AND (
    -- Workflow jobs are matched by their task definition's task type;
    -- standalone jobs are matched by their queue name.
    (j.task_execution_id IS NOT NULL AND td.task_type = ANY($1::text[]))
    OR (j.task_execution_id IS NULL AND j.queue_name = ANY($1::text[]))
  )
ORDER BY j.priority DESC, j.scheduled_at ASC
FOR UPDATE OF j SKIP LOCKED LIMIT $2;

-- name: CreateExecution :one
INSERT INTO executions (
    job_id, worker_id, status
) VALUES (
    $1, $2, 'running'
) RETURNING *;

-- name: UpdateExecutionStatus :exec
UPDATE executions
SET status = $1, finished_at = NOW()
WHERE id = $2;

-- name: CreateRetry :one
INSERT INTO retries (
    job_id, execution_id, attempt_number, retry_at, error_reason
) VALUES (
    $1, $2, $3, NOW(), $4
) RETURNING *;

-- name: CreateDeadLetterJob :one
INSERT INTO dead_letter_jobs (
    job_id, error_reason
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetDeadLetterJobs :many
SELECT * FROM dead_letter_jobs ORDER BY failed_at DESC;

-- name: GetActiveJobsCount :one
SELECT count(*) FROM jobs WHERE status = 'running';

-- name: GetQueueDepth :one
SELECT count(*) FROM jobs WHERE queue_name = $1 AND status = 'pending';

-- name: GetActiveExecution :one
SELECT * FROM executions
WHERE job_id = $1 AND worker_id = $2 AND finished_at IS NULL
ORDER BY started_at DESC LIMIT 1;

-- name: GetQueueJobCounts :one
SELECT
    COALESCE(SUM(CASE WHEN j.status = 'pending' THEN 1 ELSE 0 END), 0)::BIGINT AS pending_count,
    COALESCE(SUM(CASE WHEN j.status = 'running' THEN 1 ELSE 0 END), 0)::BIGINT AS running_count,
    COALESCE(SUM(CASE WHEN j.status = 'completed' THEN 1 ELSE 0 END), 0)::BIGINT AS completed_count,
    COALESCE(SUM(CASE WHEN j.status = 'failed' THEN 1 ELSE 0 END), 0)::BIGINT AS failed_count,
    COALESCE(SUM(CASE WHEN j.status = 'dead_letter' THEN 1 ELSE 0 END), 0)::BIGINT AS dead_letter_count
FROM jobs j
WHERE j.queue_name = $1;

-- name: RetryJob :one
UPDATE jobs
SET status = 'pending', retry_count = 0, updated_at = NOW()
WHERE id = $1 AND status IN ('failed', 'dead_letter')
RETURNING *;

-- name: DeleteDeadLetterJob :exec
DELETE FROM dead_letter_jobs WHERE job_id = $1;

-- name: ListWorkersWithStatus :many
SELECT
    w.id,
    w.worker_host,
    (w.last_heartbeat > NOW() - INTERVAL '15 seconds') AS active,
    w.last_heartbeat
FROM worker w
ORDER BY w.last_heartbeat DESC;

