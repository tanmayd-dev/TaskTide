-- name: CreateWorkflow :one
INSERT INTO workflow_definition (
    workflow_name
) VALUES (
    $1
) RETURNING *;
