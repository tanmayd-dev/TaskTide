-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workflow_definition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_name TEXT
);

CREATE TABLE task_definition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_type TEXT,
    task_name TEXT,
    workflow_id UUID REFERENCES workflow_definition(id)
);
-- +goose Down
