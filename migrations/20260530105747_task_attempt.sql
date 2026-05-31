-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE task_attempt (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_execution_base UUID REFERENCES task_execution(id) NOT NULL,
    task_completed BOOL NOT NULL,
    task_succeded BOOL NOT NULL
);

-- +goose Down
DROP TABLE task_attempt CASCADE;
