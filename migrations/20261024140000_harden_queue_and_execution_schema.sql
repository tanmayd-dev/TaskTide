-- +goose Up
-- Migrate timestamp columns of job-queue tables to TIMESTAMPTZ
ALTER TABLE queues ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE jobs ALTER COLUMN scheduled_at TYPE TIMESTAMPTZ;
ALTER TABLE jobs ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE jobs ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE worker_leases ALTER COLUMN leased_at TYPE TIMESTAMPTZ;
ALTER TABLE worker_leases ALTER COLUMN expires_at TYPE TIMESTAMPTZ;

ALTER TABLE executions ALTER COLUMN started_at TYPE TIMESTAMPTZ;
ALTER TABLE executions ALTER COLUMN finished_at TYPE TIMESTAMPTZ;

ALTER TABLE retries ALTER COLUMN retry_at TYPE TIMESTAMPTZ;

ALTER TABLE dead_letter_jobs ALTER COLUMN failed_at TYPE TIMESTAMPTZ;

-- Seed default queue
INSERT INTO queues (name, paused) VALUES ('default', FALSE) ON CONFLICT (name) DO NOTHING;

-- Add explicit task failure state
ALTER TABLE task_execution ADD COLUMN task_failed BOOLEAN NOT NULL DEFAULT FALSE;

-- Composite and partial indexes for hot paths
CREATE INDEX idx_jobs_pending_claim ON jobs (queue_name, priority DESC, scheduled_at ASC) WHERE status = 'pending';
CREATE INDEX idx_worker_leases_active ON worker_leases (expires_at) WHERE completed = FALSE;
CREATE INDEX idx_worker_leases_worker ON worker_leases (worker_id) WHERE completed = FALSE;
CREATE INDEX idx_executions_active ON executions (job_id, worker_id) WHERE finished_at IS NULL;
CREATE INDEX idx_task_execution_wf_lookup ON task_execution (workflow_id, task_base);

-- +goose Down
DROP INDEX IF EXISTS idx_task_execution_wf_lookup;
DROP INDEX IF EXISTS idx_executions_active;
DROP INDEX IF EXISTS idx_worker_leases_worker;
DROP INDEX IF EXISTS idx_worker_leases_active;
DROP INDEX IF EXISTS idx_jobs_pending_claim;

ALTER TABLE task_execution DROP COLUMN IF EXISTS task_failed;

DELETE FROM queues WHERE name = 'default';

ALTER TABLE dead_letter_jobs ALTER COLUMN failed_at TYPE TIMESTAMP;
ALTER TABLE retries ALTER COLUMN retry_at TYPE TIMESTAMP;
ALTER TABLE executions ALTER COLUMN finished_at TYPE TIMESTAMP;
ALTER TABLE executions ALTER COLUMN started_at TYPE TIMESTAMP;
ALTER TABLE worker_leases ALTER COLUMN expires_at TYPE TIMESTAMP;
ALTER TABLE worker_leases ALTER COLUMN leased_at TYPE TIMESTAMP;
ALTER TABLE jobs ALTER COLUMN updated_at TYPE TIMESTAMP;
ALTER TABLE jobs ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE jobs ALTER COLUMN scheduled_at TYPE TIMESTAMP;
ALTER TABLE queues ALTER COLUMN created_at TYPE TIMESTAMP;
