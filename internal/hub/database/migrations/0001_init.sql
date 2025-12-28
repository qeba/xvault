-- +goose Up
-- +goose StatementBegin
-- Enable pgcrypto extension for UUID generation and encryption
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ==================================================================
-- tenants
-- Customer accounts (single-user or org)
-- +goose StatementEnd

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    plan TEXT DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- users
-- User accounts belonging to a tenant
-- +goose StatementBegin

CREATE TYPE user_role AS ENUM ('owner', 'admin', 'member');
-- +goose StatementEnd

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'member',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(email)
);

-- ==================================================================
-- tenant_keys
-- Encryption keypairs per tenant (platform-managed for v0)
-- +goose StatementBegin

CREATE TYPE key_status AS ENUM ('active', 'rotated', 'disabled');
-- +goose StatementEnd

CREATE TABLE tenant_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    algorithm TEXT NOT NULL DEFAULT 'age-x25519',
    public_key TEXT NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    key_status key_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- credentials
-- Encrypted secrets for sources (or storage later)
-- +goose StatementBegin

CREATE TYPE credential_kind AS ENUM ('source', 'storage');
-- +goose StatementEnd

CREATE TABLE credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind credential_kind NOT NULL DEFAULT 'source',
    ciphertext TEXT NOT NULL,
    key_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- sources
-- Things to back up (servers/sites/databases)
-- +goose StatementBegin

CREATE TYPE source_type AS ENUM ('ssh', 'sftp', 'ftp', 'mysql', 'postgres', 'wordpress');
CREATE TYPE source_status AS ENUM ('active', 'disabled');
-- +goose StatementEnd

CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type source_type NOT NULL,
    name TEXT NOT NULL,
    status source_status NOT NULL DEFAULT 'active',
    config JSONB NOT NULL DEFAULT '{}',
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- schedules
-- When to run backups for a source
-- +goose StatementBegin

CREATE TYPE schedule_status AS ENUM ('enabled', 'disabled');
-- +goose StatementEnd

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    cron TEXT,
    interval_minutes INT,
    timezone TEXT DEFAULT 'UTC',
    status schedule_status NOT NULL DEFAULT 'enabled',
    retention_policy JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, source_id)
);

-- ==================================================================
-- workers
-- Registry of data-plane workers (for routing + health)
-- +goose StatementBegin

CREATE TYPE worker_status AS ENUM ('online', 'offline', 'draining');
-- +goose StatementEnd

CREATE TABLE workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status worker_status NOT NULL DEFAULT 'offline',
    capabilities JSONB NOT NULL DEFAULT '{}',
    storage_base_path TEXT NOT NULL DEFAULT '/var/lib/xvault/backups',
    last_seen_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- jobs
-- Queued execution units (backup, restore, delete)
-- +goose StatementBegin

CREATE TYPE job_type AS ENUM ('backup', 'restore', 'delete_snapshot');
CREATE TYPE job_status AS ENUM ('queued', 'running', 'finalizing', 'completed', 'failed', 'canceled');
-- +goose StatementEnd

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id UUID REFERENCES sources(id) ON DELETE SET NULL,
    type job_type NOT NULL,
    status job_status NOT NULL DEFAULT 'queued',
    priority INT NOT NULL DEFAULT 0,
    target_worker_id UUID REFERENCES workers(id) ON DELETE SET NULL,
    lease_expires_at TIMESTAMP,
    attempt INT NOT NULL DEFAULT 0,
    payload JSONB NOT NULL DEFAULT '{}',
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    error_code TEXT,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- snapshots
-- Completed (or failed) backup results
-- +goose StatementBegin

CREATE TYPE snapshot_status AS ENUM ('completed', 'failed');
CREATE TYPE storage_backend AS ENUM ('local_fs', 's3');
-- +goose StatementEnd

CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE RESTRICT,
    status snapshot_status NOT NULL DEFAULT 'completed',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMP NOT NULL DEFAULT now(),
    finished_at TIMESTAMP NOT NULL DEFAULT now(),
    duration_ms BIGINT,
    manifest_json JSONB,
    encryption_algorithm TEXT NOT NULL DEFAULT 'age-x25519',
    encryption_key_id UUID REFERENCES tenant_keys(id) ON DELETE SET NULL,
    encryption_recipient TEXT,
    storage_backend storage_backend NOT NULL DEFAULT 'local_fs',
    worker_id UUID REFERENCES workers(id) ON DELETE SET NULL,
    local_path TEXT,
    bucket TEXT,
    object_key TEXT,
    etag TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- audit_events
-- Minimal audit trail for sensitive actions
-- +goose StatementBegin

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id UUID,
    ip TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ==================================================================
-- Indexes
-- +goose StatementBegin

-- users indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- tenant_keys indexes
CREATE INDEX idx_tenant_keys_tenant_id ON tenant_keys(tenant_id);
CREATE INDEX idx_tenant_keys_status ON tenant_keys(key_status);

-- credentials indexes
CREATE INDEX idx_credentials_tenant_id ON credentials(tenant_id);
CREATE INDEX idx_credentials_kind ON credentials(kind);

-- sources indexes
CREATE INDEX idx_sources_tenant_id ON sources(tenant_id);
CREATE INDEX idx_sources_status ON sources(tenant_id, status);
CREATE INDEX idx_sources_credential_id ON sources(credential_id);

-- schedules indexes
CREATE INDEX idx_schedules_tenant_id ON schedules(tenant_id);
CREATE INDEX idx_schedules_source_id ON schedules(source_id);

-- workers indexes
CREATE INDEX idx_workers_status ON workers(status, last_seen_at);

-- jobs indexes
CREATE INDEX idx_jobs_status_priority ON jobs(status, priority, created_at);
CREATE INDEX idx_jobs_tenant_id ON jobs(tenant_id, created_at);
CREATE INDEX idx_jobs_target_worker_id ON jobs(target_worker_id);
CREATE INDEX idx_jobs_source_id ON jobs(source_id);
CREATE INDEX idx_jobs_lease_expires_at ON jobs(lease_expires_at) WHERE lease_expires_at IS NOT NULL;

-- snapshots indexes
CREATE INDEX idx_snapshots_tenant_source_created ON snapshots(tenant_id, source_id, created_at DESC);
CREATE INDEX idx_snapshots_job_id ON snapshots(job_id);
CREATE INDEX idx_snapshots_worker_id ON snapshots(worker_id);
CREATE INDEX idx_snapshots_storage_backend ON snapshots(storage_backend);

-- audit_events indexes
CREATE INDEX idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX idx_audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_user_id);
-- +goose StatementEnd

-- +goose Down
-- Drop indexes first
DROP INDEX IF EXISTS idx_audit_events_actor;
DROP INDEX IF EXISTS idx_audit_events_created_at;
DROP INDEX IF EXISTS idx_audit_events_tenant_id;
DROP INDEX IF EXISTS idx_snapshots_storage_backend;
DROP INDEX IF EXISTS idx_snapshots_worker_id;
DROP INDEX IF EXISTS idx_snapshots_job_id;
DROP INDEX IF EXISTS idx_snapshots_tenant_source_created;
DROP INDEX IF EXISTS idx_jobs_lease_expires_at;
DROP INDEX IF EXISTS idx_jobs_source_id;
DROP INDEX IF EXISTS idx_jobs_target_worker_id;
DROP INDEX IF EXISTS idx_jobs_tenant_id;
DROP INDEX IF EXISTS idx_jobs_status_priority;
DROP INDEX IF EXISTS idx_workers_status;
DROP INDEX IF EXISTS idx_schedules_source_id;
DROP INDEX IF EXISTS idx_schedules_tenant_id;
DROP INDEX IF EXISTS idx_sources_credential_id;
DROP INDEX IF EXISTS idx_sources_status;
DROP INDEX IF EXISTS idx_sources_tenant_id;
DROP INDEX IF EXISTS idx_credentials_kind;
DROP INDEX IF EXISTS idx_credentials_tenant_id;
DROP INDEX IF EXISTS idx_tenant_keys_status;
DROP INDEX IF EXISTS idx_tenant_keys_tenant_id;
DROP INDEX IF EXISTS idx_users_tenant_id;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables (in correct order due to foreign keys)
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS workers;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS tenant_keys;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- Drop custom types
DROP TYPE IF EXISTS storage_backend CASCADE;
DROP TYPE IF EXISTS snapshot_status CASCADE;
DROP TYPE IF EXISTS job_status CASCADE;
DROP TYPE IF EXISTS job_type CASCADE;
DROP TYPE IF EXISTS worker_status CASCADE;
DROP TYPE IF EXISTS schedule_status CASCADE;
DROP TYPE IF EXISTS key_status CASCADE;
DROP TYPE IF EXISTS credential_kind CASCADE;
DROP TYPE IF EXISTS source_status CASCADE;
DROP TYPE IF EXISTS source_type CASCADE;
DROP TYPE IF EXISTS user_role CASCADE;

-- Drop extension
DROP EXTENSION IF EXISTS pgcrypto;
