-- +goose Up
-- Track download tokens and expiration

-- Add download tracking to snapshots
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS download_token TEXT;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS download_expires_at TIMESTAMP;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS download_url TEXT;

-- Add index for token lookups
CREATE INDEX IF NOT EXISTS idx_snapshots_download_token ON snapshots(download_token) WHERE download_token IS NOT NULL;

-- Create system settings table for configuration
CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Insert default settings
INSERT INTO system_settings (key, value, description) VALUES
    ('download_expiration_hours', '1', 'Default download link expiration time in hours'),
    ('max_download_size_mb', '5000', 'Maximum size for downloads in MB')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS system_settings;
DROP INDEX IF EXISTS idx_snapshots_download_token;
ALTER TABLE snapshots DROP COLUMN IF EXISTS download_url;
ALTER TABLE snapshots DROP COLUMN IF EXISTS download_expires_at;
ALTER TABLE snapshots DROP COLUMN IF EXISTS download_token;
