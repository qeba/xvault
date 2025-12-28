# Automated Backup SaaS - Implementation Plan

**Version:** 1.1 (Remote Pull / Worker-first / Local storage v0)
**Created:** December 27, 2025

---

## Quick Overview

You're building an **Automated Backup SaaS** with the name xVault platform with these components.

```
┌──────────────┐          ┌──────────────────────┐
│  Dashboard    │─────────▶│   HUB API (Control)  │
│  (Vue.js)     │   JWT    │ - Users/Schedules    │
└──────────────┘          │ - Sources/Creds      │
                          │ - Metadata only      │
                          └──────────┬───────────┘
                                     │ jobs
                                     ▼
                          ┌──────────────────────┐
                          │ Queue (Redis)        │
                          └──────────┬───────────┘
                                     ▼
                          ┌──────────────────────┐
                          │ Worker Fleet (Data)  │
                          │ - Pull from sources  │
                          │ - Encrypt/package    │
                          │ - Store locally (v0) │
                          └──────────┬───────────┘
                                     ▼
                          ┌──────────────────────┐
                          │ Worker Local Storage │
                          │ (filesystem, v0)     │
                          └──────────────────────┘
```

**Key Principle:** The Hub is **control plane only** (metadata + orchestration). The **Worker Fleet** is the data plane.

## Repo & Dev Environment (Monorepo + Docker)

Development approach:

- Monorepo (single repository) containing Hub + Worker + shared packages.
- Each component builds into its own Docker image.
- Local development and testing runs on Docker Compose.
- Future deployment remains flexible because everything runs in containers.

Start-here sequence for the team:

- [docs/dev-start.md](dev-start.md)

## Storage Strategy (Start Simple)

### v0 (Now): Store backups on the Worker filesystem

- Worker writes encrypted artifacts to local disk (no S3/Garage yet).
- Hub stores only metadata + a reference to the artifact (e.g., `{ worker_id, local_path }`).
- This gets you a working end-to-end backup flow with the fewest moving parts.

Recommended layout (per worker):

```
/var/lib/xvault/
    backups/
        {tenant_id}/
            {source_id}/
                {snapshot_id}/
                    backup.tar.zst.enc
                    manifest.json
                    meta.json
```

Multi-tenant rules (do this early):

- Use opaque IDs (`tenant_id`, `source_id`, `snapshot_id`) for directory names.
- Never use user-provided names (e.g., server name, hostname, email) in filesystem paths.
- Store a `meta.json` alongside artifacts that includes `{tenant_id, source_id, snapshot_id, worker_id}`.
- Hub persists a snapshot locator: `{storage_backend: local_fs, worker_id, local_path}`.

## Packaging & Encryption (v0)

Keep v0 simple: produce a single encrypted artifact per snapshot.

Artifact pipeline:

- Pull files / dump DB into `/tmp/gobackup/{job_id}/...`
- Create archive: `tar`
- Compress: `zstd`
- Encrypt: produce `backup.tar.zst.enc`

Manifest requirements (minimum):

- IDs: `tenant_id`, `source_id`, `snapshot_id`, `job_id`, `worker_id`
- Sizes: `size_bytes` (final) + optional pre-compress size
- Hash: `sha256` (final artifact)
- Encryption metadata: algorithm + recipient identifier (key id/fingerprint)

Later, you can add dedupe (Kopia) as an internal packaging backend without changing Hub orchestration.

## Keys & Secrets (v0)

Source credentials:

- Stored encrypted at rest in Hub DB.
- Worker fetches just-in-time at job start.

Backup encryption keys (platform-managed for v0):

- On tenant creation (or first source), generate a tenant keypair.
- Store the tenant private key encrypted at rest in DB (platform can decrypt for restores).
- Workers encrypt snapshot artifacts to the tenant public key.

Future (customer-managed encryption):

- Tenant provides their public key.
- Workers encrypt to customer key.
- Platform cannot decrypt; restore requires customer-provided private key or client-side decrypt.

### v1 (Later): Push from Worker → S3/Garage

- Keep the same “Source → Worker” pull and encryption.
- Replace the final “write to local disk only” step with “write locally + upload” (or stream upload).
- Hub continues to remain metadata-only; object storage becomes the durable store.

## Minimum Viable Milestones

### Milestone A: Worker-only backups (local storage)

- Define job payload format (source type, credentials reference, retention policy).
- Implement Worker job runner:
    - Pull from SSH/SFTP to a temp dir
    - Package (tar + zstd) + encrypt
    - Move final artifact into the worker “backups/” directory
    - Report `{snapshot_id, size, duration, status, worker_id, storage_backend, local_path}` to Hub

Acceptance criteria (multi-user safe):

- Two different tenants with the same “server name” cannot collide (IDs only).
- A snapshot can be uniquely addressed by `(tenant_id, source_id, snapshot_id)`.
- Hub can answer “where is this snapshot stored?” without guessing.

### Milestone B: Hub orchestration

- CRUD: sources + schedules (credentials encrypted at rest).
- Enqueue jobs in Redis.
- Track job/snapshot metadata in Postgres.

Data model reference:

- Minimal Hub schema to implement first: [docs/data-model.md](data-model.md)

Add worker-aware snapshot routing (required for v0 local storage):

- Maintain a `workers` table (or registry): `worker_id`, `status`, `last_seen`, `capabilities`, `storage_base_path`.
- Store snapshot locator in `snapshots` table: `storage_backend`, `worker_id`, `local_path`.
- Restore flow: Hub enqueues restore job specifically targeted to `worker_id` that owns the snapshot.

### Milestone C: Upload backend (S3/Garage)

- Add uploader module to Worker.
- Add storage namespace management to Hub (bucket/prefix, scoped credentials).

This becomes a storage-backend swap:

- Keep the same snapshot addressing and metadata model.
- Replace `local_path` with S3 keys (or store both during migration).

