-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE input_file (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_name TEXT
);

-- +goose Down
DROP TABLE input_file CASCADE;
