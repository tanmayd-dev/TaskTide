-- name: CreateWorkflowExecution :one
INSERT INTO workflow_execution (
    workflow_base
) VALUES (
    $1
) RETURNING *;

-- name: GetWorkflowExecution :one
SELECT * FROM workflow_execution WHERE (
    id = $1
);

-- name: CreateTaskExecution :one
INSERT INTO task_execution (
    task_base,
    task_worker,
    task_assigned,
    task_completed,
    task_input,
    task_output,
    workflow_id
) VALUES (
    $1, $2, FALSE, FALSE, $3, $4, $5
) RETURNING *;

-- name: GetTaskExecution :one
SELECT * FROM task_execution
WHERE id =  $1
LIMIT 1;

-- name: GetUnassignedTasks :one
SELECT id, task_base, task_worker, task_assigned, task_completed, task_input, task_output, workflow_id FROM task_execution A
WHERE (
    A.task_assigned = FALSE AND
    A.task_completed = FALSE AND
    NOT EXISTS (
        SELECT 1 FROM task_definition_depends d
        LEFT JOIN task_execution dep
            ON dep.workflow_id = A.workflow_id
            AND dep.task_base = d.task_depends_on
        WHERE d.task_id = A.task_base
          AND (dep.task_completed = FALSE OR dep.id IS NULL)
    )
) LIMIT 1;

-- name: GetTasksExecution :many
SELECT * FROM task_execution WHERE (
    id = $1
);

-- name: MarkTaskComplete :exec
UPDATE task_execution SET
task_completed = TRUE,
task_assigned = FALSE
WHERE (
    id = $1
);

-- name: GetWorkflowTaskExecutions :many
SELECT * FROM task_execution WHERE (
    workflow_id = $1
);

-- name: UnassignTask :exec
UPDATE task_execution SET
task_assigned = FALSE,
task_worker = NULL
WHERE (
    id = $1
);
