# xVault Development Progress

**Last Updated:** 2025-12-29

This file tracks the implementation progress of xVault features based on the development sequence in [docs/dev-start.md](dev-start.md) and milestones in [docs/plan.md](plan.md).

## Technology Stack

| Component | Technology | Status |
|-----------|------------|--------|
| **Hub API** | Go + Fiber v2 | ✅ Configured |
| **Worker** | Go | ✅ Configured |
| **Frontend** | Vue.js | 🔄 Deferred to v2+ |
| **Database** | PostgreSQL | ✅ Docker configured |
| **Queue** | Redis | ✅ Docker configured |
| **Storage v0** | Worker-local filesystem | ✅ Docker volume configured |
| **Storage v1** | S3/Garage | 🔄 Deferred |

---

## Legend

- ⏳ **Not Started** - Task not yet begun
- 🚧 **In Progress** - Currently being worked on
- ✅ **Done** - Completed and tested
- ⚠️ **Blocked** - Waiting on dependencies or decisions
- 🔄 **Deferred** - Moved to later phase

---

## Foundation (Phase 0)

**Status**: ✅ **Complete** - Scaffolding done, Docker setup ready

| Step | Task | Status | Notes |
|------|------|--------|-------|
| 0.1 | Monorepo structure (`/cmd`, `/internal`, `/pkg`, `/migrations`, `/deploy`) | ✅ | Repo scaffolded |
| 0.2 | `go.mod` with module `xvault` and Go 1.25 | ✅ | |
| 0.3 | Hub Dockerfile ([deploy/docker/hub/Dockerfile](deploy/docker/hub/Dockerfile)) | ✅ | Multi-stage, distroless base |
| 0.4 | Worker Dockerfile ([deploy/docker/worker/Dockerfile](deploy/docker/worker/Dockerfile)) | ✅ | Multi-stage, distroless base |
| 0.5 | Docker Compose with Postgres, Redis, Hub, Worker | ✅ | |
| 0.6 | Environment variables (.env.example) | ✅ | |
| 0.7 | Placeholder Hub service (health check, basic routes) | ✅ | Fiber v2, listens on :8080 |
| 0.8 | Placeholder Worker service (Redis ping, storage base config) | ✅ | Connects to Redis, awaits jobs |

**Storage Path Note**: Worker storage base is `/var/lib/xvault/backups` (mounted volume in Compose)

---

## Step 4: Database Migrations

**Status**: ✅ **Complete** - Full schema implemented with Goose migration tool

**Goal**: Implement the minimal v0 schema from [docs/data-model.md](data-model.md)

**Deliverables**:
- `/migrations` directory with SQL migration files
- Migration runner in Hub (startup or `migrate` command)

| Task | Status | Notes |
|------|--------|-------|
| 4.1 | Set up migration tool/library | ✅ | Using Goose v3 (github.com/pressly/goose/v3) |
| 4.2 | Create `tenants` table | ✅ | `id`, `name`, `plan`, timestamps |
| 4.3 | Create `users` table | ✅ | `id`, `tenant_id`, `email`, `password_hash`, `role`, timestamps |
| 4.4 | Create `credentials` table | ✅ | `id`, `tenant_id`, `kind`, `ciphertext`, `key_id`, timestamps |
| 4.5 | Create `tenant_keys` table | ✅ | `id`, `tenant_id`, `algorithm`, `public_key`, `encrypted_private_key`, `key_status`, timestamps |
| 4.6 | Create `sources` table | ✅ | `id`, `tenant_id`, `type`, `name`, `status`, `config` (JSONB), `credential_id`, timestamps |
| 4.7 | Create `schedules` table | ✅ | `id`, `tenant_id`, `source_id`, `cron`/`interval_minutes`, `timezone`, `enabled`, `retention_policy` (JSONB), timestamps |
| 4.8 | Create `workers` table | ✅ | `id`, `name`, `status`, `capabilities` (JSONB), `storage_base_path`, `last_seen_at`, timestamps |
| 4.9 | Create `jobs` table | ✅ | `id`, `tenant_id`, `source_id`, `type`, `status`, `priority`, `target_worker_id`, `lease_expires_at`, `attempt`, `payload` (JSONB), timestamps, error fields |
| 4.10 | Create `snapshots` table | ✅ | `id`, `tenant_id`, `source_id`, `job_id`, `status`, `size_bytes`, duration fields, `manifest_json`, encryption metadata, locator fields (`storage_backend`, `worker_id`, `local_path`), timestamps |
| 4.11 | Create `audit_events` table (optional for v0) | ✅ | Included for complete audit trail |
| 4.12 | Add indexes/constraints per data-model.md | ✅ | All indexes and foreign keys added |
| 4.13 | Hub runs migrations on startup OR provides `migrate` command | ✅ | Supports `-migrate` flag and `HUB_AUTO_MIGRATE` env var |

**Implementation Details**:
- Migration file: [internal/hub/database/migrations/0001_init.sql](internal/hub/database/migrations/0001_init.sql)
- Database package: [internal/hub/database/migrate.go](internal/hub/database/migrate.go)
- Hub CLI: `./bin/hub -migrate` (run migrations and exit)
- Hub CLI: `./bin/hub -migrate-status` (show migration status)
- Auto-migrate: Set `HUB_AUTO_MIGRATE=true` to run migrations on startup

**Database Types (Enums)**:
- `user_role`: owner, admin, member
- `key_status`: active, rotated, disabled
- `credential_kind`: source, storage
- `source_type`: ssh, sftp, ftp, mysql, postgres, wordpress
- `source_status`: active, disabled
- `schedule_status`: enabled, disabled
- `worker_status`: online, offline, draining
- `job_type`: backup, restore, delete_snapshot
- `job_status`: queued, running, finalizing, completed, failed, canceled
- `snapshot_status`: completed, failed
- `storage_backend`: local_fs, s3

---

## Step 5: First Runnable Slice (End-to-End)

**Status**: ✅ **Complete** - Full end-to-end backup pipeline tested and working

**Goal**: Prove end-to-end orchestration with smallest surface area

**Acceptance**:
- ✅ One backup run results in a `snapshots` row with `storage_backend=local_fs`
- ✅ A file exists on worker storage under `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`
- ✅ Worker successfully claims and processes backup jobs from Hub

**Connector Scope**: SSH/SFTP only initially (simplest, covers most use cases)

### 5.1 Hub: Tenant Management

| Task | Status | Notes |
|------|--------|-------|
| 5.1.1 | `POST /api/v1/tenants` endpoint | ✅ | [`internal/hub/handlers/handlers.go:47`](internal/hub/handlers/handlers.go) |
| 5.1.2 | Generate tenant keypair on creation (Age/x25519) | ✅ | [`internal/hub/service/service.go:34`](internal/hub/service/service.go) |
| 5.1.3 | Store tenant private key encrypted at rest | ✅ | Using `HUB_ENCRYPTION_KEK` envelope encryption |
| 5.1.4 | `GET /api/v1/tenants/:id` endpoint | 🔄 | Not implemented yet (low priority) |

### 5.2 Hub: Source & Credential Management

| Task | Status | Notes |
|------|--------|-------|
| 5.2.1 | `POST /api/v1/credentials` endpoint | ✅ | [`internal/hub/handlers/handlers.go:78`](internal/hub/handlers/handlers.go) |
| 5.2.2 | Envelope encryption implementation | ✅ | [`pkg/crypto/age.go:72`](pkg/crypto/age.go) |
| 5.2.3 | `POST /api/v1/sources` endpoint | ✅ | [`internal/hub/handlers/handlers.go:103`](internal/hub/handlers/handlers.go) |
| 5.2.4 | `GET /api/v1/sources` list endpoint | ✅ | [`internal/hub/handlers/handlers.go:126`](internal/hub/handlers/handlers.go) |
| 5.2.5 | Source config validation (SSH/SFTP) | 🔄 | Client-side validation only for v0 |

### 5.3 Hub: Job Queue & Orchestration

| Task | Status | Notes |
|------|--------|-------|
| 5.3.1 | `POST /api/v1/jobs` endpoint (manual trigger) | ✅ | [`internal/hub/handlers/handlers.go:166`](internal/hub/handlers/handlers.go) |
| 5.3.2 | Job payload format definition | ✅ | [`pkg/types/types.go:58`](pkg/types/types.go) |
| 5.3.3 | Enqueue job to Redis | ✅ | Uses `xvault:jobs:queue` key |
| 5.3.4 | Internal: `POST /internal/jobs/claim` endpoint | ✅ | Worker claims next queued job |
| 5.3.5 | Internal: `POST /internal/jobs/:id/complete` endpoint | ✅ | Worker reports completion + snapshot |
| 5.3.6 | Internal: `GET /internal/credentials/:id` endpoint | ✅ | Worker fetches encrypted creds |
| 5.3.7 | Internal: `GET /internal/tenants/:id/public-key` endpoint | ✅ | Worker fetches tenant public key |
| 5.3.8 | Internal: `POST /internal/workers/register` endpoint | ✅ | Worker registration |
| 5.3.9 | Internal: `POST /internal/workers/heartbeat` endpoint | ✅ | Worker heartbeats |

### 5.4 Worker: Job Loop

| Task | Status | Notes |
|------|--------|-------|
| 5.4.1 | Redis job dequeue (blocking or polling) | ✅ | Polling via Hub API claim endpoint |
| 5.4.2 | Claim job via Hub API | ✅ | [`internal/worker/client/client.go:48`](internal/worker/client/client.go) |
| 5.4.3 | Fetch and decrypt credentials | ✅ | JIT credential fetch via Hub API |
| 5.4.4 | Job lease management (heartbeat/renewal) | ✅ | 30s heartbeat interval |
| 5.4.5 | Error handling and retry logic | ✅ | Errors logged, next job claimed on failure |
| 5.4.6 | Graceful shutdown (finish current job) | ✅ | SIGINT/SIGTERM handling with context cancel |

### 5.5 Worker: SSH/SFTP Connector

| Task | Status | Notes |
|------|--------|-------|
| 5.5.1 | SSH client connection | ✅ | [`internal/worker/connector/sftp.go:37`](internal/worker/connector/sftp.go) |
| 5.5.2 | SFTP file download to temp dir | ✅ | Uses `/tmp/gobackup/{job_id}/source-mirror/` |
| 5.5.3 | Recursive directory pull | ✅ | SFTP walker with recursive pull |
| 5.5.4 | Error handling for connection failures | ✅ | Errors propagated up for job failure reporting |

### 5.6 Worker: Packaging & Encryption

| Task | Status | Notes |
|------|--------|-------|
| 5.6.1 | Create tar archive from staged data | ✅ | Simple ustar format implementation |
| 5.6.2 | Compress with Zstandard (zstd) | ✅ | Using klauspost/compress/zstd |
| 5.6.3 | Encrypt with Age (tenant public key) | ✅ | Using pkg/crypto age encryption |
| 5.6.4 | Generate `backup.tar.zst.enc` artifact | ✅ | Full artifact pipeline |
| 5.6.5 | Generate `manifest.json` | ✅ | IDs, sizes, hashes, encryption metadata |
| 5.6.6 | Generate `meta.json` | ✅ | tenant_id, source_id, snapshot_id, job_id, worker_id |
| 5.6.7 | Cleanup temp directory | ✅ | Aggressive cleanup after job |

### 5.7 Worker: Local Storage (v0)

| Task | Status | Notes |
|------|--------|-------|
| 5.7.1 | Create multi-tenant directory structure | ✅ | Path: `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/` |
| 5.7.2 | Write artifact to durable path | ✅ | [`internal/worker/storage/storage.go:36`](internal/worker/storage/storage.go) |
| 5.7.3 | Write manifest.json and meta.json | ✅ | Both files written with artifact |

### 5.8 Hub: Snapshot Metadata

| Task | Status | Notes |
|------|--------|-------|
| 5.8.1 | Store snapshot record in database | ✅ | [`internal/hub/repository/repository.go:432`](internal/hub/repository/repository.go) |
| 5.8.2 | Store snapshot locator | ✅ | storage_backend, worker_id, local_path |
| 5.8.3 | `GET /api/v1/snapshots` list endpoint | ✅ | [`internal/hub/handlers/handlers.go:345`](internal/hub/handlers/handlers.go) |
| 5.8.4 | `GET /api/v1/snapshots/:id` details endpoint | ✅ | [`internal/hub/handlers/handlers.go:368`](internal/hub/handlers/handlers.go) |

### 5.9 End-to-End Integration Test

| Task | Status | Notes |
|------|--------|-------|
| 5.9.1 | Create tenant → verify keypair generated | ✅ | Tenant created with Age/x25519 keypair |
| 5.9.2 | Create source → verify credentials encrypted | ✅ | Credentials encrypted with platform KEK |
| 5.9.3 | Enqueue backup job → verify appears in Redis | ✅ | Job enqueued to Redis, status=queued |
| 5.9.4 | Worker claims job → verify status=running | ✅ | Worker claimed job via Hub API |
| 5.9.5 | Worker completes SSH/SFTP backup | ✅ | **Real SSH server test: 10.0.100.85:/home/web/test** |
| 5.9.6 | Verify snapshot stored in worker filesystem | ✅ | Artifact, manifest, meta.json all present |
| 5.9.7 | Verify snapshot record in Hub DB | ✅ | Snapshot record with correct locator |
| 5.9.8 | List snapshots via API | ✅ | API endpoint working |

**✅ END-TO-END TEST COMPLETE**

Successfully backed up files from real SSH server `10.0.100.85:/home/web/test`:
- **Job ID**: `4e4dd30a-3493-4021-bc56-c5b5acf9aa06`
- **Snapshot ID**: `17a1cbfe36bbf9e35cea08da736b608f`
- **Files Pulled**: 2 files (100MB.bin + file = 104,857,600 bytes)
- **Artifact Size**: 104,886,398 bytes (encrypted + compressed)
- **Duration**: 593ms
- **Encryption**: age-x25519
- **Storage Path**: `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`

**Files Created**:
```
/var/lib/xvault/backups/tenants/.../snapshots/17a1cbfe36bbf9e35cea08da736b608f/
├── backup.tar.zst.enc (104,886,398 bytes)
├── manifest.json (698 bytes)
└── meta.json (165 bytes)
```

**V0 Credential Encryption Note**: For v0, credentials are encrypted with the platform KEK (not tenant public key) so workers can decrypt them. This is a temporary approach for the MVP; v1 will use proper envelope encryption where workers can't decrypt credentials directly.

**Worker Dockerfile**: Changed from distroless to debian:bookworm-slim with dedicated `worker` user (UID 1000) to fix storage permission issues.

**Test Commands**:
```bash
# Create tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" -d '{"name":"my-tenant"}'

# Create credential (password base64 encoded)
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"...","kind":"source","plaintext":"dGVzdC1wYXNz"}'

# Create SSH source
curl -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"...","type":"ssh","name":"server","credential_id":"...","config":{"host":"10.0.100.85","port":22,"username":"web","paths":["/home/web/test"],"use_password":true}}'

# Enqueue job
curl -X POST "http://localhost:8080/api/v1/jobs?tenant_id=..." \
  -H "Content-Type: application/json" -d '{"source_id":"..."}'
```

**API Documentation**: See [docs/api-reference.md](api-reference.md) for complete API reference.

---

## Step 6: Retention & Cleanup (v0)

**Goal**: Prevent unbounded disk growth

| Task | Status | Notes |
|------|--------|-------|
| 6.1 | Retention policy evaluation in Hub | ⏳ | Parse `retention_policy` JSONB from schedules |
| 6.2 | Identify snapshots to delete per policy | ⏳ | |
| 6.3 | Enqueue `delete_snapshot` jobs | ⏳ | Must target `snapshot.worker_id` |
| 6.4 | Worker: handle `delete_snapshot` job type | ⏳ | |
| 6.5 | Worker deletes local filesystem path | ⏳ | |
| 6.6 | Worker reports completion to Hub | ⏳ | |
| 6.7 | Hub updates snapshot status or deletes record | ⏳ | |

---

## Step 7: Restore Export (Optional v0)

**Goal**: Enable restore downloads in v0 (before S3/Garage)

| Task | Status | Notes |
|------|--------|-------|
| 7.1 | `POST /api/v1/jobs/:id/restore` endpoint | ⏳ | |
| 7.2 | Hub enqueues restore job targeted to `snapshot.worker_id` | ⏳ | |
| 7.3 | Worker: handle `restore` job type | ⏳ | |
| 7.4 | Worker reads encrypted backup from local storage | ⏳ | |
| 7.5 | Worker decrypts and extracts to temp dir | ⏳ | |
| 7.6 | Worker creates zip/tar archive | ⏳ | |
| 7.7 | Worker reports restore complete | ⏳ | |
| 7.8 | Provide download mechanism | ⏳ | May need additional infra for v0 (or manual retrieval) |

---

## Deferred to v1 (S3/Garage Storage)

| Feature | Notes |
|---------|-------|
| S3/Garage upload module | Worker uploads after local write |
| Scoped credential generation | Per-tenant or per-source S3 credentials |
| Presigned URL downloads | Native restore downloads |
| Multi-worker cross-disk restores | Shared storage enables any worker to restore |
| Advanced dedupe (Kopia) | Optional optimization |

---

## Deferred to v2+ (Frontend Dashboard & Authentication)

| Feature | Framework | Notes |
|---------|-----------|-------|
| **Frontend Dashboard** | **Vue.js** | v1 = API testing only (cURL/Postman), v2 = UI development |
| **User Authentication** | JWT + Fiber middleware | v1 = no auth or simple API key, v2 = full JWT |
| **Multi-user Support** | Vue.js + Hub API | v1 = single tenant or simple auth, v2 = multi-tenant |
| **Admin Dashboard** | Vue.js | v3+ - system monitoring, user management, billing |

**Frontend Stack (v2+):**
- Framework: Vue.js 3
- API Client: Axios or native fetch
- Authentication: JWT tokens stored in httpOnly cookies or localStorage
- State Management: Pinia (if needed)
- Build Tool: Vite

---

## Additional Connectors (Post-MVP)

After SSH/SFTP is working, add these connectors incrementally:

| Connector | Status | Notes |
|-----------|--------|-------|
| FTP (files only) | 🔄 | Simpler than SSH, no remote command execution |
| MySQL dump (direct) | 🔄 | Connect directly to TCP port |
| PostgreSQL dump (direct) | 🔄 | Connect directly to TCP port |
| MySQL dump (via SSH) | 🔄 | SSH tunnel or remote mysqldump |
| PostgreSQL dump (via SSH) | 🔄 | SSH tunnel or remote pg_dump |
| WordPress (over SSH) | 🔄 | wp-config.php + files via SSH |

---

## Development Checklist

When starting a new task:

1. **Read relevant documentation:**
   - [docs/architecture.md](architecture.md) - For architecture decisions
   - [docs/data-model.md](data-model.md) - For database schema
   - [docs/dev-start.md](dev-start.md) - For development sequence
   - [docs/plan.md](plan.md) - For implementation milestones

2. **Update this file:**
   - Mark task as 🚧 **In Progress**
   - Add any notes or decisions made

3. **Implement:**
   - Write code following the monorepo structure
   - Keep shared types in `/pkg` only
   - Don't cross-import `/internal` between services

4. **Test:**
   - Test locally with `docker compose`
   - Update task status to ✅ **Done** when passing
   - Document any issues in Notes column

5. **Move to next task**

---

## Quick Reference Commands

```bash
# Start full dev stack (with auto-migrate enabled)
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build

# Build services locally
CGO_ENABLED=0 go build -o bin/hub ./cmd/hub
CGO_ENABLED=0 go build -o bin/worker ./cmd/worker

# Run migrations manually
export DATABASE_URL="postgres://xvault:xvault@localhost:5432/xvault?sslmode=disable"
./bin/hub -migrate

# Check migration status
./bin/hub -migrate-status

# Run Hub with auto-migrate (for local development)
export DATABASE_URL="postgres://xvault:xvault@localhost:5432/xvault?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export HUB_AUTO_MIGRATE="true"
./bin/hub

# Run Worker
export WORKER_ID="worker-1"
export WORKER_STORAGE_BASE="/var/lib/xvault/backups"
export HUB_BASE_URL="http://localhost:8080"
export REDIS_URL="redis://localhost:6379/0"
./bin/worker

# Run tests
go test ./...

# Check logs
docker compose logs hub
docker compose logs worker
```

---

## Key Architecture Reminders

1. **Multi-tenancy**: Always use opaque IDs (`tenant_id`, `source_id`, `snapshot_id`) - never user-provided names in paths
2. **Storage path**: `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`
3. **Temp path**: `/tmp/gobackup/{job_id}/` (aggressive cleanup required)
4. **No secrets in Redis**: Job payloads reference `credential_id`, not plaintext
5. **Hub is control plane only**: Never transfers backup data
6. **Worker routing**: Restore/delete jobs must target the `worker_id` that owns the snapshot
