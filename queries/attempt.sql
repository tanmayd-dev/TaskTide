-- name: CreateTaskAttempt :one
INSERT INTO task_attempt (
    task_execution_base,
    task_completed,
    task_succeeded,
    task_input
) VALUES (
    $1, $2, $3, $4
) RETURNING *;
