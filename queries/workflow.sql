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

-- name: UpdateTaskOutput :exec
UPDATE task_definition
SET task_output = $2
WHERE id = $1;

-- name: UpdateWorkflowOutput :exec
UPDATE workflow_definition
SET workflow_output = $2
WHERE id = $1;
