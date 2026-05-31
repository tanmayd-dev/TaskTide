-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workflow_definition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_name TEXT
);

CREATE TABLE task_definition (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_type TEXT NOT NULL,
    task_name TEXT,
    workflow_id UUID REFERENCES workflow_definition(id) NOT NULL
);

CREATE TABLE task_definition_depends (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES task_definition(id) NOT NULL,
    task_depends_on UUID REFERENCES task_definition(id) NOT NULL
);

-- +goose Down
DROP TABLE task_definition CASCADE;
DROP TABLE workflow_definition CASCADE;
DROP TABLE task_definition_depends CASCADE;
