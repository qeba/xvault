# xVault API Reference

This document describes all available API endpoints for the xVault backup service.

**Base URL**: `http://localhost:8080`

**Authentication**: For v0, tenant_id is passed via query parameter or `X-Tenant-ID` header. Production will use JWT tokens.

---

## Public API

### Tenants

#### Create Tenant
```http
POST /api/v1/tenants
Content-Type: application/json

{
  "name": "string"
}
```

**Response (201)**:
```json
{
  "tenant": {
    "id": "uuid",
    "name": "string",
    "plan": "free",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  },
  "public_key": "age1..."
}
```

Creates a new tenant account with an Age/x25519 keypair for encryption.

---

### Credentials

#### Create Credential
```http
POST /api/v1/credentials
Content-Type: application/json

{
  "tenant_id": "uuid",
  "kind": "source",
  "plaintext": "base64-encoded-secret"
}
```

**Response (201)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "kind": "source",
  "ciphertext": "base64-encoded-encrypted",
  "key_id": "platform-kek",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

Creates an encrypted credential. For v0, credentials are encrypted with platform KEK so workers can decrypt them.

---

### Sources

#### Create Source
```http
POST /api/v1/sources
Content-Type: application/json

{
  "tenant_id": "uuid",
  "type": "ssh",
  "name": "string",
  "credential_id": "uuid",
  "config": {
    "host": "10.0.100.85",
    "port": 22,
    "username": "web",
    "paths": ["/home/web/test"],
    "use_password": true
  }
}
```

**Response (201)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "type": "ssh",
  "name": "string",
  "status": "active",
  "config": {...},
  "credential_id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

Creates a backup source (SSH/SFTP server, FTP, database, etc.).

#### List Sources
```http
GET /api/v1/sources?tenant_id={uuid}
```

**Response (200)**:
```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "type": "ssh",
    "name": "string",
    "status": "active",
    ...
  }
]
```

Lists all sources for a tenant.

#### Get Source
```http
GET /api/v1/sources/{id}
```

**Response (200)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "type": "ssh",
  "name": "string",
  "status": "active",
  "config": {...},
  "credential_id": "uuid"
}
```

Gets details of a specific source.

---

### Jobs

#### Enqueue Backup Job
```http
POST /api/v1/jobs?tenant_id={uuid}
Content-Type: application/json

{
  "source_id": "uuid"
}
```

**Response (201)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "source_id": "uuid",
  "type": "backup",
  "status": "queued",
  "priority": 5,
  "attempt": 0,
  "payload": {
    "source_id": "uuid",
    "credential_id": "uuid",
    "source_config": {...}
  },
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

Manually triggers a backup job for a source.

---

### Snapshots

#### List Snapshots
```http
GET /api/v1/snapshots?tenant_id={uuid}&source_id={uuid}
```

**Response (200)**:
```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "source_id": "uuid",
    "job_id": "uuid",
    "status": "completed",
    "size_bytes": 104886398,
    "started_at": "timestamp",
    "finished_at": "timestamp",
    "duration_ms": 593,
    "encryption_algorithm": "age-x25519",
    "storage_backend": "local_fs",
    "worker_id": "worker-1",
    "local_path": "/var/lib/xvault/backups/tenants/.../snapshots/..."
  }
]
```

Lists all snapshots for a tenant/source. Both `tenant_id` and `source_id` are required.

#### Get Snapshot
```http
GET /api/v1/snapshots/{id}
```

**Response (200)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "source_id": "uuid",
  "job_id": "uuid",
  "status": "completed",
  "size_bytes": 104886398,
  "manifest_json": {...},
  "encryption_algorithm": "age-x25519",
  "storage_backend": "local_fs",
  "worker_id": "worker-1",
  "local_path": "/var/lib/xvault/backups/tenants/.../snapshots/..."
}
```

Gets details of a specific snapshot including manifest.

---

## Internal API (Worker → Hub)

These endpoints are used by workers to claim jobs and report status.

### Jobs

#### Claim Job
```http
POST /internal/jobs/claim
Content-Type: application/json

{
  "worker_id": "worker-1"
}
```

**Response (200)**:
```json
{
  "job_id": "uuid",
  "tenant_id": "uuid",
  "source_id": "uuid",
  "type": "backup",
  "payload": {
    "source_id": "uuid",
    "credential_id": "uuid",
    "source_config": {...}
  },
  "lease_expires_at": "timestamp"
}
```

Claims the next available job from the queue. Uses `FOR UPDATE SKIP LOCKED` for concurrent worker safety.

#### Complete Job
```http
POST /internal/jobs/{job_id}/complete
Content-Type: application/json

{
  "worker_id": "worker-1",
  "status": "completed",
  "snapshot": {
    "snapshot_id": "uuid",
    "status": "completed",
    "size_bytes": 104886398,
    "started_at": "timestamp",
    "finished_at": "timestamp",
    "duration_ms": 593,
    "manifest_json": {...},
    "encryption_algorithm": "age-x25519",
    "locator": {
      "storage_backend": "local_fs",
      "worker_id": "worker-1",
      "local_path": "/var/lib/xvault/backups/..."
    }
  }
}
```

Reports job completion (success or failure) and creates snapshot record.

---

### Credentials

#### Get Credential
```http
GET /internal/credentials/{id}
```

**Response (200)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "kind": "source",
  "ciphertext": "base64-encoded-encrypted",
  "key_id": "platform-kek"
}
```

Fetches an encrypted credential. Workers decrypt using platform KEK.

---

### Tenant Keys

#### Get Tenant Public Key
```http
GET /internal/tenants/{id}/public-key
```

**Response (200)**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "algorithm": "age-x25519",
  "public_key": "age1...",
  "encrypted_private_key": "base64-encoded",
  "key_status": "active"
}
```

Fetches a tenant's public key for encrypting backup artifacts.

---

### Workers

#### Register Worker
```http
POST /internal/workers/register
Content-Type: application/json

{
  "worker_id": "worker-1",
  "name": "Worker worker-1",
  "storage_base_path": "/var/lib/xvault/backups",
  "capabilities": {
    "connectors": ["ssh", "sftp"],
    "storage": ["local_fs"]
  }
}
```

**Response (201)**: Worker record

Registers a worker with the Hub. Creates or updates worker record.

#### Worker Heartbeat
```http
POST /internal/workers/heartbeat
Content-Type: application/json

{
  "worker_id": "worker-1",
  "status": "online"
}
```

**Response (200)**: Success

Updates worker's last_seen_at timestamp and status.

---

## Health

#### Health Check
```http
GET /healthz
```

**Response (200)**:
```json
{"ok": true}
```

Simple health check endpoint.

---

## Status Codes

| Code | Description |
|------|-------------|
| 200  | Success |
| 201  | Created |
| 400  | Bad Request |
| 401  | Unauthorized |
| 404  | Not Found |
| 500  | Internal Server Error |

## Error Response Format

```json
{
  "error": "Error message",
  "details": "Detailed error information"
}
```

---

## Source Config Types

### SSH/SFTP
```json
{
  "host": "string",
  "port": 22,
  "username": "string",
  "paths": ["string"],
  "use_password": true
}
```

### FTP
```json
{
  "host": "string",
  "port": 21,
  "username": "string",
  "paths": ["string"],
  "use_password": true
}
```

### MySQL
```json
{
  "host": "string",
  "port": 3306,
  "username": "string",
  "database": "string",
  "tables": ["table1", "table2"]
}
```

### PostgreSQL
```json
{
  "host": "string",
  "port": 5432,
  "username": "string",
  "database": "string",
  "schemas": ["schema1", "schema2"]
}
```
