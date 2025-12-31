-- +goose Up
-- +goose StatementBegin
ALTER TABLE schedules ADD COLUMN last_run_at TIMESTAMP;
ALTER TABLE schedules ADD COLUMN next_run_at TIMESTAMP;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_schedules_next_run ON schedules(next_run_at) WHERE status = 'enabled';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_schedules_next_run;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE schedules DROP COLUMN IF EXISTS last_run_at;
ALTER TABLE schedules DROP COLUMN IF EXISTS next_run_at;
-- +goose StatementEnd
