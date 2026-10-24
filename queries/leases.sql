-- name: AcquireLease :one
INSERT INTO worker_leases (job_id, worker_id, expires_at)
VALUES ($1, $2, NOW() + sqlc.arg(lease_duration)::INTERVAL)
ON CONFLICT (job_id) DO UPDATE SET
  worker_id = EXCLUDED.worker_id, expires_at = EXCLUDED.expires_at,
  leased_at = NOW(), completed = FALSE
WHERE worker_leases.completed = TRUE OR worker_leases.expires_at < NOW()
RETURNING *;

-- name: GetLease :one
SELECT * FROM worker_leases WHERE job_id = $1;

-- name: GetActiveLease :one
SELECT * FROM worker_leases WHERE job_id = $1 AND completed = FALSE;

-- name: ExtendLease :one
UPDATE worker_leases SET expires_at = NOW() + sqlc.arg(extension_interval)::INTERVAL
WHERE job_id = sqlc.arg(job_id) AND worker_id = sqlc.arg(worker_id) AND completed = FALSE
RETURNING *;

-- name: ReleaseLease :exec
DELETE FROM worker_leases WHERE job_id = $1;

-- name: GetExpiredLeases :many
SELECT * FROM worker_leases
WHERE expires_at < NOW()
  AND completed = FALSE;

-- name: MarkLeaseCompleted :exec
UPDATE worker_leases
SET completed = TRUE
WHERE job_id = $1;

-- name: GetLeasesByWorker :many
SELECT * FROM worker_leases WHERE worker_id = $1 AND completed = FALSE;
