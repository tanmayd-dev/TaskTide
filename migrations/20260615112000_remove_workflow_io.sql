-- +goose Up
ALTER TABLE workflow_definition DROP COLUMN workflow_input;
ALTER TABLE workflow_definition DROP COLUMN workflow_output;

-- +goose Down
ALTER TABLE workflow_definition ADD COLUMN workflow_input UUID REFERENCES input_file(id);
ALTER TABLE workflow_definition ADD COLUMN workflow_output UUID REFERENCES input_file(id);
