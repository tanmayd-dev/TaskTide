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

-- name: AddTaskDependency :one
INSERT INTO task_definition_depends (
    task_id,
    task_depends_on
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetTaskDependencies :many
SELECT * FROM task_definition_depends
WHERE task_id = $1;