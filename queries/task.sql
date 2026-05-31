-- name: CreateTask :one
INSERT INTO task_definition (
    task_name,
    task_type,
    workflow_id
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetTask :one
SELECT * FROM task_definition WHERE
id = $1
LIMIT 1;

-- name: GetWorkflowTasks :many
SELECT * FROM task_definition WHERE (
    workflow_id = $1
);

-- name: CreateTaskAttempt :one
INSERT INTO task_attempt(
    task_execution_base,
    task_completed,
    task_succeded
) VALUES (
    $1, FALSE, FALSE
) RETURNING *;
