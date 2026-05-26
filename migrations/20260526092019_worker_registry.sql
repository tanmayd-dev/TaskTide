-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE worker (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    worker_host TEXT,
    worker_port TEXT
);

CREATE TABLE task_execution (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_base UUID REFERENCES task_definition(id),
    task_worker INT REFERENCES worker(id)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE worker CASCADE;
DROP TABLE task_execution CASCADE;