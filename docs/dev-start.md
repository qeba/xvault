# Start Development (Monorepo + Docker + Compose)

This document is the step-by-step sequence to start building xVault in a way that supports local development via Docker Compose and future deployment via containers anywhere.

## Decisions (Lock-in)

- Monorepo (single git repo) that contains Hub, Worker, and shared packages.
- Each service builds into its own Docker image.
- Local development uses Docker Compose for dependencies (Postgres, Redis) and for running Hub/Worker.
- v0 storage is worker-local filesystem (mounted volume in Compose).

## Suggested Monorepo Layout

```
/xvault
  /docs
  /cmd
    /hub            # Hub API entrypoint (Go)
    /worker         # Worker entrypoint (Go)
  /internal
    /hub            # Hub domain logic
    /worker         # Worker domain logic
  /pkg
    /types          # Shared types: job payloads, status enums
    /crypto         # Shared helpers: key handling, envelope encryption
  /migrations       # Postgres migrations
  /deploy
    /docker
      /hub
        Dockerfile
      /worker
        Dockerfile
    docker-compose.yml
    .env.example
  go.mod
  go.sum
```

Notes:
- Keep cross-service shared code in `/pkg` only (small surface area). Avoid importing `/internal` across services.

## Local Development Stack (Docker Compose)

Compose should run:

- `postgres` (Hub metadata)
- `redis` (job queue)
- `hub` (Go service)
- `worker` (Go service)

Worker should have a persistent volume mounted for v0 backups:

- Host/volume → container path: `/var/lib/xvault/backups`

## Environment Variables (Baseline)

Hub:
- `DATABASE_URL`
- `REDIS_URL`
- `HUB_JWT_SECRET` (or similar)
- `HUB_ENCRYPTION_KEK` (platform key-encryption-key for encrypting stored secrets/private keys)

Worker:
- `WORKER_ID`
- `HUB_BASE_URL`
- `REDIS_URL`
- `WORKER_STORAGE_BASE` (default `/var/lib/xvault/backups`)

## Start Development Sequence (Recommended)

### 1) Scaffold the monorepo

Deliverables:
- repo folders in the layout above
- `go.mod` with module name

### 2) Add Dockerfiles for Hub + Worker

Deliverables:
- [deploy/docker/hub/Dockerfile](deploy/docker/hub/Dockerfile)
- [deploy/docker/worker/Dockerfile](deploy/docker/worker/Dockerfile)

Targets:
- build small, reproducible images
- run as non-root where practical

### 3) Add docker-compose for local dev

Deliverables:
- [deploy/docker-compose.yml](deploy/docker-compose.yml)
- [deploy/.env.example](deploy/.env.example)

Acceptance:
- `docker compose up` brings up Postgres + Redis + Hub + Worker

### 4) Implement DB migrations (minimal v0 schema)

Use the schema in [docs/data-model.md](data-model.md).

Deliverables:
- `/migrations` with initial tables
- Hub runs migrations on startup OR provide a `migrate` command

### 5) Implement the “first runnable slice”

Goal: prove end-to-end orchestration with the smallest surface area.

Slice:
- Hub endpoint to create a tenant (auto-create tenant keypair)
- Hub endpoint to create a source with encrypted credentials
- Hub endpoint to enqueue a backup job
- Worker consumes backup job → pulls a trivial source connector (start with SSH/SFTP) → produces `backup.tar.zst.enc` → writes to worker volume → reports snapshot locator to Hub
- Hub endpoint to list snapshots per source

Acceptance:
- One backup run results in:
  - a `snapshots` row with `storage_backend=local_fs`
  - a file on worker storage under `tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`

### 6) Add retention cleanup job type (v0)

Goal: prevent unbounded disk growth.

Approach:
- Hub computes which snapshots to delete for a schedule
- Hub enqueues `delete_snapshot` jobs targeted to `snapshot.worker_id`
- Worker deletes local path and reports completion

### 7) Add restore export (optional v0)

If you need restores before S3:
- restore job runs on owning worker
- produces a restore archive locally (or streams later)

## Team Workflow Suggestions

- Make one engineer own “Hub schema + migrations” early (reduces churn).
- Make one engineer own “Worker pipeline + artifact format” early.
- Keep shared types (`/pkg/types`) stable; version fields in manifests.

## What Not To Build Yet

- Multi-worker cross-disk restores (requires shared storage or replication)
- Advanced dedupe (Kopia)
- BYO storage buckets
- Full dashboard UI
