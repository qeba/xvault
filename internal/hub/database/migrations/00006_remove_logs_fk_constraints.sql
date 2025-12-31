-- +goose Up
-- Remove foreign key constraints from logs table to allow logs to reference
-- entities that may not exist yet (e.g., snapshot_id before snapshot is created)
-- or entities that may be deleted later.
-- The indexes are kept for query performance.

-- +goose StatementBegin
-- First, drop the existing foreign key constraints
ALTER TABLE logs DROP CONSTRAINT IF EXISTS logs_job_id_fkey;
ALTER TABLE logs DROP CONSTRAINT IF EXISTS logs_snapshot_id_fkey;
ALTER TABLE logs DROP CONSTRAINT IF EXISTS logs_source_id_fkey;
ALTER TABLE logs DROP CONSTRAINT IF EXISTS logs_schedule_id_fkey;

-- Change the column types from UUID to TEXT to allow storing IDs without FK validation
-- This also handles the case where the worker sends a snapshot_id before the snapshot record exists
ALTER TABLE logs ALTER COLUMN job_id TYPE TEXT;
ALTER TABLE logs ALTER COLUMN snapshot_id TYPE TEXT;
ALTER TABLE logs ALTER COLUMN source_id TYPE TEXT;
ALTER TABLE logs ALTER COLUMN schedule_id TYPE TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Convert columns back to UUID and re-add foreign key constraints
-- Note: This may fail if there are logs with IDs that don't exist in the referenced tables
ALTER TABLE logs ALTER COLUMN job_id TYPE UUID USING job_id::UUID;
ALTER TABLE logs ALTER COLUMN snapshot_id TYPE UUID USING snapshot_id::UUID;
ALTER TABLE logs ALTER COLUMN source_id TYPE UUID USING source_id::UUID;
ALTER TABLE logs ALTER COLUMN schedule_id TYPE UUID USING schedule_id::UUID;

ALTER TABLE logs ADD CONSTRAINT logs_job_id_fkey FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL;
ALTER TABLE logs ADD CONSTRAINT logs_snapshot_id_fkey FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE SET NULL;
ALTER TABLE logs ADD CONSTRAINT logs_source_id_fkey FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE SET NULL;
ALTER TABLE logs ADD CONSTRAINT logs_schedule_id_fkey FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE SET NULL;
-- +goose StatementEnd
