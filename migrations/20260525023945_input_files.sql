-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE input_file (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_name TEXT NOT NULL
);

ALTER TABLE workflow_definition
ADD COLUMN workflow_input UUID REFERENCES input_file(id),
ADD COLUMN workflow_output UUID REFERENCES input_file(id);

ALTER TABLE task_definition
ADD COLUMN task_input_file UUID REFERENCES input_file(id),
ADD COLUMN task_input_task UUID REFERENCES task_definition(id),
ADD COLUMN task_output UUID REFERENCES input_file(id);

-- +goose Down
DROP TABLE input_file CASCADE;
