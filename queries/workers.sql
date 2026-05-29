-- name: AddWorker :one
INSERT INTO worker (
    worker_host,
    worker_port,
    worker_available
) VALUES (
    $1, $2, TRUE
) RETURNING *;

-- name: CreateTaskExecution :one
INSERT INTO task_execution (
    task_base,
    task_worker,
    task_assigned,
    task_completed
) VALUES (
    $1, $2, FALSE, FALSE
) RETURNING *;

-- name: MarkWorkerUnavailable :one
UPDATE worker
SET worker_available = FALSE
WHERE id = $1
RETURNING *;

-- name: AssignWorkerToTask :one
UPDATE task_execution
SET task_assigned = TRUE,
task_worker = $1
WHERE id = $2
RETURNING *;

-- name: GetAvailableWorker :one
SELECT * FROM worker WHERE (
    worker_available = TRUE
) LIMIT 1;

-- name: GetUnassignedTasks :one
SELECT * FROM task_execution WHERE (
    task_assigned = FALSE AND
    task_completed = FALSE
) LIMIT 1;

-- name: GetTasksExecution :many
SELECT * FROM task_execution WHERE (
    id = $1
);

-- name: MarkTaskComplete :exec
UPDATE task_execution SET
task_completed = TRUE,
task_assigned = FALSE
WHERE (
    id = $1
);

-- name: MarkWorkerAvailable :exec
UPDATE worker SET
worker_available = TRUE
WHERE (
    id = $1
);