# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Tools

## Web Search, Internet and Fetching

- Run `date` before web search for current time context.
- Use `web-search-prime` MCP for web search and internet access.
- Use `web-reader` MCP for fetching complete webpage and website content.
- Use `zread` for Documentation search  in repo to deeply analyze implementation details.
- Do not use `WebSearch` tool.

## Documentation

- Use `context7` MCP for packages documentation and examples.

## Image/Video Analysis

- Use `zai-mcp-server` MCP for image and video analysis.

# Workflow Rules (MUST FOLLOW)

## Dependency Management

- BEFORE adding ANY dependency to package.json, requirements.txt, Cargo.toml, pyproject.toml, go.mod, pom.xml, build.gradle, or any manifest:
  - Use the **dependency-verification** skill
  - Verify the latest stable version on the registry
  - Check for security advisories
  - Review breaking changes in recent releases
- NEVER blindly add dependencies without verification

## Code Implementation

- BEFORE implementing new features or integrating unfamiliar libraries:
  - Use the **example-driven-implementation** skill
  - Search for authoritative usage examples
  - Check official documentation
  - Verify version compatibility

---

# xVault Project Guide

## Project Overview

xVault is an **Automated Backup SaaS** built as a monorepo in Go. The platform consists of:

- **Hub API** ([cmd/hub](cmd/hub)): Control plane using Fiber web framework - handles orchestration, metadata, and job queuing
- **Worker** ([cmd/worker](cmd/worker)): Data plane that pulls from customer sources, encrypts/packages backups, and stores them
- **Storage**: v0 uses worker-local filesystem; v1 will add S3/Garage

**Key Principle**: Hub is control plane only (metadata + orchestration). Worker fleet is the data plane. Hub never transfers backup data.

## Architecture

### Control Plane vs Data Plane Separation

- Hub stores metadata in PostgreSQL, queues jobs in Redis
- Workers dequeue jobs, fetch credentials JIT, pull data from sources, encrypt, and store locally
- Backup data flows: Customer Source → Worker → Storage (never through Hub)
- Snapshot locators abstract storage: `{storage_backend, worker_id, local_path}` for v0

### Multi-Tenancy (Critical)

All filesystem paths and storage use opaque IDs only - never user-provided names:

- `tenant_id`, `source_id`, `snapshot_id`, `worker_id` are UUIDs
- Worker storage layout: `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`
- Hub stores snapshot locators; UI never guesses paths

### Job Flow

1. Dashboard → Hub: create schedule/source
2. Hub → Redis: enqueue backup job
3. Worker: dequeue job, fetch encrypted credentials, pull from source
4. Worker: package (tar → zstd → encrypt) → `backup.tar.zst.enc`
5. Worker → Hub: report metadata (snapshot_id, size, worker_id, local_path)
6. Hub: store snapshot with locator

## Commands

### Local Development

```bash
# Start full stack (Postgres, Redis, Hub, Worker)
cp deploy/.env.example deploy/.env
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build

# Run individual services
go run ./cmd/hub
go run ./cmd/worker
```

### Building

```bash
# Build Hub
CGO_ENABLED=0 go build -o bin/hub ./cmd/hub

# Build Worker
CGO_ENABLED=0 go build -o bin/worker ./cmd/worker

# Docker builds (via Compose)
docker compose build hub worker
```

### Testing

```bash
# Run tests
go test ./...

# Run tests for specific package
go test ./internal/hub/...
go test ./internal/worker/...
```

### Database

```bash
# Run migrations (when implemented)
go run ./cmd/hub migrate
```

## Monorepo Structure

```
/cmd
  /hub            # Hub API entrypoint
  /worker         # Worker entrypoint
/internal
  /hub            # Hub domain logic (handlers, services, repos)
  /worker         # Worker domain logic (job loop, connectors, packaging)
/pkg
  /types          # Shared types: job payloads, status enums
  /crypto         # Shared helpers: key handling, envelope encryption (future)
/migrations       # Postgres migrations
/deploy
  /docker
    /hub/Dockerfile
    /worker/Dockerfile
  docker-compose.yml
  .env.example
```

**Rule**: Keep cross-service shared code in `/pkg` only. Avoid importing `/internal` across services.

## Environment Variables

### Hub
- `HUB_LISTEN_ADDR` (default `:8080`)
- `DATABASE_URL` - Postgres connection string
- `REDIS_URL` - Redis connection string
- `HUB_JWT_SECRET` - JWT signing (future)
- `HUB_ENCRYPTION_KEK` - Key-encryption-key for tenant private keys

### Worker
- `WORKER_ID` - Unique worker identity (e.g., `worker-1`)
- `REDIS_URL` - Redis for job queue
- `WORKER_STORAGE_BASE` (default `/var/lib/xvault/backups`)
- `HUB_BASE_URL` - Hub API endpoint for status updates

## Data Model Reference

Key tables (see [docs/data-model.md](docs/data-model.md) for full schema):

- `tenants` - customer accounts
- `sources` - backup targets with credential references
- `credentials` - encrypted secrets (source credentials, tenant keys)
- `tenant_keys` - platform-managed encryption keys per tenant
- `jobs` - queued work units with status, target_worker_id, lease
- `snapshots` - backup results with locator (storage_backend, worker_id, local_path)
- `workers` - registry for routing restores to correct worker

## Routing Rules (v0 Local Storage)

- **Backup jobs**: any eligible worker can run
- **Restore/delete jobs**: must target `snapshot.worker_id` (snapshot lives on that worker's disk)

## Worker Storage Paths

- Temporary: `/tmp/gobackup/{job_id}/` (cleaned after job)
- Durable: `/var/lib/xvault/backups/tenants/{tenant_id}/sources/{source_id}/snapshots/{snapshot_id}/`
  - Contains: `backup.tar.zst.enc`, `manifest.json`, `meta.json`

## Documentation

- [docs/architecture.md](docs/architecture.md) - Full architecture and data flow
- [docs/data-model.md](docs/data-model.md) - Database schema
- [docs/dev-start.md](docs/dev-start.md) - Development setup sequence
- [docs/plan.md](docs/plan.md) - Implementation milestones

## Development Notes

- Go 1.25+ required
- Fiber v2 for Hub API
- Redis v9 for job queue
- PostgreSQL for metadata
- v0 storage: worker-local filesystem only
- Future: S3/Garage upload, Kopia dedupe, Vue.js dashboard
