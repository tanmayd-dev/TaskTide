-- name: CreateTaskAttempt :one
INSERT INTO task_attempt (
    task_execution_base,
    task_completed,
    task_succeeded,
    task_input
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetTaskAttemptsCount :one
SELECT COUNT(*) FROM task_attempt WHERE task_execution_base = $1;

