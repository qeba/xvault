package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JobType represents the type of job to execute
type JobType string

const (
	JobTypeBackup         JobType = "backup"
	JobTypeRestore        JobType = "restore"
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
	SourceID     string          `json:"source_id"`
	CredentialID string          `json:"credential_id"`
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
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	Username    string `json:"username"`
	UseSSH      bool   `json:"use_ssh,omitempty"`
	SSHHost     string `json:"ssh_host,omitempty"`
	SSHPort     int    `json:"ssh_port,omitempty"`
	SSHUsername string `json:"ssh_username,omitempty"`
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
	TenantID   string `json:"tenant_id"`
	SourceID   string `json:"source_id"`
	SnapshotID string `json:"snapshot_id"`
	JobID      string `json:"job_id"`
	WorkerID   string `json:"worker_id"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	DurationMs int64  `json:"duration_ms"`
	SizeBytes  int64  `json:"size_bytes"`
	SHA256     string `json:"sha256"`

	// Encryption metadata
	EncryptionAlgorithm string `json:"encryption_algorithm"`
	EncryptionKeyID     string `json:"encryption_key_id"`
	EncryptionRecipient string `json:"encryption_recipient,omitempty"`

	// Content summary
	ContentSummary ContentSummary `json:"content_summary"`
}

// ContentSummary describes what's in the snapshot
type ContentSummary struct {
	Type      string   `json:"type"` // "files", "database", "wordpress"
	Paths     []string `json:"paths,omitempty"`
	FileCount int      `json:"file_count,omitempty"`
	// For databases
	DatabaseName string `json:"database_name,omitempty"`
	DatabaseSize int64  `json:"database_size,omitempty"`
}

// SnapshotLocator represents where a snapshot is stored
// This is returned to the Hub and stored in the snapshots table
type SnapshotLocator struct {
	StorageBackend StorageBackend `json:"storage_backend"`
	WorkerID       string         `json:"worker_id,omitempty"`  // Required for local_fs
	LocalPath      string         `json:"local_path,omitempty"` // Required for local_fs
	Bucket         string         `json:"bucket,omitempty"`     // For S3
	ObjectKey      string         `json:"object_key,omitempty"` // For S3
	ETag           string         `json:"etag,omitempty"`       // For S3
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
	WorkerID string          `json:"worker_id"`
	Status   JobStatus       `json:"status"`
	Error    string          `json:"error,omitempty"`
	Snapshot *SnapshotResult `json:"snapshot,omitempty"`
	Restore  *RestoreResult  `json:"restore,omitempty"`
}

// SnapshotResult is the snapshot metadata reported by the worker
type SnapshotResult struct {
	SnapshotID          string          `json:"snapshot_id"`
	Status              SnapshotStatus  `json:"status"`
	SizeBytes           int64           `json:"size_bytes"`
	StartedAt           string          `json:"started_at"`
	FinishedAt          string          `json:"finished_at"`
	DurationMs          int64           `json:"duration_ms"`
	ManifestJSON        json.RawMessage `json:"manifest_json"`
	EncryptionAlgorithm string          `json:"encryption_algorithm"`
	Locator             SnapshotLocator `json:"locator"`
}

// RestoreResult is the restore metadata reported by the worker
type RestoreResult struct {
	RestoreID     string `json:"restore_id"`
	SnapshotID    string `json:"snapshot_id"`
	Status        string `json:"status"`
	DownloadURL   string `json:"download_url,omitempty"`
	DownloadToken string `json:"download_token,omitempty"`
	SizeBytes     int64  `json:"size_bytes"`
	ExpiresAt     string `json:"expires_at"`
}

// WorkerRegisterRequest is the request body for a worker to register
type WorkerRegisterRequest struct {
	WorkerID        string         `json:"worker_id"`
	Name            string         `json:"name"`
	StorageBasePath string         `json:"storage_base_path"`
	Capabilities    map[string]any `json:"capabilities"`
}

// WorkerHeartbeatRequest is the request body for worker heartbeats
type WorkerHeartbeatRequest struct {
	WorkerID      string         `json:"worker_id"`
	Status        string         `json:"status"` // "online", "offline", "draining"
	SystemMetrics *SystemMetrics `json:"system_metrics,omitempty"`
}

// SystemMetrics contains system resource usage information from a worker
type SystemMetrics struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryPercent    float64 `json:"memory_percent"`
	MemoryTotalBytes int64   `json:"memory_total_bytes"`
	MemoryUsedBytes  int64   `json:"memory_used_bytes"`
	DiskTotalBytes   int64   `json:"disk_total_bytes"`
	DiskUsedBytes    int64   `json:"disk_used_bytes"`
	DiskFreeBytes    int64   `json:"disk_free_bytes"`
	DiskPercent      float64 `json:"disk_percent"`
	ActiveJobs       int     `json:"active_jobs"`
	UptimeSeconds    int64   `json:"uptime_seconds"`
}

// RetentionPolicy defines how long snapshots should be kept
// This is stored as JSONB in the schedules table
type RetentionPolicy struct {
	// Mode specifies the retention mode (from frontend)
	// Values: "all" (keep all), "latest_n" (keep last N), "within_duration" (keep within duration)
	Mode string `json:"mode,omitempty"`

	// KeepLastN keeps the most recent N snapshots regardless of time
	KeepLastN *int `json:"keep_last_n,omitempty"`

	// KeepWithinDuration specifies a duration string like "30d", "7d", "24h"
	// This is converted to MaxAgeDays for evaluation
	KeepWithinDuration string `json:"keep_within_duration,omitempty"`

	// KeepDaily keeps one snapshot per day for the specified number of days
	KeepDaily *int `json:"keep_daily,omitempty"`

	// KeepWeekly keeps one snapshot per week for the specified number of weeks
	KeepWeekly *int `json:"keep_weekly,omitempty"`

	// KeepMonthly keeps one snapshot per month for the specified number of months
	KeepMonthly *int `json:"keep_monthly,omitempty"`

	// MinAgeHours specifies the minimum age in hours before a snapshot can be deleted
	// This prevents newly created snapshots from being immediately deleted
	MinAgeHours *int `json:"min_age_hours,omitempty"`

	// MaxAgeDays deletes all snapshots older than this many days, regardless of other rules
	MaxAgeDays *int `json:"max_age_days,omitempty"`
}

// Normalize converts frontend-friendly fields to the internal format
// This handles mode-based policies and duration string parsing
func (rp *RetentionPolicy) Normalize() {
	// If mode is "all", clear all retention constraints (keep everything)
	if rp.Mode == "all" {
		rp.KeepLastN = nil
		rp.MaxAgeDays = nil
		rp.KeepWithinDuration = ""
		return
	}

	// If mode is "latest_n" and keep_last_n is set, that's already correct
	if rp.Mode == "latest_n" && rp.KeepLastN != nil {
		// Clear duration-based settings
		rp.MaxAgeDays = nil
		rp.KeepWithinDuration = ""
		return
	}

	// If mode is "within_duration", convert keep_within_duration to max_age_days
	if rp.Mode == "within_duration" && rp.KeepWithinDuration != "" {
		days := ParseDurationToDays(rp.KeepWithinDuration)
		if days > 0 {
			rp.MaxAgeDays = &days
		}
		// Clear keep_last_n since we're using duration
		rp.KeepLastN = nil
		return
	}

	// Fallback: if keep_within_duration is set but mode isn't, still parse it
	if rp.KeepWithinDuration != "" && rp.MaxAgeDays == nil {
		days := ParseDurationToDays(rp.KeepWithinDuration)
		if days > 0 {
			rp.MaxAgeDays = &days
		}
	}
}

// ParseDurationToDays parses a duration string like "30d", "7d", "24h" and returns days
// Supports: Nd (days), Nh (hours), Nw (weeks), Nm (months approximate as 30 days)
func ParseDurationToDays(duration string) int {
	if duration == "" {
		return 0
	}

	// Try to parse as "Nd" format
	var value int
	var unit string

	n, err := fmt.Sscanf(duration, "%d%s", &value, &unit)
	if err != nil || n != 2 || value <= 0 {
		return 0
	}

	switch unit {
	case "d", "D", "day", "days":
		return value
	case "h", "H", "hour", "hours":
		// Convert hours to days (round up)
		return (value + 23) / 24
	case "w", "W", "week", "weeks":
		return value * 7
	case "m", "M", "month", "months":
		return value * 30 // Approximate
	default:
		return 0
	}
}

// DefaultRetentionPolicy returns a sensible default retention policy
func DefaultRetentionPolicy() RetentionPolicy {
	keepLastN := 7
	minAgeHours := 24
	return RetentionPolicy{
		KeepLastN:   &keepLastN,
		MinAgeHours: &minAgeHours,
	}
}

// RetentionEvaluationResult represents the result of evaluating a retention policy
type RetentionEvaluationResult struct {
	// SnapshotsToDelete contains snapshot IDs that should be deleted
	SnapshotsToDelete []string `json:"snapshots_to_delete"`

	// SnapshotsToKeep contains snapshot IDs that are protected by the retention policy
	SnapshotsToKeep []string `json:"snapshots_to_keep"`

	// Summary contains a human-readable summary
	Summary string `json:"summary"`
}
