-- name: CreateAttempt :one
INSERT INTO task_attempt (
    task_execution_base,
    task_completed,
    task_succeded
) VALUES (
    $1, FALSE, FALSE
) RETURNING *;

-- name: SucceedTask :one
UPDATE task_attempt SET
task_completed = TRUE,
task_succeded = TRUE
WHERE id = $1
RETURNING *;

-- name: FailTask :one
UPDATE task_attempt SET
task_completed = TRUE,
task_succeded = FALSE
WHERE id = $1
RETURNING *;
