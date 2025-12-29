package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"xvault/pkg/types"
)

// Repository handles database operations for the Hub
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Tenant represents a tenant record
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTenant creates a new tenant
func (r *Repository) CreateTenant(ctx context.Context, name string) (*Tenant, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO tenants (id, name, plan, created_at, updated_at)
	          VALUES ($1, $2, 'free', $3, $4)
	          RETURNING id, name, plan, created_at, updated_at`

	var tenant Tenant
	err := r.db.QueryRowContext(ctx, query, id, name, now, now).Scan(
		&tenant.ID, &tenant.Name, &tenant.Plan, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return &tenant, nil
}

// GetTenant retrieves a tenant by ID
func (r *Repository) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	query := `SELECT id, name, plan, created_at, updated_at FROM tenants WHERE id = $1`

	var tenant Tenant
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Plan, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return &tenant, nil
}

// TenantKey represents a tenant encryption key record
type TenantKey struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	Algorithm          string    `json:"algorithm"`
	PublicKey          string    `json:"public_key"`
	EncryptedPrivateKey string   `json:"encrypted_private_key"`
	KeyStatus          string    `json:"key_status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateTenantKey creates a new tenant encryption key
func (r *Repository) CreateTenantKey(ctx context.Context, tenantID, algorithm, publicKey, encryptedPrivateKey string) (*TenantKey, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO tenant_keys (id, tenant_id, algorithm, public_key, encrypted_private_key, key_status, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, 'active', $6, $7)
	          RETURNING id, tenant_id, algorithm, public_key, encrypted_private_key, key_status, created_at, updated_at`

	var key TenantKey
	err := r.db.QueryRowContext(ctx, query, id, tenantID, algorithm, publicKey, encryptedPrivateKey, now, now).Scan(
		&key.ID, &key.TenantID, &key.Algorithm, &key.PublicKey, &key.EncryptedPrivateKey, &key.KeyStatus, &key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant key: %w", err)
	}

	return &key, nil
}

// GetActiveTenantKey retrieves the active key for a tenant
func (r *Repository) GetActiveTenantKey(ctx context.Context, tenantID string) (*TenantKey, error) {
	query := `SELECT id, tenant_id, algorithm, public_key, encrypted_private_key, key_status, created_at, updated_at
	          FROM tenant_keys
	          WHERE tenant_id = $1 AND key_status = 'active'
	          ORDER BY created_at DESC
	          LIMIT 1`

	var key TenantKey
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&key.ID, &key.TenantID, &key.Algorithm, &key.PublicKey, &key.EncryptedPrivateKey, &key.KeyStatus, &key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tenant key: %w", err)
	}

	return &key, nil
}

// Credential represents an encrypted credential record
type Credential struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Kind      string    `json:"kind"`
	Ciphertext string   `json:"ciphertext"`
	KeyID     string    `json:"key_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCredential creates a new encrypted credential
func (r *Repository) CreateCredential(ctx context.Context, tenantID, kind, ciphertext, keyID string) (*Credential, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO credentials (id, tenant_id, kind, ciphertext, key_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)
	          RETURNING id, tenant_id, kind, ciphertext, key_id, created_at, updated_at`

	var cred Credential
	err := r.db.QueryRowContext(ctx, query, id, tenantID, kind, ciphertext, keyID, now, now).Scan(
		&cred.ID, &cred.TenantID, &cred.Kind, &cred.Ciphertext, &cred.KeyID, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	return &cred, nil
}

// GetCredential retrieves a credential by ID (returns encrypted ciphertext)
func (r *Repository) GetCredential(ctx context.Context, id string) (*Credential, error) {
	query := `SELECT id, tenant_id, kind, ciphertext, key_id, created_at, updated_at
	          FROM credentials WHERE id = $1`

	var cred Credential
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cred.ID, &cred.TenantID, &cred.Kind, &cred.Ciphertext, &cred.KeyID, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	return &cred, nil
}

// Source represents a backup source
type Source struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	Status       string          `json:"status"`
	Config       json.RawMessage `json:"config"`
	CredentialID string          `json:"credential_id"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CreateSource creates a new backup source
func (r *Repository) CreateSource(ctx context.Context, tenantID, sourceType, name, credentialID string, config json.RawMessage) (*Source, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO sources (id, tenant_id, type, name, status, config, credential_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8)
	          RETURNING id, tenant_id, type, name, status, config, credential_id, created_at, updated_at`

	var source Source
	err := r.db.QueryRowContext(ctx, query, id, tenantID, sourceType, name, config, credentialID, now, now).Scan(
		&source.ID, &source.TenantID, &source.Type, &source.Name, &source.Status, &source.Config, &source.CredentialID, &source.CreatedAt, &source.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return &source, nil
}

// GetSource retrieves a source by ID
func (r *Repository) GetSource(ctx context.Context, id string) (*Source, error) {
	query := `SELECT id, tenant_id, type, name, status, config, credential_id, created_at, updated_at
	          FROM sources WHERE id = $1`

	var source Source
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID, &source.TenantID, &source.Type, &source.Name, &source.Status, &source.Config, &source.CredentialID, &source.CreatedAt, &source.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return &source, nil
}

// ListSources retrieves all sources for a tenant
func (r *Repository) ListSources(ctx context.Context, tenantID string) ([]*Source, error) {
	query := `SELECT id, tenant_id, type, name, status, config, credential_id, created_at, updated_at
	          FROM sources WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	var sources []*Source
	for rows.Next() {
		var source Source
		err := rows.Scan(
			&source.ID, &source.TenantID, &source.Type, &source.Name, &source.Status, &source.Config, &source.CredentialID, &source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, nil
}

// Job represents a job record
type Job struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	SourceID       *string         `json:"source_id,omitempty"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Priority       int             `json:"priority"`
	TargetWorkerID *string         `json:"target_worker_id,omitempty"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at,omitempty"`
	Attempt        int             `json:"attempt"`
	Payload        json.RawMessage `json:"payload"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	ErrorCode      *string         `json:"error_code,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// CreateJob creates a new job
func (r *Repository) CreateJob(ctx context.Context, tenantID string, jobType types.JobType, sourceID *string, payload json.RawMessage, priority int) (*Job, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO jobs (id, tenant_id, source_id, type, status, priority, payload, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, 'queued', $5, $6, $7, $8)
	          RETURNING id, tenant_id, source_id, type, status, priority, target_worker_id, lease_expires_at,
	                    attempt, payload, started_at, finished_at, error_code, error_message, created_at, updated_at`

	var job Job
	err := r.db.QueryRowContext(ctx, query, id, tenantID, sourceID, string(jobType), priority, payload, now, now).Scan(
		&job.ID, &job.TenantID, &job.SourceID, &job.Type, &job.Status, &job.Priority, &job.TargetWorkerID, &job.LeaseExpiresAt,
		&job.Attempt, &job.Payload, &job.StartedAt, &job.FinishedAt, &job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return &job, nil
}

// CreateJobWithTargetWorker creates a new job with a specific target worker
// This is used for delete_snapshot jobs that must go to the worker that owns the snapshot
func (r *Repository) CreateJobWithTargetWorker(ctx context.Context, tenantID string, jobType types.JobType, sourceID *string, targetWorkerID string, payload json.RawMessage, priority int) (*Job, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO jobs (id, tenant_id, source_id, type, status, priority, target_worker_id, payload, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, 'queued', $5, $6, $7, $8, $9)
	          RETURNING id, tenant_id, source_id, type, status, priority, target_worker_id, lease_expires_at,
	                    attempt, payload, started_at, finished_at, error_code, error_message, created_at, updated_at`

	var job Job
	err := r.db.QueryRowContext(ctx, query, id, tenantID, sourceID, string(jobType), priority, targetWorkerID, payload, now, now).Scan(
		&job.ID, &job.TenantID, &job.SourceID, &job.Type, &job.Status, &job.Priority, &job.TargetWorkerID, &job.LeaseExpiresAt,
		&job.Attempt, &job.Payload, &job.StartedAt, &job.FinishedAt, &job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create job with target worker: %w", err)
	}

	return &job, nil
}

// ClaimJob updates a job to running status and sets a lease
func (r *Repository) ClaimJob(ctx context.Context, workerID string, leaseDuration time.Duration) (*Job, error) {
	now := time.Now()
	leaseExpires := now.Add(leaseDuration)

	query := `UPDATE jobs
	          SET status = 'running',
	              target_worker_id = $1,
	              lease_expires_at = $2,
	              started_at = $3,
	              attempt = attempt + 1,
	              updated_at = $3
	          WHERE id = (
	              SELECT id FROM jobs
	              WHERE status = 'queued'
	              ORDER BY priority DESC, created_at ASC
	              LIMIT 1
	              FOR UPDATE SKIP LOCKED
	          )
	          RETURNING id, tenant_id, source_id, type, status, priority, target_worker_id, lease_expires_at,
	                    attempt, payload, started_at, finished_at, error_code, error_message, created_at, updated_at`

	var job Job
	err := r.db.QueryRowContext(ctx, query, workerID, leaseExpires, now).Scan(
		&job.ID, &job.TenantID, &job.SourceID, &job.Type, &job.Status, &job.Priority, &job.TargetWorkerID, &job.LeaseExpiresAt,
		&job.Attempt, &job.Payload, &job.StartedAt, &job.FinishedAt, &job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		// Return sql.ErrNoRows directly so caller can distinguish "no jobs available" from actual errors
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	return &job, nil
}

// CompleteJob marks a job as completed or failed
func (r *Repository) CompleteJob(ctx context.Context, jobID string, status types.JobStatus, errorMsg *string) error {
	now := time.Now()

	query := `UPDATE jobs
	          SET status = $1,
	              finished_at = $2,
	              error_message = $3,
	              updated_at = $2
	          WHERE id = $4`

	_, err := r.db.ExecContext(ctx, query, string(status), now, errorMsg, jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (r *Repository) GetJob(ctx context.Context, jobID string) (*Job, error) {
	query := `SELECT id, tenant_id, source_id, type, status, priority, target_worker_id, lease_expires_at,
	          attempt, payload, started_at, finished_at, error_code, error_message, created_at, updated_at
	          FROM jobs WHERE id = $1`

	var job Job
	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.TenantID, &job.SourceID, &job.Type, &job.Status, &job.Priority, &job.TargetWorkerID, &job.LeaseExpiresAt,
		&job.Attempt, &job.Payload, &job.StartedAt, &job.FinishedAt, &job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// Snapshot represents a snapshot record
type Snapshot struct {
	ID                  string          `json:"id"`
	TenantID            string          `json:"tenant_id"`
	SourceID            string          `json:"source_id"`
	JobID               string          `json:"job_id"`
	Status              string          `json:"status"`
	SizeBytes           int64           `json:"size_bytes"`
	StartedAt           time.Time       `json:"started_at"`
	FinishedAt          time.Time       `json:"finished_at"`
	DurationMs          *int64          `json:"duration_ms,omitempty"`
	ManifestJSON        json.RawMessage `json:"manifest_json,omitempty"`
	EncryptionAlgorithm string          `json:"encryption_algorithm"`
	EncryptionKeyID     *string         `json:"encryption_key_id,omitempty"`
	EncryptionRecipient *string         `json:"encryption_recipient,omitempty"`
	StorageBackend      string          `json:"storage_backend"`
	WorkerID            *string         `json:"worker_id,omitempty"`
	LocalPath           *string         `json:"local_path,omitempty"`
	Bucket              *string         `json:"bucket,omitempty"`
	ObjectKey           *string         `json:"object_key,omitempty"`
	ETag                *string         `json:"etag,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// CreateSnapshot creates a new snapshot record
func (r *Repository) CreateSnapshot(ctx context.Context, tenantID, sourceID, jobID string, result types.SnapshotResult) (*Snapshot, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO snapshots
	          (id, tenant_id, source_id, job_id, status, size_bytes, started_at, finished_at, duration_ms,
	           manifest_json, encryption_algorithm, encryption_recipient, storage_backend, worker_id, local_path,
	           created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'age-x25519', $11, $12, $13, $14, $15, $16)
	          RETURNING id, tenant_id, source_id, job_id, status, size_bytes, started_at, finished_at, duration_ms,
	                    manifest_json, encryption_algorithm, encryption_key_id, encryption_recipient,
	                    storage_backend, worker_id, local_path, bucket, object_key, etag, created_at, updated_at`

	var snapshot Snapshot
	err := r.db.QueryRowContext(ctx, query,
		id, tenantID, sourceID, jobID, string(result.Status), result.SizeBytes, result.StartedAt, result.FinishedAt,
		result.DurationMs, result.ManifestJSON, result.EncryptionAlgorithm, result.Locator.StorageBackend,
		result.Locator.WorkerID, result.Locator.LocalPath, now, now,
	).Scan(
		&snapshot.ID, &snapshot.TenantID, &snapshot.SourceID, &snapshot.JobID, &snapshot.Status, &snapshot.SizeBytes,
		&snapshot.StartedAt, &snapshot.FinishedAt, &snapshot.DurationMs, &snapshot.ManifestJSON, &snapshot.EncryptionAlgorithm,
		&snapshot.EncryptionKeyID, &snapshot.EncryptionRecipient, &snapshot.StorageBackend, &snapshot.WorkerID,
		&snapshot.LocalPath, &snapshot.Bucket, &snapshot.ObjectKey, &snapshot.ETag, &snapshot.CreatedAt, &snapshot.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	return &snapshot, nil
}

// ListSnapshots retrieves snapshots for a source
func (r *Repository) ListSnapshots(ctx context.Context, tenantID, sourceID string, limit int) ([]*Snapshot, error) {
	query := `SELECT id, tenant_id, source_id, job_id, status, size_bytes, started_at, finished_at, duration_ms,
	          manifest_json, encryption_algorithm, encryption_key_id, encryption_recipient,
	          storage_backend, worker_id, local_path, bucket, object_key, etag, created_at, updated_at
	          FROM snapshots
	          WHERE tenant_id = $1 AND source_id = $2
	          ORDER BY created_at DESC
	          LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		var snap Snapshot
		err := rows.Scan(
			&snap.ID, &snap.TenantID, &snap.SourceID, &snap.JobID, &snap.Status, &snap.SizeBytes,
			&snap.StartedAt, &snap.FinishedAt, &snap.DurationMs, &snap.ManifestJSON, &snap.EncryptionAlgorithm,
			&snap.EncryptionKeyID, &snap.EncryptionRecipient, &snap.StorageBackend, &snap.WorkerID,
			&snap.LocalPath, &snap.Bucket, &snap.ObjectKey, &snap.ETag, &snap.CreatedAt, &snap.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		snapshots = append(snapshots, &snap)
	}

	return snapshots, nil
}

// GetSnapshot retrieves a snapshot by ID
func (r *Repository) GetSnapshot(ctx context.Context, id string) (*Snapshot, error) {
	query := `SELECT id, tenant_id, source_id, job_id, status, size_bytes, started_at, finished_at, duration_ms,
	          manifest_json, encryption_algorithm, encryption_key_id, encryption_recipient,
	          storage_backend, worker_id, local_path, bucket, object_key, etag, created_at, updated_at
	          FROM snapshots WHERE id = $1`

	var snap Snapshot
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&snap.ID, &snap.TenantID, &snap.SourceID, &snap.JobID, &snap.Status, &snap.SizeBytes,
		&snap.StartedAt, &snap.FinishedAt, &snap.DurationMs, &snap.ManifestJSON, &snap.EncryptionAlgorithm,
		&snap.EncryptionKeyID, &snap.EncryptionRecipient, &snap.StorageBackend, &snap.WorkerID,
		&snap.LocalPath, &snap.Bucket, &snap.ObjectKey, &snap.ETag, &snap.CreatedAt, &snap.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	return &snap, nil
}

// Worker represents a worker record
type Worker struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	Capabilities    json.RawMessage `json:"capabilities"`
	StorageBasePath string          `json:"storage_base_path"`
	LastSeenAt      *time.Time      `json:"last_seen_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// RegisterWorker creates or updates a worker record
func (r *Repository) RegisterWorker(ctx context.Context, workerID, name, storageBasePath string, capabilities json.RawMessage) (*Worker, error) {
	now := time.Now()

	// Try to insert first, then update if exists
	query := `INSERT INTO workers (id, name, status, capabilities, storage_base_path, last_seen_at, created_at, updated_at)
	          VALUES ($1, $2, 'online', $3, $4, $5, $6, $7)
	          ON CONFLICT (id) DO UPDATE
	          SET name = EXCLUDED.name,
	              status = 'online',
	              capabilities = EXCLUDED.capabilities,
	              storage_base_path = EXCLUDED.storage_base_path,
	              last_seen_at = EXCLUDED.last_seen_at,
	              updated_at = EXCLUDED.updated_at
	          RETURNING id, name, status, capabilities, storage_base_path, last_seen_at, created_at, updated_at`

	var worker Worker
	err := r.db.QueryRowContext(ctx, query, workerID, name, capabilities, storageBasePath, now, now, now).Scan(
		&worker.ID, &worker.Name, &worker.Status, &worker.Capabilities, &worker.StorageBasePath, &worker.LastSeenAt, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	return &worker, nil
}

// UpdateWorkerHeartbeat updates the worker's last_seen timestamp
func (r *Repository) UpdateWorkerHeartbeat(ctx context.Context, workerID, status string) error {
	now := time.Now()

	query := `UPDATE workers
	          SET last_seen_at = $1,
	              status = $2,
	              updated_at = $1
	          WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, now, status, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %w", err)
	}

	return nil
}

// GetWorker retrieves a worker by ID
func (r *Repository) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	query := `SELECT id, name, status, capabilities, storage_base_path, last_seen_at, created_at, updated_at
	          FROM workers WHERE id = $1`

	var worker Worker
	err := r.db.QueryRowContext(ctx, query, workerID).Scan(
		&worker.ID, &worker.Name, &worker.Status, &worker.Capabilities, &worker.StorageBasePath, &worker.LastSeenAt, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	return &worker, nil
}

// Schedule represents a schedule record with retention policy
type Schedule struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	SourceID        string          `json:"source_id"`
	Cron            *string         `json:"cron,omitempty"`
	IntervalMinutes *int            `json:"interval_minutes,omitempty"`
	Timezone        string          `json:"timezone"`
	Status          string          `json:"status"`
	RetentionPolicy json.RawMessage `json:"retention_policy"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// GetScheduleForSource retrieves the schedule for a specific source
func (r *Repository) GetScheduleForSource(ctx context.Context, sourceID string) (*Schedule, error) {
	query := `SELECT id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at
	          FROM schedules WHERE source_id = $1`

	var schedule Schedule
	err := r.db.QueryRowContext(ctx, query, sourceID).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
		&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// ListAllSchedules retrieves all schedules (for retention evaluation)
func (r *Repository) ListAllSchedules(ctx context.Context) ([]*Schedule, error) {
	query := `SELECT id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at
	          FROM schedules WHERE status = 'enabled'`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		err := rows.Scan(
			&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
			&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// ListSnapshotsForRetention retrieves all completed snapshots for a source, ordered by creation time
func (r *Repository) ListSnapshotsForRetention(ctx context.Context, tenantID, sourceID string) ([]*Snapshot, error) {
	query := `SELECT id, tenant_id, source_id, job_id, status, size_bytes, started_at, finished_at, duration_ms,
	          manifest_json, encryption_algorithm, encryption_key_id, encryption_recipient,
	          storage_backend, worker_id, local_path, bucket, object_key, etag, created_at, updated_at
	          FROM snapshots
	          WHERE tenant_id = $1 AND source_id = $2 AND status = 'completed'
	          ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for retention: %w", err)
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		var snap Snapshot
		err := rows.Scan(
			&snap.ID, &snap.TenantID, &snap.SourceID, &snap.JobID, &snap.Status, &snap.SizeBytes,
			&snap.StartedAt, &snap.FinishedAt, &snap.DurationMs, &snap.ManifestJSON, &snap.EncryptionAlgorithm,
			&snap.EncryptionKeyID, &snap.EncryptionRecipient, &snap.StorageBackend, &snap.WorkerID,
			&snap.LocalPath, &snap.Bucket, &snap.ObjectKey, &snap.ETag, &snap.CreatedAt, &snap.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		snapshots = append(snapshots, &snap)
	}

	return snapshots, nil
}

// DeleteSnapshot removes a snapshot record from the database
func (r *Repository) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	query := `DELETE FROM snapshots WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, snapshotID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	return nil
}

// CreateSchedule creates a new schedule for a source
func (r *Repository) CreateSchedule(ctx context.Context, tenantID, sourceID string, cron *string, intervalMinutes *int, timezone string, retentionPolicy json.RawMessage) (*Schedule, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO schedules (id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, 'enabled', $7, $8, $9)
	          RETURNING id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at`

	var schedule Schedule
	err := r.db.QueryRowContext(ctx, query, id, tenantID, sourceID, cron, intervalMinutes, timezone, retentionPolicy, now, now).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
		&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return &schedule, nil
}

// UpdateSchedule updates an existing schedule
func (r *Repository) UpdateSchedule(ctx context.Context, scheduleID string, cron *string, intervalMinutes *int, timezone string, status string, retentionPolicy json.RawMessage) (*Schedule, error) {
	now := time.Now()

	query := `UPDATE schedules
	          SET cron = $2, interval_minutes = $3, timezone = $4, status = $5, retention_policy = $6, updated_at = $7
	          WHERE id = $1
	          RETURNING id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at`

	var schedule Schedule
	err := r.db.QueryRowContext(ctx, query, scheduleID, cron, intervalMinutes, timezone, status, retentionPolicy, now).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
		&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return &schedule, nil
}

// UpdateScheduleRetention updates only the retention policy for a schedule
func (r *Repository) UpdateScheduleRetention(ctx context.Context, scheduleID string, retentionPolicy json.RawMessage) (*Schedule, error) {
	now := time.Now()

	query := `UPDATE schedules
	          SET retention_policy = $2, updated_at = $3
	          WHERE id = $1
	          RETURNING id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at`

	var schedule Schedule
	err := r.db.QueryRowContext(ctx, query, scheduleID, retentionPolicy, now).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
		&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule retention policy: %w", err)
	}

	return &schedule, nil
}

// ListSchedulesByTenant retrieves all schedules for a specific tenant
func (r *Repository) ListSchedulesByTenant(ctx context.Context, tenantID string) ([]*Schedule, error) {
	query := `SELECT id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at
	          FROM schedules WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules for tenant: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		err := rows.Scan(
			&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
			&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// GetSchedule retrieves a schedule by ID
func (r *Repository) GetSchedule(ctx context.Context, scheduleID string) (*Schedule, error) {
	query := `SELECT id, tenant_id, source_id, cron, interval_minutes, timezone, status, retention_policy, created_at, updated_at
	          FROM schedules WHERE id = $1`

	var schedule Schedule
	err := r.db.QueryRowContext(ctx, query, scheduleID).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.SourceID, &schedule.Cron, &schedule.IntervalMinutes,
		&schedule.Timezone, &schedule.Status, &schedule.RetentionPolicy, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}
