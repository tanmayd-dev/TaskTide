-- +goose Up
CREATE TABLE queues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    paused BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    queue_name TEXT REFERENCES queues(name) ON DELETE RESTRICT NOT NULL,
    payload JSONB NOT NULL,
    max_retries INT NOT NULL DEFAULT 3,
    retry_count INT NOT NULL DEFAULT 0,
    delay INT NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMP NOT NULL DEFAULT NOW(),
    priority INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'dead_letter'
    idempotency_key TEXT UNIQUE,
    task_execution_id UUID REFERENCES task_execution(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE worker_leases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE UNIQUE,
    worker_id TEXT NOT NULL,
    leased_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    worker_id TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMP,
    status TEXT NOT NULL -- 'succeeded', 'failed'
);

CREATE TABLE retries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES executions(id) ON DELETE CASCADE,
    attempt_number INT NOT NULL,
    retry_at TIMESTAMP NOT NULL,
    error_reason TEXT
);

CREATE TABLE dead_letter_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    failed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    error_reason TEXT
);

-- +goose Down
DROP TABLE dead_letter_jobs CASCADE;
DROP TABLE retries CASCADE;
DROP TABLE executions CASCADE;
DROP TABLE worker_leases CASCADE;
DROP TABLE jobs CASCADE;
DROP TABLE queues CASCADE;
