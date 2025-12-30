# xVault API Reference

This document describes all available API endpoints for the xVault backup service.

**Base URL**: `http://localhost:8080`

**Authentication**: JWT tokens required for protected endpoints. Include `Authorization: Bearer <token>` header.

---

## Auth API

### Register User
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "string",
  "password": "string",
  "tenant_name": "string"
}
```

**Response (201)**:
```json
{
  "user": {
    "id": "uuid",
    "email": "string",
    "role": "owner",
    "tenant_id": "uuid"
  },
  "access_token": "jwt...",
  "refresh_token": "jwt..."
}
```

Creates a new user account and tenant.

### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "string",
  "password": "string"
}
```

**Response (200)**:
```json
{
  "user": {
    "id": "uuid",
    "email": "string",
    "role": "owner",
    "tenant_id": "uuid"
  },
  "access_token": "jwt...",
  "refresh_token": "jwt..."
}
```

Authenticates user and returns tokens.

### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "jwt..."
}
```

**Response (200)**:
```json
{
  "access_token": "jwt...",
  "refresh_token": "jwt..."
}
```

Refreshes access token.

### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer <token>
```

**Response (200)**: Success

Invalidates tokens.

### Get Current User
```http
GET /api/v1/auth/me
Authorization: Bearer <token>
```

**Response (200)**:
```json
{
  "id": "uuid",
  "email": "string",
  "role": "owner",
  "tenant_id": "uuid"
}
```

Gets current user info.

---

## Public API

*All endpoints require JWT authentication.*

### Tenants

#### Create Tenant
```http
POST /api/v1/tenants
Content-Type: application/json
Authorization: Bearer <token>

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
Authorization: Bearer <token>

{
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
Authorization: Bearer <token>

{
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
GET /api/v1/sources
Authorization: Bearer <token>
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

Lists all sources for the authenticated user's tenant.

#### Get Source
```http
GET /api/v1/sources/{id}
Authorization: Bearer <token>
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
POST /api/v1/jobs
Content-Type: application/json
Authorization: Bearer <token>

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
GET /api/v1/snapshots?source_id={uuid}
Authorization: Bearer <token>
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

Lists all snapshots for a source.

#### Get Snapshot
```http
GET /api/v1/snapshots/{id}
Authorization: Bearer <token>
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

### Schedules

#### List Schedules
```http
GET /api/v1/schedules
Authorization: Bearer <token>
```

**Response (200)**:
```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "source_id": "uuid",
    "cron": "0 2 * * *",
    "timezone": "UTC",
    "enabled": true,
    "retention_policy": {...},
    "created_at": "timestamp"
  }
]
```

Lists all schedules for the tenant.

#### Create Schedule
```http
POST /api/v1/schedules
Content-Type: application/json
Authorization: Bearer <token>

{
  "source_id": "uuid",
  "cron": "0 2 * * *",
  "timezone": "UTC",
  "retention_policy": {
    "keep_last_n": 7,
    "keep_daily": 30,
    "keep_weekly": 12,
    "keep_monthly": 6,
    "min_age_hours": 24,
    "max_age_days": 365
  }
}
```

**Response (201)**: Schedule object

Creates a backup schedule with retention policy.

---

## Admin API

*Requires admin role.*

#### List Users
```http
GET /api/v1/admin/users
Authorization: Bearer <token>
```

**Response (200)**: Array of users

Lists all users across tenants.

#### Create User
```http
POST /api/v1/admin/users
Content-Type: application/json
Authorization: Bearer <token>

{
  "email": "string",
  "password": "string",
  "tenant_name": "string",
  "role": "member"
}
```

**Response (201)**: User object

Creates a new user and tenant.

#### Get User
```http
GET /api/v1/admin/users/{id}
Authorization: Bearer <token>
```

**Response (200)**: User object

Gets user details.

#### Update User
```http
PUT /api/v1/admin/users/{id}
Content-Type: application/json
Authorization: Bearer <token>

{
  "email": "string",
  "role": "admin"
}
```

**Response (200)**: User object

Updates user.

#### Delete User
```http
DELETE /api/v1/admin/users/{id}
Authorization: Bearer <token>
```

**Response (200)**: Success

Deletes user.

#### List Tenants
```http
GET /api/v1/admin/tenants
Authorization: Bearer <token>
```

**Response (200)**: Array of tenants

Lists all tenants.

#### Get Tenant
```http
GET /api/v1/admin/tenants/{id}
Authorization: Bearer <token>
```

**Response (200)**: Tenant object

Gets tenant details.

#### Delete Tenant
```http
DELETE /api/v1/admin/tenants/{id}
Authorization: Bearer <token>
```

**Response (200)**: Success

Deletes tenant and all associated data.

#### List Sources (Admin)
```http
GET /api/v1/admin/sources
Authorization: Bearer <token>
```

**Response (200)**:
```json
{
  "sources": [
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
  ]
}
```

Lists all sources across all tenants.

#### Get Source (Admin)
```http
GET /api/v1/admin/sources/{id}
Authorization: Bearer <token>
```

**Response (200)**: Source object

Gets source details.

#### Create Source (Admin)
```http
POST /api/v1/admin/sources
Content-Type: application/json
Authorization: Bearer <token>

{
  "tenant_id": "uuid",
  "type": "ssh",
  "name": "Production Server",
  "config": {
    "host": "192.168.1.100",
    "port": 22,
    "username": "backup",
    "paths": ["/var/www", "/home/app"],
    "use_password": true
  },
  "credential": "base64-encoded-password-or-private-key"
}
```

**Response (201)**: Source object

Creates a new source with encrypted credential. The credential is base64-encoded and will be encrypted before storage.

#### Update Source (Admin)
```http
PUT /api/v1/admin/sources/{id}
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "Updated Name",
  "status": "disabled",
  "config": {...},
  "credential": "base64-encoded-new-credential"
}
```

**Response (200)**: Source object

Updates a source. Include `credential` to rotate the stored credential.

#### Delete Source (Admin)
```http
DELETE /api/v1/admin/sources/{id}
Authorization: Bearer <token>
```

**Response (204)**: No Content

Deletes a source.

#### Run Retention for All Sources
```http
POST /api/v1/admin/retention/run
Authorization: Bearer <token>
```

**Response (200)**: Retention results

Evaluates and applies retention policies for all sources.

#### Run Retention for Source
```http
POST /api/v1/admin/retention/run/{sourceId}
Authorization: Bearer <token>
```

**Response (200)**: Retention results

Evaluates and applies retention policy for a specific source.

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
| 403  | Forbidden |
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
