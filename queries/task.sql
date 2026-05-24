-- name: CreateTask :one
INSERT INTO task_definition (
    task_name,
    task_type,
    workflow_id
) VALUES (
    $1, $2, $3
) RETURNING *;
