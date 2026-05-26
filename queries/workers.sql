-- name: AddWorker :one
INSERT INTO worker (
    worker_host,
    worker_port
) VALUES (
    $1, $2
) RETURNING *;

-- name: AssignWorkerToTask :one
INSERT INTO task_execution (
    task_base,
    task_worker
) VALUES (
    $1, $2
) RETURNING *;
