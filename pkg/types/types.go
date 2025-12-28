package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JobType represents the type of job to execute
type JobType string

const (
	JobTypeBackup        JobType = "backup"
	JobTypeRestore       JobType = "restore"
	JobTypeDeleteSnapshot JobType = "delete_snapshot"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusRunning    JobStatus = "running"
	JobStatusFinalizing JobStatus = "finalizing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCanceled   JobStatus = "canceled"
)

// SourceType represents the type of source to back up
type SourceType string

const (
	SourceTypeSSH       SourceType = "ssh"
	SourceTypeSFTP      SourceType = "sftp"
	SourceTypeFTP       SourceType = "ftp"
	SourceTypeMySQL     SourceType = "mysql"
	SourceTypePostgres  SourceType = "postgres"
	SourceTypeWordPress SourceType = "wordpress"
)

// SnapshotStatus represents the status of a snapshot
type SnapshotStatus string

const (
	SnapshotStatusCompleted SnapshotStatus = "completed"
	SnapshotStatusFailed    SnapshotStatus = "failed"
)

// StorageBackend represents the storage backend type
type StorageBackend string

const (
	StorageBackendLocalFS StorageBackend = "local_fs"
	StorageBackendS3      StorageBackend = "s3"
)

// JobPayload is the JSON payload stored in the jobs table
// It contains references to credentials but NOT plaintext secrets
type JobPayload struct {
	SourceID   string      `json:"source_id"`
	CredentialID string    `json:"credential_id"`
	SourceConfig json.RawMessage `json:"source_config"` // Type-specific config
	// For restore jobs
	RestoreSnapshotID *string `json:"restore_snapshot_id,omitempty"`
	// For delete jobs
	DeleteSnapshotID *string `json:"delete_snapshot_id,omitempty"`
}

// SourceConfigSSH represents SSH/SFTP connection config
type SourceConfigSSH struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Paths    []string `json:"paths"`
	// For SSH key auth (preferred over password)
	// Password is NOT stored here - it's in credentials
	UsePassword bool `json:"use_password,omitempty"`
}

// SourceConfigFTP represents FTP connection config
type SourceConfigFTP struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Paths    []string `json:"paths"`
	Passive  bool     `json:"passive,omitempty"`
}

// SourceConfigMySQL represents MySQL connection config
type SourceConfigMySQL struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Database        string `json:"database"`
	Username        string `json:"username"`
	UseSSH          bool   `json:"use_ssh,omitempty"`
	SSHHost         string `json:"ssh_host,omitempty"`
	SSHPort         int    `json:"ssh_port,omitempty"`
	SSHUsername     string `json:"ssh_username,omitempty"`
}

// SourceConfigPostgres represents PostgreSQL connection config
type SourceConfigPostgres struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	UseSSH   bool   `json:"use_ssh,omitempty"`
	SSHHost  string `json:"ssh_host,omitempty"`
	SSHPort  int    `json:"ssh_port,omitempty"`
}

// SnapshotManifest represents the manifest.json stored with each snapshot
type SnapshotManifest struct {
	TenantID    string `json:"tenant_id"`
	SourceID    string `json:"source_id"`
	SnapshotID  string `json:"snapshot_id"`
	JobID       string `json:"job_id"`
	WorkerID    string `json:"worker_id"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
	DurationMs  int64  `json:"duration_ms"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`

	// Encryption metadata
	EncryptionAlgorithm  string `json:"encryption_algorithm"`
	EncryptionKeyID      string `json:"encryption_key_id"`
	EncryptionRecipient  string `json:"encryption_recipient,omitempty"`

	// Content summary
	ContentSummary ContentSummary `json:"content_summary"`
}

// ContentSummary describes what's in the snapshot
type ContentSummary struct {
	Type     string   `json:"type"` // "files", "database", "wordpress"
	Paths    []string `json:"paths,omitempty"`
	FileCount int     `json:"file_count,omitempty"`
	// For databases
	DatabaseName string `json:"database_name,omitempty"`
	DatabaseSize int64  `json:"database_size,omitempty"`
}

// SnapshotLocator represents where a snapshot is stored
// This is returned to the Hub and stored in the snapshots table
type SnapshotLocator struct {
	StorageBackend StorageBackend `json:"storage_backend"`
	WorkerID       string         `json:"worker_id,omitempty"` // Required for local_fs
	LocalPath      string         `json:"local_path,omitempty"`  // Required for local_fs
	Bucket         string         `json:"bucket,omitempty"`       // For S3
	ObjectKey      string         `json:"object_key,omitempty"`   // For S3
	ETag           string         `json:"etag,omitempty"`         // For S3
}

// JSONB wrapper for database storage
type JSONB map[string]any

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// JobClaimRequest is the request body for a worker to claim a job
type JobClaimRequest struct {
	WorkerID string `json:"worker_id"`
}

// JobClaimResponse is the response when a worker claims a job
type JobClaimResponse struct {
	JobID          string     `json:"job_id"`
	TenantID       string     `json:"tenant_id"`
	SourceID       string     `json:"source_id,omitempty"`
	Type           JobType    `json:"type"`
	Payload        JobPayload `json:"payload"`
	LeaseExpiresAt string     `json:"lease_expires_at"`
}

// JobCompleteRequest is the request body for a worker to report job completion
type JobCompleteRequest struct {
	WorkerID  string          `json:"worker_id"`
	Status    JobStatus       `json:"status"`
	Error     string          `json:"error,omitempty"`
	Snapshot  *SnapshotResult `json:"snapshot,omitempty"`
}

// SnapshotResult is the snapshot metadata reported by the worker
type SnapshotResult struct {
	SnapshotID        string           `json:"snapshot_id"`
	Status            SnapshotStatus   `json:"status"`
	SizeBytes         int64            `json:"size_bytes"`
	StartedAt         string           `json:"started_at"`
	FinishedAt        string           `json:"finished_at"`
	DurationMs        int64            `json:"duration_ms"`
	ManifestJSON      json.RawMessage  `json:"manifest_json"`
	EncryptionAlgorithm string         `json:"encryption_algorithm"`
	Locator           SnapshotLocator  `json:"locator"`
}

// WorkerRegisterRequest is the request body for a worker to register
type WorkerRegisterRequest struct {
	WorkerID        string                 `json:"worker_id"`
	Name            string                 `json:"name"`
	StorageBasePath string                 `json:"storage_base_path"`
	Capabilities    map[string]any         `json:"capabilities"`
}

// WorkerHeartbeatRequest is the request body for worker heartbeats
type WorkerHeartbeatRequest struct {
	WorkerID string `json:"worker_id"`
	Status   string `json:"status"` // "online", "offline", "draining"
}
