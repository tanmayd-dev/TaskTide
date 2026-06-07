-- name: AddWorker :one
INSERT INTO worker (
    worker_host,
    worker_port,
    worker_available
) VALUES (
    $1, $2, TRUE
) RETURNING *;

-- name: CreateWorkflowExecution :one
INSERT INTO workflow_execution (
    workflow_base
) VALUES (
    $1
) RETURNING *;

-- name: CreateTaskExecution :one
INSERT INTO task_execution (
    task_base,
    task_worker,
    task_assigned,
    task_completed,
    task_input,
    workflow_id
) VALUES (
    $1, $2, FALSE, FALSE, $3, $4
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
SELECT * FROM task_execution A
INNER JOIN task_definition_depends ON task_definition_depends.id = A.task_base
INNER JOIN task_execution B ON A.workflow_id = B.workflow_id AND B.task_base = task_definition_depends.task_depends_on
WHERE (
    A.task_assigned = FALSE AND
    A.task_completed = FALSE AND
    B.task_completed = TRUE
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

-- name: MarkTaskAttemptSuccess :exec
UPDATE task_attempt SET
task_completed = TRUE,
task_succeeded = TRUE
WHERE (
    id = $1
);

-- name: MarkTaskAttemptFailure :exec
UPDATE task_attempt SET
task_completed = TRUE,
task_succeeded = FALSE
WHERE (
    id = $1
);
