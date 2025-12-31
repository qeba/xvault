-- +goose Up
-- +goose StatementBegin
CREATE TYPE log_level AS ENUM ('debug', 'info', 'warn', 'error');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP NOT NULL DEFAULT now(),
    level log_level NOT NULL,
    message TEXT NOT NULL,
    worker_id TEXT,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    snapshot_id UUID REFERENCES snapshots(id) ON DELETE SET NULL,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX idx_logs_job_id ON logs(job_id);
CREATE INDEX idx_logs_snapshot_id ON logs(snapshot_id);
CREATE INDEX idx_logs_source_id ON logs(source_id);
CREATE INDEX idx_logs_schedule_id ON logs(schedule_id);
CREATE INDEX idx_logs_level ON logs(level);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_logs_level;
DROP INDEX IF EXISTS idx_logs_schedule_id;
DROP INDEX IF EXISTS idx_logs_source_id;
DROP INDEX IF EXISTS idx_logs_snapshot_id;
DROP INDEX IF EXISTS idx_logs_job_id;
DROP INDEX IF EXISTS idx_logs_timestamp;
DROP TABLE IF EXISTS logs CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TYPE IF EXISTS log_level CASCADE;
-- +goose StatementEnd
