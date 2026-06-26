-- name: AddWorker :one
INSERT INTO worker (
    worker_host,
    worker_port,
    worker_available
) VALUES (
    $1, $2, TRUE
) RETURNING *;

-- name: MarkWorkerUnavailable :one
UPDATE worker
SET worker_available = FALSE
WHERE id = $1
  AND worker_available = TRUE
RETURNING *;

-- name: AssignWorkerToTask :one
UPDATE task_execution
SET task_assigned = TRUE,
    task_worker = $1
WHERE id = $2
  AND task_assigned = FALSE
RETURNING *;

-- name: GetAvailableWorker :one
SELECT * FROM worker WHERE (
    worker_available = TRUE
) LIMIT 1;

-- name: MarkWorkerAvailable :exec
UPDATE worker SET
worker_available = TRUE
WHERE (
    id = $1
);

-- name: DeleteWorker :exec
DELETE FROM worker WHERE (
    id = $1
);
