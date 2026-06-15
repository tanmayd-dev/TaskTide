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

-- name: CreateTask :one
INSERT INTO task_definition (
    task_name,
    task_type,
    workflow_id,
    task_input_file,
    task_input_task,
    task_output
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetTask :one
SELECT * FROM task_definition WHERE
id = $1
LIMIT 1;

-- name: GetWorkflowTasks :many
SELECT * FROM task_definition WHERE (
    workflow_id = $1
);

-- name: AddTaskDependency :one
INSERT INTO task_definition_depends (
    task_id,
    task_depends_on
) VALUES (
    $1, $2
) RETURNING *;

-- name: UpdateTaskInputTask :exec
UPDATE task_definition SET
task_input_task = $2
WHERE id = $1
RETURNING *;

-- name: GetTaskDependencies :many
SELECT * FROM task_definition_depends
WHERE task_id = $1;

-- name: ListWorkflows :many
SELECT * FROM workflow_definition;
