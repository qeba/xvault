# Hub Data Model (Minimal v0)

**Purpose:** This is the smallest relational model that supports multi-tenant xVault backups with worker-local storage (v0) and a clean migration to S3/Garage later.

## Principles

- **Every row is tenant-scoped** (either directly via `tenant_id` or indirectly via a parent).
- **IDs are opaque** (UUID/ULID). Never derive from user input.
- **Hub stores metadata only**. Backup bytes remain on Workers (v0).
- **Snapshots are addressed canonically** by `(tenant_id, source_id, snapshot_id)`.
- **Location is abstracted** by a snapshot locator (`storage_backend` + backend-specific fields).

## Recommended ID Types

- `tenant_id`, `user_id`, `source_id`, `schedule_id`, `job_id`, `snapshot_id`, `worker_id`: UUID (or ULID).

## Tables

### `tenants`

Represents a customer account (single-user or org).

- `id` (PK)
- `name` (display only)
- `plan` (optional: free/pro)
- `created_at`, `updated_at`

### `users`

Users belong to a tenant.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `email` (unique, normalized)
- `password_hash` (or external auth later)
- `role` (e.g., owner/admin/member)
- `created_at`, `updated_at`

Indexes/constraints:
- Unique: `(email)`

### `sources`

A “thing to back up” (server/site/db). One tenant has many sources.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `type` (enum/string: `ssh`, `sftp`, `ftp`, `mysql`, `postgres`, `wordpress`)
- `name` (display only)
- `status` (enum: `active`, `disabled`)
- `config` (JSONB: host, port, paths, db name, etc — non-secret)
- `credential_id` (FK → `credentials.id`)
- `created_at`, `updated_at`

Indexes/constraints:
- Index: `(tenant_id, status)`

### `credentials`

Encrypted secrets for a source (or storage later). Keep minimal and strict.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `kind` (enum/string: `source`, later `storage`)
- `ciphertext` (bytes/base64)
- `key_id` (which KEK/version encrypted it)
- `created_at`, `updated_at`

Notes:
- Never put plaintext secrets into Redis job payloads.
- Worker fetches/decrypts credentials at job start (JIT).

### `tenant_keys` (v0 platform-managed)

Represents the encryption identity used to encrypt snapshot artifacts for a tenant.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `algorithm` (string: e.g., `age-x25519`)
- `public_key` (text)
- `encrypted_private_key` (bytes/base64)
- `key_status` (enum: `active`, `rotated`, `disabled`)
- `created_at`, `updated_at`

Notes:
- v0: platform stores the private key encrypted at rest so it can perform restores.
- later: allow a tenant-provided public key where the platform does not hold the private key.

### `schedules`

Defines when to run backups for a source.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `source_id` (FK → `sources.id`)
- `cron` (string) or `interval_minutes`
- `timezone` (string)
- `enabled` (bool)
- `retention_policy` (JSONB: keep last N, keep daily for X, etc)
- `created_at`, `updated_at`

Indexes/constraints:
- Unique (optional): `(tenant_id, source_id)` if you only want one schedule per source

### `workers`

Registry of data-plane workers (for routing + health).

- `id` (PK) (this is `worker_id`)
- `name` (display only)
- `status` (enum: `online`, `offline`, `draining`)
- `capabilities` (JSONB: supported connectors, max concurrency)
- `storage_base_path` (string, e.g., `/var/lib/xvault/backups`)
- `last_seen_at`
- `created_at`, `updated_at`

Indexes:
- Index: `(status, last_seen_at)`

### `jobs`

A queued execution unit (backup or restore). Jobs are what the Worker actually runs.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `source_id` (FK → `sources.id`, nullable for some job types)
- `type` (enum: `backup`, `restore`, `delete_snapshot`)
- `status` (enum: `queued`, `running`, `finalizing`, `completed`, `failed`, `canceled`)
- `priority` (int)
- `target_worker_id` (FK → `workers.id`, nullable; required for local snapshot restore/delete)
- `lease_expires_at` (for safe retries)
- `attempt` (int)
- `payload` (JSONB: non-secret inputs; references to credentials)
- `started_at`, `finished_at`
- `error_code` (short string)
- `error_message` (text)
- `created_at`, `updated_at`

Indexes:
- `(status, priority, created_at)`
- `(tenant_id, created_at)`

### `snapshots`

One completed (or failed) backup result, referenced by the UI.

- `id` (PK) (`snapshot_id`)
- `tenant_id` (FK → `tenants.id`)
- `source_id` (FK → `sources.id`)
- `job_id` (FK → `jobs.id`)
- `status` (enum: `completed`, `failed`)
- `size_bytes` (bigint)
- `started_at`, `finished_at`, `duration_ms`
- `manifest_json` (JSONB, optional; or store a path)

- `manifest_json` (JSONB, optional; or store a path)

Encryption metadata:

- `encryption_algorithm` (string)
- `encryption_key_id` (FK → `tenant_keys.id`, nullable if using a different backend later)
- `encryption_recipient` (string, optional: key fingerprint / recipient id)
-
- Locator fields (v0 local):
  - `storage_backend` (enum: `local_fs`, later `s3`)
  - `worker_id` (FK → `workers.id`)
  - `local_path` (string)
-
- Locator fields (later S3/Garage):
  - `bucket` (string)
  - `object_key` (string)
  - `etag` (string)
-
- `created_at`, `updated_at`

Indexes/constraints:
- Unique: `(tenant_id, source_id, id)` (usually implied by PK + FKs)
- Index: `(tenant_id, source_id, created_at)`

### `audit_events` (recommended even in v0)

Minimal audit trail for sensitive actions.

- `id` (PK)
- `tenant_id` (FK → `tenants.id`)
- `actor_user_id` (FK → `users.id`, nullable for system)
- `action` (string: `source.create`, `backup.run`, `credential.update`, etc)
- `target_type` / `target_id`
- `ip` (optional)
- `created_at`

## Relationships (Summary)

- `tenant` has many `users`, `sources`, `schedules`, `credentials`, `jobs`, `snapshots`
- `source` has many `schedules`, `jobs`, `snapshots`
- `job` may create one `snapshot` (backup) or produce an export (restore)
- `snapshot` contains a locator that tells the Hub/UI “where it is”

## Routing Rules (v0 local storage)

- Backup jobs: any eligible worker can run (unless you later add affinity).
- Restore jobs: must set `target_worker_id` to the `snapshot.worker_id`.
- Delete/retention jobs: must also route to `snapshot.worker_id` (because the bytes are local).

## Multi-Tenant Filesystem Contract (Worker)

Worker writes artifacts under:

```
/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/
```

The Hub never constructs this path; it stores it as `local_path` as reported by the Worker.
