-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE worker (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    worker_host TEXT,
    worker_port TEXT,
    worker_available BOOL
);

CREATE TABLE task_execution (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_base UUID REFERENCES task_definition(id),
    task_worker INT REFERENCES worker(id),
    task_assigned BOOL,
    task_completed BOOL
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE worker CASCADE;
DROP TABLE task_execution CASCADE;