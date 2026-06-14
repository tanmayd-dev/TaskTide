-- name: CreateTaskAttempt :one
INSERT INTO task_attempt (
    task_execution_base,
    task_completed,
    task_succeeded,
    task_input
) VALUES (
    $1, FALSE, FALSE, $2
) RETURNING *;

-- name: SucceedTask :one
UPDATE task_attempt SET
task_completed = TRUE,
task_succeeded = TRUE
WHERE id = $1
RETURNING *;

-- name: FailTask :one
UPDATE task_attempt SET
task_completed = TRUE,
task_succeeded = FALSE
WHERE id = $1
RETURNING *;
