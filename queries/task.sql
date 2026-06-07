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