-- name: CreateWorkflow :one
INSERT INTO workflow_definition (
    workflow_name
) VALUES (
    $1
) RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM workflow_definition WHERE (
    id = $1
);