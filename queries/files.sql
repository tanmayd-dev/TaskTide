-- name: CreateFile :one
INSERT INTO input_file (
    file_name
) VALUES (
    $1
) RETURNING *;

-- name: DeleteFile :exec
DELETE FROM input_file WHERE (
    id = $1
);

-- name: GetFile :many
SELECT * FROM input_file WHERE (
    id = $1
);