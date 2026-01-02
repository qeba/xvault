-- +goose Up
-- +goose StatementBegin
ALTER TABLE workers ADD COLUMN IF NOT EXISTS system_metrics JSONB NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON COLUMN workers.system_metrics IS 'System metrics reported by worker: cpu_percent, memory_percent, memory_total_bytes, memory_used_bytes, disk_total_bytes, disk_used_bytes, disk_free_bytes, disk_percent, active_jobs, uptime_seconds';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE workers DROP COLUMN IF EXISTS system_metrics;
-- +goose StatementEnd
