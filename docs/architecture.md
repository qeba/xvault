# Architecture Overview

**Version:** 2.1 (Local storage v0)
**Last Updated:** December 27, 2025

---

## Goals & Non-Goals

### Goals (v1)

- **No install on customer servers**: users provide connection details (FTP/SFTP/SSH/DB) and we run backups from our infrastructure.
- **Automated**: schedules, retention, reporting.
- **Pluggable sources**: start with SSH/SFTP + MySQL/Postgres; add FTP and CMS helpers.
- **Separation of concerns**: Hub remains orchestration/control plane; Workers do data-plane work.

### Non-Goals (v1)

- Backing up private/internal-only servers with zero customer network changes.
      - If the customer cannot expose SSH/SFTP/DB to the internet, we’ll need a future “connector/tunnel” option.

---

## Data Flow Architecture

### Critical Principle: Control Plane vs Data Plane

In this model, **the Hub API does not transfer backup data**, but the platform still operates a **data plane**: a fleet of Workers that pull data from customer sources.

## Multi-Tenancy & Snapshot Addressing (Important Early)

Because xVault is a SaaS with many users and many sources per user, every stored backup must have a stable, unique address that does not depend on “which machine it happened to run on”.

### Canonical IDs

Use opaque IDs everywhere (no emails/usernames in paths):

- `tenant_id` (aka user/org)
- `source_id` (one per server/source)
- `snapshot_id` (one per backup run)
- `worker_id` (identity of the worker that executed the job)

### Snapshot Locator (Hub Metadata)

When a Worker completes a backup, it reports a **snapshot locator** to the Hub. The Hub stores this locator and the UI uses it to show where a snapshot lives.

Example fields:

- `storage_backend`: `local_fs` (v0), `s3` (later)
- `worker_id`: required for `local_fs`
- `local_path` (or `local_ref`): required for `local_fs`
- `object_key` / `bucket` / `etag`: later for S3/Garage

Key idea: the frontend should never need to guess paths or know about servers; it only talks to Hub, which owns the locator.

```
┌──────────────┐          ┌──────────────────────┐
│  Dashboard    │─────────▶│   HUB API (Control)  │
│  (Vue.js)     │  JWT     │ - Orchestration      │
└──────────────┘          │ - Metadata only       │
                          └──────────┬───────────┘
                                     │ jobs
                                     ▼
                          ┌──────────────────────┐
                          │ Queue (Redis)        │
                          └──────────┬───────────┘
                                     │
                                     ▼
                          ┌──────────────────────┐
                          │ Worker Fleet (Data)  │
                          │ - Pull from sources  │
                          │ - Encrypt/package    │
                          │ - Store locally (v0) │
                          └──────┬────────┬──────┘
                                 │        │
                           files/ssh   db dumps
                                 │        │
                                 ▼        ▼
                        ┌─────────────────────────┐
                        │ Customer Sources         │
                        │ - SFTP/SSH               │
                        │ - FTP (files only)       │
                        │ - MySQL/Postgres         │
                        │ - WordPress (over SSH)   │
                        └─────────────────────────┘

                          ┌──────────────────────────────┐
                          │ Worker Local Storage (v0)     │
                          │ - Encrypted artifacts on disk │
                          └──────────────────────────────┘

                          ┌──────────────────────────────┐
                          │ Storage (Garage/S3) (later)   │
                          │ - Encrypted objects           │
                          └──────────────────────────────┘
```

### Backup Flow (Remote Pull)

```
┌──────────────────────────────────────────────────────────────────┐
│ BACKUP OPERATION                                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. DASHBOARD → HUB: Create backup schedule                      │
│                                                                  │
│  2. HUB → REDIS: Enqueue job                                     │
│                                                                  │
│  3. WORKER: Dequeue job                                          │
│                                                                  │
│  4. WORKER: Pull data from customer source                       │
│     ├─ Files (SSH/SFTP): rsync/sftp download to temp dir         │
│     ├─ Files (FTP): download mirror to temp dir (files only)     │
│     ├─ Database (direct): connect and dump (e.g., pg_dump)       │
│     ├─ Database (over SSH): run mysqldump/pg_dump remotely       │
│     └─ Temporary files: /tmp/gobackup/{job_id}/ (worker-local)   │
│                                                                  │
│  5. WORKER: Encrypt/package + write artifact to local storage    │
│     ├─ Final path: /var/lib/xvault/backups/{tenant}/{source}/... │
│     └─ (later) optionally upload the same artifact to S3/Garage  │
│                                                                  │
│  6. WORKER → HUB: Report completion (metadata only)              │
│     { snapshot_id, size, duration, status, worker_id, local_ref }│
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Restore Flow (v1)

In v0 (local storage), restores are easiest if the Worker that owns the snapshot performs the restore.

Later (when you add S3/Garage), restores can return a presigned URL.

```
┌──────────────────────────────────────────────────────────────────┐
│ RESTORE OPERATION (v1 - Simple)                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. USER → DASHBOARD: Trigger restore                           │
│                                                                  │
│  2. DASHBOARD → HUB: POST /api/v1/jobs/{id}/restore             │
│                                                                  │
│  3. HUB → WORKER: Enqueue restore job                            │
│                                                                  │
│  4. WORKER: Execute restore                                      │
│     ├─ Read encrypted backup from local worker storage (v0)      │
│     ├─ Decrypt/restore into: /tmp/gobackup/restore/{job_id}/     │
│     ├─ Create zip/tar archive                                    │
│     └─ Write restore artifact locally (v0) or upload later (v1)  │
│                                                                  │
│  5. WORKER → HUB: Report restore complete + download URL         │
│                                                                  │
│  6. USER → STORAGE: Download zip via signed URL                 │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Component Responsibilities

### Worker (Go service)

| Responsibility | Details |
|----------------|---------|
| Data Transfer | Pull from customer sources; store locally (v0); upload to Storage (later) |
| Temporary Storage | `/tmp/gobackup/{job_id}/` on worker machines |
| Packaging + Encryption | v0: build a single encrypted artifact (archive + compression + encryption); later optionally switch to Kopia for dedupe/snapshots |
| Deduplication | Later (optional) |
| Source Connectors | SSH/SFTP, FTP (files), DB dump (direct or via SSH) |
| Communication | Dequeue jobs, report metadata and status |

### Hub API (Go + Fiber)

| Responsibility | Details |
|----------------|---------|
| Job Orchestration | Queues jobs, tracks status |
| Storage Management | Creates buckets, generates scoped credentials |
| Metadata Storage | Job history, schedules, users (PostgreSQL) |
| Authentication | JWT for dashboard, API keys for agents |
| **NOT** | Backup data transfer (handled by Workers), backup processing logic |

### Storage (v0 local, v1 Garage/S3)

| Responsibility | Details |
|----------------|---------|
| Data Storage | v0: local encrypted artifacts on worker disk; v1: encrypted objects in S3-compatible storage |
| Access Pattern | v0: worker-local filesystem; v1: direct S3 API from workers |
| Credentials | v0: OS-level disk access; v1: scoped per user/source (bucket or prefix policy) |

### Dashboard (Vue.js)

| Phase | Features |
|-------|----------|
| v1 | API testing only, manual cURL/Postman |
| v2 | Multi-user authentication, server management |
| v3 | Admin dashboard (future) |

---

## Temporary Storage Strategy

### Worker-Side Temporary Storage

```
/tmp/gobackup/
├── {job_id}/
│   ├── source-mirror/        # Pulled files (FTP/SFTP/SSH)
│   ├── dumps/                # Optional DB dump artifacts (or streaming)
│   └── restore/              # Restore extraction
└── worker.log
```

**Key Points:**
- Temporary files exist **only on worker machines**
- You can stream DB dumps (preferred) or materialize to disk (simpler)
- Cleanup happens after each job
- Temp directory must be isolated per job and aggressively cleaned

### Worker-Side Durable Storage (v0)

In v0, “Storage” is simply the Worker’s filesystem. The Worker writes a final encrypted artifact into a durable path (outside `/tmp`).

Recommended base path:

```
/var/lib/xvault/backups/
```

Multi-tenant directory layout (v0):

```
/var/lib/xvault/backups/
      tenants/
            {tenant_id}/
                  sources/
                        {source_id}/
                              snapshots/
                                    {snapshot_id}/
                                          backup.tar.zst.enc
                                          manifest.json
                                          meta.json
```

Rules:
- Only allow `[a-zA-Z0-9_-]` in IDs when used in filesystem paths.
- Never use user-provided names in paths.
- Store `tenant_id/source_id/snapshot_id` inside `meta.json` as well (for recovery/auditing).

Notes:
- Ensure the directory is on a disk with enough capacity (and monitoring).
- Apply a retention policy to avoid unbounded growth.

## Packaging Format (v0)

To keep v0 simple and debuggable, each snapshot is stored as a single encrypted file plus a manifest.

Recommended pipeline:

- Stage inputs into `/tmp/gobackup/{job_id}/...`
- Create a tar archive from staged data
- Compress with Zstandard (zstd)
- Encrypt into a final artifact (public-key or platform-managed recipient)

Example artifact name:

- `backup.tar.zst.enc`

The manifest should include:

- `tenant_id`, `source_id`, `snapshot_id`, `job_id`, `worker_id`
- content summary (paths, DB dump info)
- sizes + hashes (at least for the final artifact)
- encryption metadata (algorithm, recipient key id / fingerprint)

Later, you can replace the packaging layer with Kopia without changing Hub orchestration.

### No Temporary Storage on Hub

The Hub **does not** handle backup data:
- No data transfer through Hub
- Only metadata and orchestration state in PostgreSQL

---

## Credentials & Secrets Flow

This model requires storing **source credentials** (SSH keys/passwords, FTP credentials, DB passwords) securely.

Baseline approach:

- Hub stores source credentials **encrypted at rest** (envelope encryption).
- Workers fetch credentials at job start, use them in-memory, and do not persist them.

## Backup Encryption Keys (v0 Platform-Managed)

v0 goal: encryption works reliably with minimal operational complexity.

- When a tenant is created (or first source is created), the platform generates a tenant encryption identity.
- The platform stores the tenant private key encrypted at rest (DB ciphertext) and can decrypt during restore.
- Workers encrypt backups using the tenant public key (or a platform recipient key).

Future option (customer-managed):

- Tenant uploads a public key.
- Workers encrypt backups to the customer public key.
- Platform cannot decrypt; restores require the customer to provide the private key (or do client-side decrypt).

Future hardening options:

- Integrate an external secrets manager (Vault/KMS).
- Support customer-provided SSH keys.
- Support “bring-your-own storage” so customers can own the S3 bucket.

```
┌─────────────────────────────────────────────────────────────┐
│ SOURCE + STORAGE PROVISIONING                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. User adds a Source in Dashboard                          │
│     → type: ssh|sftp|ftp|mysql|postgres|wordpress            │
│     → credentials stored encrypted                           │
│                                                             │
│  2. Hub enqueues a job in Redis                              │
│                                                             │
│  3. Worker dequeues job                                      │
│     → fetches source credentials                             │
│     → pulls data + encrypts + stores locally (v0)            │
│                                                             │
│  4. Worker reports metadata (snapshot_id/local_ref/etc)      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Scalability Considerations

### Horizontal Scaling

| Component | Scaling Strategy |
|-----------|------------------|
| Hub API | Stateless, can run multiple instances behind load balancer |
| Dashboard | Static assets, can be served from CDN |
| Workers | Horizontal scale; autoscale by queue depth and bandwidth |
| Storage | v0: local disks per worker (limited); v1: Garage/S3 supports clustering |

### Multi-Worker Local Storage Reality (v0)

With `local_fs` storage, snapshots are physically attached to a specific worker’s disk.

- Pros: simplest path to a working product.
- Cons: if that worker is lost, the snapshots stored only on it are lost.

To keep behavior correct:

- Hub must record `worker_id` in the snapshot locator.
- Restore jobs must be routed to the same `worker_id` (or a storage node that can access that disk).
- The UI lists snapshots from Hub metadata; it does not browse worker filesystems.

### Bottlenecks to Avoid

```
❌ WRONG (single worker becomes bottleneck):
      Source1 ─┐
      Source2 ─┼──▶ ONE WORKER ───▶ STORAGE
      Source3 ─┘

✅ RIGHT (worker fleet scales):
      Source1 ──────┐
      Source2 ──────┼──▶ WORKER FLEET ───▶ STORAGE
      Source3 ──────┘
                        │
                        ▼ (metadata only)
                   HUB
```

---

## API Design Principles

### Worker API (Internal)

- Authenticated via internal credentials/service identity
- Workers pull jobs from queue and/or call Hub for job payload
- Workers report status updates

Worker identity:
- Each worker has a stable `worker_id` (configured env var or generated on first boot and persisted).
- Hub maintains a registry of active workers (heartbeats) so it can route restore jobs.

### Dashboard API (External)

- Authenticated via JWT
- Used by frontend
- CRUD operations for sources, schedules
- Read-only views for job history

### Storage API (S3-Compatible)

- No custom API, use standard S3 protocol
- Scoped credentials per server
- Presigned URLs for user downloads (restore)

In v0 (local storage), “download URLs” are not naturally available without extra infra.
Common v0 approach is: “restore job creates an export artifact” and Hub provides a one-time link to download it (implemented later), or the operator manually retrieves it.

---

## Development Phases

### Phase 0: Foundation
- Project setup + local dev stack (Docker Compose)

### Phase 1: Worker (Priority)
- Worker service + job runner
- Source connectors (SSH/SFTP first)
- DB dumps (direct or over SSH)
- Store encrypted backups on worker filesystem (v0)

### Phase 2: Hub API
- Fiber endpoints
- Job queue (Redis)
- Storage management

### Phase 3: Storage Setup (later)
- Garage/S3 deployment
- Bucket/prefix automation + scoped credentials
- Worker uploader module

### Phase 4: Dashboard (Future)
- Vue.js frontend
- Multi-user support

### Phase 5: Admin Dashboard (Later)
- User management
- System monitoring
- Billing integration

---

## Key Takeaways

1. **No customer install (v1)** - backups are pulled by your Worker fleet
2. **Hub stays control plane** - metadata/orchestration only
3. **Data plane is Workers** - backup data path: Source → Worker → Storage
4. **Security focus shifts** - protect credentials and worker runtime
5. **Start with SSH/SFTP** - it covers WordPress + DB via remote commands
