-- name: DeleteWorker :exec
DELETE FROM worker WHERE (
    id = $1
);
-- name: RegisterWorkerWithMetadata :one
INSERT INTO worker (
    worker_host,
    worker_port,
    worker_available,
    concurrency,
    capabilities,
    last_heartbeat
) VALUES (
    $1, $2, TRUE, $3, $4, NOW()
) RETURNING *;

-- name: UpdateWorkerHeartbeat :exec
UPDATE worker
SET last_heartbeat = NOW()
WHERE id = $1;

-- name: GetActiveWorkers :many
SELECT * FROM worker
WHERE last_heartbeat >= $1;

-- name: GetDeadWorkers :many
SELECT * FROM worker
WHERE last_heartbeat < $1;
