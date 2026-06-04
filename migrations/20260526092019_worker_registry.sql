-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE worker (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_host TEXT NOT NULL,
    worker_port TEXT NOT NULL,
    worker_available BOOL NOT NULL
);

CREATE TABLE task_execution (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_base UUID REFERENCES task_definition(id) NOT NULL,
    task_worker UUID REFERENCES worker(id),
    task_assigned BOOL NOT NULL,
    task_completed BOOL NOT NULL,
    task_input UUID REFERENCES input_file(id),
    task_output UUID REFERENCES input_file(id)
);

CREATE TABLE task_execution_depends (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES task_execution(id) NOT NULL,
    task_depends_on UUID REFERENCES task_execution(id) NOT NULL
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE worker CASCADE;
DROP TABLE task_execution CASCADE;
DROP TABLE task_execution_depends CASCADE;
