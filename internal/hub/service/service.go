package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"

	"xvault/internal/hub/repository"
	"xvault/pkg/crypto"
	"xvault/pkg/types"

	"github.com/pkg/sftp"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/ssh"
)

const (
	// JobQueueKey is the Redis key for the job queue
	JobQueueKey = "xvault:jobs:queue"
	// JobLeaseDuration is how long a worker has to complete a job
	JobLeaseDuration = 30 * time.Minute
)

// Service handles business logic for the Hub
type Service struct {
	repo          *repository.Repository
	redis         *redis.Client
	encryptionKEK string
}

// NewService creates a new service instance
func NewService(repo *repository.Repository, redis *redis.Client, encryptionKEK string) *Service {
	return &Service{
		repo:          repo,
		redis:         redis,
		encryptionKEK: encryptionKEK,
	}
}

// LogSystemEvent logs a system event to the system_logs table (for hub-side logging)
// This is used to log infrastructure errors, scheduler events, and other hub-side events
func (s *Service) LogSystemEvent(ctx context.Context, level, message string, details map[string]any) {
	detailsJSON := json.RawMessage("{}")
	if details != nil {
		if data, err := json.Marshal(details); err == nil {
			detailsJSON = data
		}
	}

	if err := s.repo.CreateLog(ctx, level, message, nil, nil, nil, nil, nil, detailsJSON); err != nil {
		// Fall back to stdout if DB logging fails
		log.Printf("[%s] %s (db log failed: %v)", level, message, err)
	}
}

// LogSystemError logs a system error to the system_logs table
func (s *Service) LogSystemError(ctx context.Context, message string, err error, details map[string]any) {
	if details == nil {
		details = make(map[string]any)
	}
	if err != nil {
		details["error"] = err.Error()
	}
	s.LogSystemEvent(ctx, "error", message, details)
}

// CreateTenantRequest is the request to create a tenant
type CreateTenantRequest struct {
	Name string `json:"name"`
}

// CreateTenantResponse is the response when creating a tenant
type CreateTenantResponse struct {
	Tenant    *repository.Tenant `json:"tenant"`
	PublicKey string             `json:"public_key"`
}

// CreateTenant creates a new tenant with an encryption keypair
func (s *Service) CreateTenant(ctx context.Context, req CreateTenantRequest) (*CreateTenantResponse, error) {
	// Create tenant
	tenant, err := s.repo.CreateTenant(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Generate encryption keypair
	publicKey, privateKey, err := crypto.GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Encrypt private key with platform KEK
	encryptedPrivateKey, err := crypto.EncryptForStorage([]byte(privateKey), s.encryptionKEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Store tenant key
	_, err = s.repo.CreateTenantKey(ctx, tenant.ID, "age-x25519", publicKey, encryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to store tenant key: %w", err)
	}

	return &CreateTenantResponse{
		Tenant:    tenant,
		PublicKey: publicKey,
	}, nil
}

// CreateCredentialRequest is the request to create encrypted credentials
type CreateCredentialRequest struct {
	TenantID  string `json:"tenant_id"`
	Kind      string `json:"kind"`      // "source" or "storage"
	Plaintext string `json:"plaintext"` // Base64-encoded plaintext secret
}

// CreateCredential creates encrypted credentials for a tenant
func (s *Service) CreateCredential(ctx context.Context, req CreateCredentialRequest) (*repository.Credential, error) {
	// Decode plaintext (base64 encoded)
	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("invalid plaintext encoding: %w", err)
	}

	// For v0: Encrypt with platform KEK so workers can decrypt
	// (In production v1, use envelope encryption with tenant key)
	ciphertext, err := crypto.EncryptForStorage(plaintext, s.encryptionKEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	// Store encrypted credential (key_id references the platform KEK version)
	cred, err := s.repo.CreateCredential(ctx, req.TenantID, req.Kind, ciphertext, "platform-kek")
	if err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return cred, nil
}

// CreateSourceRequest is the request to create a backup source
type CreateSourceRequest struct {
	TenantID     string          `json:"tenant_id"`
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	CredentialID string          `json:"credential_id"`
	Config       json.RawMessage `json:"config"`
}

// CreateSource creates a new backup source
func (s *Service) CreateSource(ctx context.Context, req CreateSourceRequest) (*repository.Source, error) {
	source, err := s.repo.CreateSource(ctx, req.TenantID, req.Type, req.Name, req.CredentialID, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return source, nil
}

// GetSource retrieves a source by ID
func (s *Service) GetSource(ctx context.Context, sourceID string) (*repository.Source, error) {
	source, err := s.repo.GetSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return source, nil
}

// ListSources retrieves all sources for a tenant
func (s *Service) ListSources(ctx context.Context, tenantID string) ([]*repository.Source, error) {
	sources, err := s.repo.ListSources(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}

	return sources, nil
}

// EnqueueBackupJobRequest is the request to enqueue a backup job
type EnqueueBackupJobRequest struct {
	SourceID string `json:"source_id"`
	Priority int    `json:"priority,omitempty"` // Default 0
}

// EnqueueBackupJob creates and enqueues a backup job
func (s *Service) EnqueueBackupJob(ctx context.Context, tenantID string, req EnqueueBackupJobRequest) (*repository.Job, error) {
	// Get source to verify it exists and get credential info
	source, err := s.repo.GetSource(ctx, req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	// Verify source belongs to tenant
	if source.TenantID != tenantID {
		return nil, fmt.Errorf("source does not belong to tenant")
	}

	// Build job payload (includes credential_id, not plaintext secrets)
	payload := types.JobPayload{
		SourceID:     req.SourceID,
		CredentialID: source.CredentialID,
		SourceConfig: source.Config,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create job record
	priority := req.Priority
	if priority == 0 {
		priority = 5 // Default priority
	}

	job, err := s.repo.CreateJob(ctx, tenantID, types.JobTypeBackup, &req.SourceID, payloadJSON, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Enqueue to Redis
	jobMsg := map[string]any{
		"job_id":     job.ID,
		"tenant_id":  tenantID,
		"type":       "backup",
		"priority":   priority,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	jobMsgJSON, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job message: %w", err)
	}

	if err := s.redis.LPush(ctx, JobQueueKey, jobMsgJSON).Err(); err != nil {
		s.LogSystemError(ctx, "Redis: failed to enqueue backup job", err, map[string]any{
			"job_id":    job.ID,
			"tenant_id": tenantID,
			"source_id": req.SourceID,
		})
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return job, nil
}

// ClaimJob handles a worker's request to claim a job
func (s *Service) ClaimJob(ctx context.Context, req types.JobClaimRequest) (*types.JobClaimResponse, error) {
	// Claim the next available job
	job, err := s.repo.ClaimJob(ctx, req.WorkerID, JobLeaseDuration)
	if err != nil {
		// Return sql.ErrNoRows directly so handler can identify "no jobs available"
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	// Parse payload
	var payload types.JobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Build response
	sourceID := ""
	if job.SourceID != nil {
		sourceID = *job.SourceID
	}
	resp := &types.JobClaimResponse{
		JobID:          job.ID,
		TenantID:       job.TenantID,
		SourceID:       sourceID,
		Type:           types.JobType(job.Type),
		Payload:        payload,
		LeaseExpiresAt: job.LeaseExpiresAt.Format(time.RFC3339),
	}

	return resp, nil
}

// GetCredentialForWorker retrieves an encrypted credential for a worker
// The worker will decrypt it using the tenant's public key (cached or fetched)
func (s *Service) GetCredentialForWorker(ctx context.Context, credentialID string) (*repository.Credential, error) {
	cred, err := s.repo.GetCredential(ctx, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	return cred, nil
}

// GetTenantPublicKeyForWorker retrieves a tenant's public key for a worker
func (s *Service) GetTenantPublicKeyForWorker(ctx context.Context, tenantID string) (*repository.TenantKey, error) {
	key, err := s.repo.GetActiveTenantKey(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant key: %w", err)
	}

	return key, nil
}

// GetTenantPrivateKeyForWorker retrieves and decrypts a tenant's private key for restore operations
// This is used ONLY by workers for restore jobs (decrypting backup artifacts)
func (s *Service) GetTenantPrivateKeyForWorker(ctx context.Context, tenantID string) (string, error) {
	key, err := s.repo.GetActiveTenantKey(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant key: %w", err)
	}

	// Decrypt the private key using the platform KEK
	privateKeyBytes, err := crypto.DecryptFromStorage(key.EncryptedPrivateKey, s.encryptionKEK)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return string(privateKeyBytes), nil
}

// CompleteJob handles a worker's job completion report
func (s *Service) CompleteJob(ctx context.Context, jobID string, req types.JobCompleteRequest) error {
	// Determine final status
	var finalStatus types.JobStatus
	switch req.Status {
	case "completed":
		finalStatus = types.JobStatusCompleted
	case "failed":
		finalStatus = types.JobStatusFailed
	default:
		return fmt.Errorf("invalid job status: %s", req.Status)
	}

	// Update job status
	var errorMsg *string
	if req.Error != "" {
		errorMsg = &req.Error
	}

	if err := s.repo.CompleteJob(ctx, jobID, finalStatus, errorMsg); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Get job details to determine job type
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job details: %w", err)
	}

	// If snapshot was created, store it
	if req.Snapshot != nil {
		_, err = s.repo.CreateSnapshot(ctx, job.TenantID, *job.SourceID, jobID, *req.Snapshot)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}

		// Trigger retention evaluation for this source (debounced)
		// This runs in background and won't block job completion
		go s.maybeTriggerRetentionForSource(context.Background(), *job.SourceID)
	}

	// If delete_snapshot job completed successfully, delete the snapshot record
	if job.Type == string(types.JobTypeDeleteSnapshot) && finalStatus == types.JobStatusCompleted {
		// Parse the payload to get the snapshot ID
		var payload types.JobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse delete job payload: %w", err)
		}

		if payload.DeleteSnapshotID != nil {
			if err := s.repo.DeleteSnapshot(ctx, *payload.DeleteSnapshotID); err != nil {
				return fmt.Errorf("failed to delete snapshot record: %w", err)
			}
		}
	}

	return nil
}

// RegisterWorker handles a worker registration request
func (s *Service) RegisterWorker(ctx context.Context, req types.WorkerRegisterRequest) (*repository.Worker, error) {
	capabilitiesJSON, err := json.Marshal(req.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	worker, err := s.repo.RegisterWorker(ctx, req.WorkerID, req.Name, req.StorageBasePath, capabilitiesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	return worker, nil
}

// WorkerHeartbeat handles a worker heartbeat request
func (s *Service) WorkerHeartbeat(ctx context.Context, req types.WorkerHeartbeatRequest) error {
	// Serialize system metrics if present
	var metricsJSON json.RawMessage
	if req.SystemMetrics != nil {
		data, err := json.Marshal(req.SystemMetrics)
		if err != nil {
			return fmt.Errorf("failed to marshal system metrics: %w", err)
		}
		metricsJSON = data
	}

	if err := s.repo.UpdateWorkerHeartbeat(ctx, req.WorkerID, req.Status, metricsJSON); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
}

// ListWorkers retrieves all workers with their status and metrics
func (s *Service) ListWorkers(ctx context.Context) ([]*repository.Worker, error) {
	workers, err := s.repo.ListWorkers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	return workers, nil
}

// GetWorker retrieves a worker by ID
func (s *Service) GetWorker(ctx context.Context, workerID string) (*repository.Worker, error) {
	worker, err := s.repo.GetWorker(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	return worker, nil
}

// ListSnapshots retrieves snapshots for a source
func (s *Service) ListSnapshots(ctx context.Context, tenantID, sourceID string, limit int) ([]*repository.Snapshot, error) {
	snapshots, err := s.repo.ListSnapshots(ctx, tenantID, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	return snapshots, nil
}

// GetSnapshot retrieves a snapshot by ID
func (s *Service) GetSnapshot(ctx context.Context, snapshotID string) (*repository.Snapshot, error) {
	snapshot, err := s.repo.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	return snapshot, nil
}

// EvaluateRetentionPolicy evaluates a retention policy against a list of snapshots
// and returns which snapshots should be kept and which should be deleted
func (s *Service) EvaluateRetentionPolicy(ctx context.Context, tenantID, sourceID string, policy types.RetentionPolicy) (*types.RetentionEvaluationResult, error) {
	// Get all completed snapshots for this source, ordered by created_at ASC
	snapshots, err := s.repo.ListSnapshotsForRetention(ctx, tenantID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for retention: %w", err)
	}

	// If mode is "all", keep all snapshots (no deletion)
	if policy.Mode == "all" {
		var allIDs []string
		for _, snap := range snapshots {
			allIDs = append(allIDs, snap.ID)
		}
		return &types.RetentionEvaluationResult{
			SnapshotsToDelete: nil,
			SnapshotsToKeep:   allIDs,
			Summary:           fmt.Sprintf("Mode 'all': keeping all %d snapshots", len(snapshots)),
		}, nil
	}

	// Build a set of snapshots to protect
	protected := make(map[string]bool) // snapshot ID -> true

	now := time.Now()

	// Rule 1: MinAgeHours - protect snapshots younger than minimum age
	if policy.MinAgeHours != nil {
		minAge := time.Duration(*policy.MinAgeHours) * time.Hour
		for _, snap := range snapshots {
			snapAge := now.Sub(snap.CreatedAt)
			if snapAge < minAge {
				protected[snap.ID] = true
			}
		}
	}

	// Rule 2: MaxAgeDays - mark snapshots older than max age for deletion (overrides protection)
	// Also, when in "within_duration" mode, protect snapshots WITHIN the age window
	var maxAgeTime time.Time
	if policy.MaxAgeDays != nil {
		maxAge := time.Duration(*policy.MaxAgeDays) * 24 * time.Hour
		maxAgeTime = now.Add(-maxAge)

		// If mode is "within_duration", protect all snapshots within the duration window
		// This is the key fix: snapshots younger than maxAgeTime should be kept
		if policy.Mode == "within_duration" {
			for _, snap := range snapshots {
				if !snap.CreatedAt.Before(maxAgeTime) {
					protected[snap.ID] = true
				}
			}
		}
	}

	// Rule 3: KeepLastN - keep the N most recent snapshots
	if policy.KeepLastN != nil {
		count := *policy.KeepLastN
		// Snapshots are ordered ASC, so we take the last N
		startIdx := 0
		if len(snapshots) > count {
			startIdx = len(snapshots) - count
		}
		for i := startIdx; i < len(snapshots); i++ {
			protected[snapshots[i].ID] = true
		}
	}

	// Rule 4: KeepDaily - keep one snapshot per day for N days
	if policy.KeepDaily != nil {
		days := *policy.KeepDaily
		// Group snapshots by day (in UTC)
		dailySnapshots := make(map[string]*repository.Snapshot) // day string -> snapshot
		for _, snap := range snapshots {
			day := snap.CreatedAt.UTC().Format("2006-01-02")
			// Keep the most recent snapshot for each day
			if existing, ok := dailySnapshots[day]; !ok || snap.CreatedAt.After(existing.CreatedAt) {
				dailySnapshots[day] = snap
			}
		}
		// Protect snapshots within the daily retention window
		cutoffDate := now.AddDate(0, 0, -days).UTC().Format("2006-01-02")
		for day, snap := range dailySnapshots {
			if day >= cutoffDate {
				protected[snap.ID] = true
			}
		}
	}

	// Rule 5: KeepWeekly - keep one snapshot per week for N weeks
	if policy.KeepWeekly != nil {
		weeks := *policy.KeepWeekly
		// Group snapshots by ISO week
		weeklySnapshots := make(map[string]*repository.Snapshot) // year-week -> snapshot
		for _, snap := range snapshots {
			year, week := snap.CreatedAt.ISOWeek()
			key := fmt.Sprintf("%d-W%02d", year, week)
			if existing, ok := weeklySnapshots[key]; !ok || snap.CreatedAt.After(existing.CreatedAt) {
				weeklySnapshots[key] = snap
			}
		}
		// Protect snapshots within the weekly retention window
		cutoffWeek := now.AddDate(0, 0, -weeks*7)
		for _, snap := range weeklySnapshots {
			if snap.CreatedAt.After(cutoffWeek) {
				protected[snap.ID] = true
			}
		}
	}

	// Rule 6: KeepMonthly - keep one snapshot per month for N months
	if policy.KeepMonthly != nil {
		months := *policy.KeepMonthly
		// Group snapshots by month
		monthlySnapshots := make(map[string]*repository.Snapshot) // year-month -> snapshot
		for _, snap := range snapshots {
			month := snap.CreatedAt.UTC().Format("2006-01")
			if existing, ok := monthlySnapshots[month]; !ok || snap.CreatedAt.After(existing.CreatedAt) {
				monthlySnapshots[month] = snap
			}
		}
		// Protect snapshots within the monthly retention window
		cutoffMonth := now.AddDate(0, -months, 0)
		for _, snap := range monthlySnapshots {
			if snap.CreatedAt.After(cutoffMonth) {
				protected[snap.ID] = true
			}
		}
	}

	// Build results
	var toDelete, toKeep []string

	for _, snap := range snapshots {
		// Check max age first (overrides protection)
		if policy.MaxAgeDays != nil && snap.CreatedAt.Before(maxAgeTime) {
			toDelete = append(toDelete, snap.ID)
			continue
		}

		if protected[snap.ID] {
			toKeep = append(toKeep, snap.ID)
		} else {
			toDelete = append(toDelete, snap.ID)
		}
	}

	result := &types.RetentionEvaluationResult{
		SnapshotsToDelete: toDelete,
		SnapshotsToKeep:   toKeep,
		Summary: fmt.Sprintf("Evaluated %d snapshots: %d to keep, %d to delete",
			len(snapshots), len(toKeep), len(toDelete)),
	}

	return result, nil
}

// EnqueueDeleteJob creates a delete_snapshot job for a specific snapshot
// The job is targeted to the worker that owns the snapshot
func (s *Service) EnqueueDeleteJob(ctx context.Context, snapshotID string) (*repository.Job, error) {
	// Get snapshot details
	snapshot, err := s.repo.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Verify snapshot has a worker_id (required for local_fs storage)
	if snapshot.WorkerID == nil || *snapshot.WorkerID == "" {
		return nil, fmt.Errorf("snapshot has no worker_id, cannot delete")
	}

	// Build job payload
	payload := types.JobPayload{
		DeleteSnapshotID: &snapshotID,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create job with target worker (higher priority for cleanup)
	priority := 10 // Higher than normal backup jobs
	job, err := s.repo.CreateJobWithTargetWorker(
		ctx,
		snapshot.TenantID,
		types.JobTypeDeleteSnapshot,
		&snapshot.SourceID,
		*snapshot.WorkerID,
		payloadJSON,
		priority,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete job: %w", err)
	}

	// Enqueue to Redis
	jobMsg := map[string]any{
		"job_id":     job.ID,
		"tenant_id":  snapshot.TenantID,
		"type":       "delete_snapshot",
		"priority":   priority,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	jobMsgJSON, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job message: %w", err)
	}

	if err := s.redis.LPush(ctx, JobQueueKey, jobMsgJSON).Err(); err != nil {
		s.LogSystemError(ctx, "Redis: failed to enqueue delete snapshot job", err, map[string]any{
			"job_id":      job.ID,
			"tenant_id":   snapshot.TenantID,
			"snapshot_id": snapshotID,
		})
		return nil, fmt.Errorf("failed to enqueue delete job: %w", err)
	}

	return job, nil
}

// RunRetentionEvaluationForSource evaluates retention policy for a specific source
// and enqueues delete jobs for snapshots that should be removed
func (s *Service) RunRetentionEvaluationForSource(ctx context.Context, sourceID string) (*types.RetentionEvaluationResult, error) {
	// Get schedule for this source
	schedule, err := s.repo.GetScheduleForSource(ctx, sourceID)
	if err != nil {
		// No schedule means no retention policy - skip gracefully
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("retention: skipping source %s (no schedule configured)", sourceID)
			return &types.RetentionEvaluationResult{
				SnapshotsToKeep:   []string{},
				SnapshotsToDelete: []string{},
				Summary:           "no schedule configured, retention skipped",
			}, nil
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Parse retention policy
	var policy types.RetentionPolicy
	if len(schedule.RetentionPolicy) > 0 {
		if err := json.Unmarshal(schedule.RetentionPolicy, &policy); err != nil {
			return nil, fmt.Errorf("failed to parse retention policy: %w", err)
		}
	} else {
		// Use default policy if none is set
		policy = types.DefaultRetentionPolicy()
	}

	log.Printf("retention: source=%s, raw_policy={mode:%s, keep_last_n:%v, keep_within_duration:%s, max_age_days:%v}",
		sourceID, policy.Mode,
		policy.KeepLastN,
		policy.KeepWithinDuration,
		policy.MaxAgeDays)

	// Normalize policy to convert frontend format (mode, keep_within_duration) to internal format
	policy.Normalize()

	log.Printf("retention: source=%s, normalized_policy={mode:%s, keep_last_n:%v, max_age_days:%v}",
		sourceID, policy.Mode,
		policy.KeepLastN,
		policy.MaxAgeDays)

	// Evaluate policy
	result, err := s.EvaluateRetentionPolicy(ctx, schedule.TenantID, sourceID, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate retention policy: %w", err)
	}

	log.Printf("retention: source=%s, result={to_keep:%d, to_delete:%d, summary:%s}",
		sourceID, len(result.SnapshotsToKeep), len(result.SnapshotsToDelete), result.Summary)

	// Enqueue delete jobs for snapshots to delete
	var deleteJobsEnqueued int
	for _, snapshotID := range result.SnapshotsToDelete {
		_, err := s.EnqueueDeleteJob(ctx, snapshotID)
		if err != nil {
			log.Printf("retention: failed to enqueue delete job for snapshot %s: %v", snapshotID, err)
		} else {
			deleteJobsEnqueued++
		}
	}

	if deleteJobsEnqueued > 0 {
		log.Printf("retention: source=%s, enqueued %d delete jobs", sourceID, deleteJobsEnqueued)
	}

	return result, nil
}

// RunRetentionEvaluationForAllSources evaluates retention policies for all enabled schedules
// and enqueues delete jobs for snapshots that should be removed
func (s *Service) RunRetentionEvaluationForAllSources(ctx context.Context) ([]*SourceRetentionResult, error) {
	// Get all enabled schedules
	schedules, err := s.repo.ListAllSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	var results []*SourceRetentionResult

	for _, schedule := range schedules {
		result, err := s.RunRetentionEvaluationForSource(ctx, schedule.SourceID)
		if err != nil {
			results = append(results, &SourceRetentionResult{
				SourceID: schedule.SourceID,
				Error:    err.Error(),
			})
			continue
		}

		results = append(results, &SourceRetentionResult{
			SourceID:         schedule.SourceID,
			EvaluationResult: result,
			JobsEnqueued:     len(result.SnapshotsToDelete),
		})
	}

	return results, nil
}

// SourceRetentionResult represents the result of running retention evaluation for a source
type SourceRetentionResult struct {
	SourceID         string                           `json:"source_id"`
	EvaluationResult *types.RetentionEvaluationResult `json:"evaluation_result,omitempty"`
	JobsEnqueued     int                              `json:"jobs_enqueued"`
	Error            string                           `json:"error,omitempty"`
}

const (
	// RetentionLockKey is the Redis key prefix for retention locks
	RetentionLockKey = "xvault:retention:lock:"
	// RetentionCooldown is the minimum time between retention runs for the same source
	// This prevents retention from running too frequently for high-frequency backups
	RetentionCooldown = 60 * time.Second
)

// maybeTriggerRetentionForSource conditionally runs retention evaluation for a source
// using Redis-based debouncing to prevent concurrent runs for the same source.
// This is called after each backup completes in a goroutine.
func (s *Service) maybeTriggerRetentionForSource(ctx context.Context, sourceID string) {
	// Try to acquire a lock using Redis SETNX (set if not exists)
	lockKey := RetentionLockKey + sourceID
	acquired, err := s.redis.SetNX(ctx, lockKey, "1", RetentionCooldown).Result()
	if err != nil {
		s.LogSystemError(ctx, "Redis: failed to acquire retention lock", err, map[string]any{
			"source_id": sourceID,
		})
		log.Printf("retention: failed to acquire lock for source %s: %v", sourceID, err)
		return
	}

	// If lock was not acquired, another goroutine is already handling retention for this source
	if !acquired {
		log.Printf("retention: skipping for source %s (already in progress)", sourceID)
		return
	}

	log.Printf("retention: triggered for source %s", sourceID)

	// Run retention evaluation with a timeout
	evalCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := s.RunRetentionEvaluationForSource(evalCtx, sourceID)
	if err != nil {
		s.LogSystemError(ctx, "Retention: evaluation failed", err, map[string]any{
			"source_id": sourceID,
		})
		log.Printf("retention: failed for source %s: %v", sourceID, err)
		return
	}

	// Log results
	if result != nil {
		log.Printf("retention: source=%s, to_keep=%d, to_delete=%d, summary=%s",
			sourceID, len(result.SnapshotsToKeep), len(result.SnapshotsToDelete), result.Summary)
	}
}

// Schedule management

// CreateScheduleRequest is the request to create a schedule
type CreateScheduleRequest struct {
	TenantID        string                `json:"tenant_id"`
	SourceID        string                `json:"source_id"`
	Cron            *string               `json:"cron,omitempty"`
	IntervalMinutes *int                  `json:"interval_minutes,omitempty"`
	Timezone        string                `json:"timezone,omitempty"` // Default "UTC"
	RetentionPolicy types.RetentionPolicy `json:"retention_policy"`
}

// CreateSchedule creates a new schedule for a source
func (s *Service) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (*repository.Schedule, error) {
	// Validate source exists and belongs to tenant
	source, err := s.repo.GetSource(ctx, req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	if source.TenantID != req.TenantID {
		return nil, fmt.Errorf("source does not belong to tenant")
	}

	// Validate schedule parameters
	if req.Cron == nil && req.IntervalMinutes == nil {
		return nil, fmt.Errorf("either cron or interval_minutes must be specified")
	}

	if req.Cron != nil && req.IntervalMinutes != nil {
		return nil, fmt.Errorf("cannot specify both cron and interval_minutes")
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Marshal retention policy to JSONB
	retentionPolicyJSON, err := json.Marshal(req.RetentionPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal retention policy: %w", err)
	}

	// Create schedule
	schedule, err := s.repo.CreateSchedule(ctx, req.TenantID, req.SourceID, req.Cron, req.IntervalMinutes, timezone, retentionPolicyJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	// Calculate and set initial next_run_at
	now := time.Now()
	nextRun, calcErr := s.CalculateNextRun(schedule, now)
	if calcErr == nil {
		_ = s.repo.UpdateScheduleRunTimes(ctx, schedule.ID, now, nextRun)
		schedule.NextRunAt = &nextRun
	}

	return schedule, nil
}

// UpdateScheduleRequest is the request to update a schedule
type UpdateScheduleRequest struct {
	Cron            *string                `json:"cron,omitempty"`
	IntervalMinutes *int                   `json:"interval_minutes,omitempty"`
	Timezone        *string                `json:"timezone,omitempty"`
	Status          *string                `json:"status,omitempty"` // "enabled" or "disabled"
	RetentionPolicy *types.RetentionPolicy `json:"retention_policy,omitempty"`
}

// UpdateSchedule updates an existing schedule
func (s *Service) UpdateSchedule(ctx context.Context, scheduleID string, req UpdateScheduleRequest) (*repository.Schedule, error) {
	// Get existing schedule to verify it exists
	existing, err := s.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	// Prepare values for update (use existing values if not provided)
	cron := req.Cron
	if cron == nil {
		cron = existing.Cron
	}

	intervalMinutes := req.IntervalMinutes
	if intervalMinutes == nil {
		intervalMinutes = existing.IntervalMinutes
	}

	timezone := existing.Timezone
	timezoneChanged := false
	if req.Timezone != nil && *req.Timezone != "" && *req.Timezone != existing.Timezone {
		timezone = *req.Timezone
		timezoneChanged = true
	}

	// Check if cron changed
	cronChanged := false
	if req.Cron != nil && (existing.Cron == nil || *req.Cron != *existing.Cron) {
		cronChanged = true
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	// Marshal retention policy
	var retentionPolicyJSON json.RawMessage
	if req.RetentionPolicy != nil {
		retentionPolicyJSON, err = json.Marshal(req.RetentionPolicy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal retention policy: %w", err)
		}
	} else {
		retentionPolicyJSON = existing.RetentionPolicy
	}

	// Update schedule
	schedule, err := s.repo.UpdateSchedule(ctx, scheduleID, cron, intervalMinutes, timezone, status, retentionPolicyJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	// Recalculate next_run_at if cron or timezone changed
	if (cronChanged || timezoneChanged) && status == "enabled" {
		now := time.Now()
		nextRun, calcErr := s.CalculateNextRun(schedule, now)
		if calcErr == nil {
			// Use existing last_run_at or use a zero time
			lastRunAt := now
			if schedule.LastRunAt != nil {
				lastRunAt = *schedule.LastRunAt
			}
			_ = s.repo.UpdateScheduleRunTimes(ctx, scheduleID, lastRunAt, nextRun)
			schedule.NextRunAt = &nextRun
		}
	}

	return schedule, nil
}

// GetSchedule retrieves a schedule by ID
func (s *Service) GetSchedule(ctx context.Context, scheduleID string) (*repository.Schedule, error) {
	schedule, err := s.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return schedule, nil
}

// GetScheduleForSource retrieves the schedule for a specific source
func (s *Service) GetScheduleForSource(ctx context.Context, sourceID string) (*repository.Schedule, error) {
	schedule, err := s.repo.GetScheduleForSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule for source: %w", err)
	}

	return schedule, nil
}

// ListSchedules retrieves all schedules for a tenant
func (s *Service) ListSchedules(ctx context.Context, tenantID string) ([]*repository.Schedule, error) {
	schedules, err := s.repo.ListSchedulesByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	return schedules, nil
}

// UpdateSourceRetentionPolicyRequest is the request to update only the retention policy
type UpdateSourceRetentionPolicyRequest struct {
	RetentionPolicy types.RetentionPolicy `json:"retention_policy"`
}

// UpdateSourceRetentionPolicy updates the retention policy for a source's schedule
func (s *Service) UpdateSourceRetentionPolicy(ctx context.Context, sourceID string, req UpdateSourceRetentionPolicyRequest) (*repository.Schedule, error) {
	// Get existing schedule for the source
	schedule, err := s.repo.GetScheduleForSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found for source: %w", err)
	}

	// Marshal new retention policy
	retentionPolicyJSON, err := json.Marshal(req.RetentionPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal retention policy: %w", err)
	}

	// Update only the retention policy
	updated, err := s.repo.UpdateScheduleRetention(ctx, schedule.ID, retentionPolicyJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update retention policy: %w", err)
	}

	return updated, nil
}

// ========== Restore Service Methods ==========

// RestoreJobClaimRequest is the request to claim a restore job
type RestoreJobClaimRequest struct {
	ServiceID string `json:"service_id"`
}

// RestoreJobClaimResponse is the response when claiming a restore job
type RestoreJobClaimResponse struct {
	JobID      string `json:"job_id"`
	TenantID   string `json:"tenant_id"`
	SourceID   string `json:"source_id"`
	SnapshotID string `json:"snapshot_id"`
	LocalPath  string `json:"local_path"` // Actual path to snapshot on worker storage
}

// RestoreJobCompleteRequest is the request to complete a restore job
type RestoreJobCompleteRequest struct {
	ServiceID     string `json:"service_id"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	DownloadURL   string `json:"download_url,omitempty"`
	DownloadToken string `json:"download_token,omitempty"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
}

// EnqueueRestoreJob creates and enqueues a restore job
func (s *Service) EnqueueRestoreJob(ctx context.Context, tenantID, snapshotID string) (*repository.Job, error) {
	// Get snapshot to verify it exists and get source info
	snapshot, err := s.repo.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Verify snapshot belongs to tenant
	if snapshot.TenantID != tenantID {
		return nil, fmt.Errorf("snapshot does not belong to tenant")
	}

	// Check if snapshot is completed
	if snapshot.Status != "completed" {
		return nil, fmt.Errorf("snapshot is not completed (status: %s)", snapshot.Status)
	}

	// Create restore job payload
	payload := types.JobPayload{
		SourceID:          snapshot.SourceID,
		RestoreSnapshotID: &snapshotID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create job record (sourceID as string pointer for restore jobs)
	sourceIDPtr := &snapshot.SourceID
	job, err := s.repo.CreateJob(ctx, tenantID, types.JobTypeRestore, sourceIDPtr, payloadJSON, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Enqueue to Redis
	jobData := map[string]interface{}{
		"job_id":      job.ID,
		"tenant_id":   tenantID,
		"source_id":   snapshot.SourceID,
		"snapshot_id": snapshotID,
		"type":        "restore",
	}

	if err := s.redis.LPush(ctx, JobQueueKey, jobData).Err(); err != nil {
		s.LogSystemError(ctx, "Redis: failed to enqueue restore job", err, map[string]any{
			"job_id":      job.ID,
			"tenant_id":   tenantID,
			"snapshot_id": snapshotID,
		})
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("enqueued restore job %s for snapshot %s", job.ID, snapshotID)
	return job, nil
}

// ClaimRestoreJob claims the next available restore job for a restore service
func (s *Service) ClaimRestoreJob(ctx context.Context, req RestoreJobClaimRequest) (*RestoreJobClaimResponse, error) {
	// Find next queued restore job
	job, err := s.repo.ClaimRestoreJob(ctx, req.ServiceID)
	if err != nil {
		return nil, err
	}

	// Parse payload to get snapshot ID
	var payload types.JobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse job payload: %w", err)
	}

	if payload.RestoreSnapshotID == nil {
		return nil, fmt.Errorf("restore_snapshot_id not found in payload")
	}

	// Handle source_id pointer
	var sourceID string
	if job.SourceID != nil {
		sourceID = *job.SourceID
	}

	// Fetch snapshot record to get the actual local_path
	snapshot, err := s.repo.GetSnapshot(ctx, *payload.RestoreSnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot record: %w", err)
	}

	// Build local_path from snapshot record
	localPath := ""
	if snapshot.LocalPath != nil {
		localPath = *snapshot.LocalPath
	} else {
		// Fallback: construct path from storage backend and locator
		localPath = filepath.Join("/var/lib/xvault/backups", "tenants", job.TenantID, "sources", sourceID, "snapshots", *payload.RestoreSnapshotID)
	}

	return &RestoreJobClaimResponse{
		JobID:      job.ID,
		TenantID:   job.TenantID,
		SourceID:   sourceID,
		SnapshotID: *payload.RestoreSnapshotID,
		LocalPath:  localPath,
	}, nil
}

// CompleteRestoreJob handles restore service job completion
func (s *Service) CompleteRestoreJob(ctx context.Context, jobID string, req RestoreJobCompleteRequest) error {
	// Update job status
	var finalStatus types.JobStatus
	switch req.Status {
	case "completed":
		finalStatus = types.JobStatusCompleted
	case "failed":
		finalStatus = types.JobStatusFailed
	default:
		return fmt.Errorf("invalid job status: %s", req.Status)
	}

	var errorMsg *string
	if req.Error != "" {
		errorMsg = &req.Error
	}

	if err := s.repo.CompleteJob(ctx, jobID, finalStatus, errorMsg); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// If successful, store download metadata in database
	if req.Status == "completed" && req.DownloadToken != "" {
		// Get job details to find snapshot ID
		job, err := s.repo.GetJob(ctx, jobID)
		if err != nil {
			return fmt.Errorf("failed to get job details: %w", err)
		}

		// Parse payload to get snapshot ID
		var payload types.JobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse job payload: %w", err)
		}

		if payload.RestoreSnapshotID != nil {
			// Parse expires_at
			expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
			if err != nil {
				log.Printf("failed to parse expires_at: %v", err)
				expiresAt = time.Now().Add(1 * time.Hour) // Default to 1 hour
			}

			// Update snapshot with download info
			if err := s.UpdateSnapshotDownloadInfo(ctx, *payload.RestoreSnapshotID, req.DownloadToken, req.DownloadURL, expiresAt); err != nil {
				log.Printf("failed to update snapshot download info: %v", err)
				// Don't fail the job completion if we can't store download metadata
			}
		}

		log.Printf("restore job %s completed: download_url=%s, expires_at=%s", jobID, req.DownloadURL, req.ExpiresAt)
	}

	return nil
}

// RegisterServiceRequest is the request to register a restore service
type RegisterServiceRequest struct {
	ServiceID    string         `json:"service_id"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Capabilities map[string]any `json:"capabilities"`
}

// RestoreService represents a restore service registration
type RestoreService struct {
	ID           string         `json:"id"`
	ServiceID    string         `json:"service_id"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Capabilities map[string]any `json:"capabilities"`
	Status       string         `json:"status"`
	LastSeenAt   *time.Time     `json:"last_seen_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// RegisterRestoreService registers a restore service with the hub
func (s *Service) RegisterRestoreService(ctx context.Context, req RegisterServiceRequest) (*RestoreService, error) {
	// For v0, just return a simple response (stored in memory, not DB)
	// In production, this would be stored in a restore_services table
	now := time.Now()
	return &RestoreService{
		ID:           req.ServiceID,
		ServiceID:    req.ServiceID,
		Type:         req.Type,
		Name:         req.Name,
		Capabilities: req.Capabilities,
		Status:       "online",
		LastSeenAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ServiceHeartbeatRequest is the request for service heartbeat
type ServiceHeartbeatRequest struct {
	ServiceID string `json:"service_id"`
	Status    string `json:"status"`
}

// RestoreServiceHeartbeat updates a restore service's heartbeat
func (s *Service) RestoreServiceHeartbeat(ctx context.Context, req ServiceHeartbeatRequest) error {
	// For v0, just log (stored in memory, not DB)
	// In production, this would update the restore_services table
	log.Printf("restore service %s heartbeat: status=%s", req.ServiceID, req.Status)
	return nil
}

// ========== System Settings Management ==========

// GetSetting retrieves a system setting by key
func (s *Service) GetSetting(ctx context.Context, key string) (*repository.SystemSetting, error) {
	setting, err := s.repo.GetSetting(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}
	return setting, nil
}

// ListSettings retrieves all system settings
func (s *Service) ListSettings(ctx context.Context) ([]*repository.SystemSetting, error) {
	settings, err := s.repo.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	return settings, nil
}

// UpdateSettingRequest is the request to update a system setting
type UpdateSettingRequest struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// UpdateSetting updates a system setting
func (s *Service) UpdateSetting(ctx context.Context, key string, req UpdateSettingRequest) (*repository.SystemSetting, error) {
	setting, err := s.repo.UpsertSetting(ctx, key, req.Value, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to update setting: %w", err)
	}
	return setting, nil
}

// GetDownloadExpirationHours retrieves the download expiration setting in hours
func (s *Service) GetDownloadExpirationHours(ctx context.Context) (int, error) {
	setting, err := s.repo.GetSetting(ctx, "download_expiration_hours")
	if err != nil {
		// Return default if setting not found
		return 1, nil
	}

	var hours int
	if _, err := fmt.Sscanf(setting.Value, "%d", &hours); err != nil {
		return 1, nil // Default to 1 hour if parsing fails
	}

	return hours, nil
}

// UpdateSnapshotDownloadInfo updates download tracking info for a snapshot
func (s *Service) UpdateSnapshotDownloadInfo(ctx context.Context, snapshotID, downloadToken, downloadURL string, expiresAt time.Time) error {
	return s.repo.UpdateSnapshotDownloadInfo(ctx, snapshotID, downloadToken, downloadURL, expiresAt)
}

// ========== Admin User Management ==========

// CreateUserAdminRequest is the request to create a user as admin
type CreateUserAdminRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // "owner" | "admin" | "member"
}

// UpdateUserAdminRequest is the request to update a user as admin
type UpdateUserAdminRequest struct {
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// ListUsers returns all users (admin only)
func (s *Service) ListUsers(ctx context.Context) ([]repository.User, error) {
	return s.repo.ListUsers(ctx)
}

// GetUser returns a specific user by ID (admin only)
func (s *Service) GetUser(ctx context.Context, userID string) (*repository.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// CreateUserAdmin creates a new user with tenant as admin
func (s *Service) CreateUserAdmin(ctx context.Context, req CreateUserAdminRequest) (*repository.User, error) {
	// Validate role
	if req.Role != "owner" && req.Role != "admin" && req.Role != "member" {
		return nil, fmt.Errorf("invalid role: must be owner, admin, or member")
	}

	// Generate tenant name from email if not provided
	tenantName := req.Name
	if tenantName == "" {
		// Extract username from email (part before @)
		emailParts := strings.Split(req.Email, "@")
		if len(emailParts) > 0 && emailParts[0] != "" {
			// Capitalize first letter and add "'s Workspace"
			username := emailParts[0]
			if len(username) > 0 {
				tenantName = strings.ToUpper(string(username[0])) + username[1:] + "'s Workspace"
			} else {
				tenantName = "New Workspace"
			}
		} else {
			tenantName = "New Workspace"
		}
	}

	// Create tenant first
	tenant, err := s.repo.CreateTenant(ctx, tenantName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Generate encryption keypair for tenant
	publicKey, privateKey, err := crypto.GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption keys: %w", err)
	}

	// Encrypt private key with platform KEK
	encryptedPrivateKey, err := crypto.EncryptForStorage([]byte(privateKey), s.encryptionKEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Store tenant key
	_, err = s.repo.CreateTenantKey(ctx, tenant.ID, "age-x25519", publicKey, encryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to store tenant key: %w", err)
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.repo.CreateUser(ctx, tenant.ID, req.Email, hashedPassword, req.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateUserAdmin updates a user (admin only)
func (s *Service) UpdateUserAdmin(ctx context.Context, userID string, req UpdateUserAdminRequest) (*repository.User, error) {
	// Get existing user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Email != "" && req.Email != user.Email {
		if err := s.repo.UpdateUserEmail(ctx, userID, req.Email); err != nil {
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
	}

	if req.Role != "" && req.Role != user.Role {
		if req.Role != "owner" && req.Role != "admin" && req.Role != "member" {
			return nil, fmt.Errorf("invalid role: must be owner, admin, or member")
		}
		if err := s.repo.UpdateUserRole(ctx, userID, req.Role); err != nil {
			return nil, fmt.Errorf("failed to update role: %w", err)
		}
	}

	// Fetch updated user
	return s.repo.GetUserByID(ctx, userID)
}

// DeleteUser deletes a user (admin only)
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.DeleteUser(ctx, userID)
}

// ========== Admin Tenant Management ==========

// ListTenants returns all tenants (admin only)
func (s *Service) ListTenants(ctx context.Context) ([]repository.Tenant, error) {
	return s.repo.ListTenants(ctx)
}

// GetTenant returns a specific tenant by ID (admin only)
func (s *Service) GetTenant(ctx context.Context, tenantID string) (*repository.Tenant, error) {
	return s.repo.GetTenantByID(ctx, tenantID)
}

// DeleteTenant deletes a tenant and all associated data (admin only)
// It first enqueues delete_snapshot jobs to clean up worker storage, then deletes the tenant
func (s *Service) DeleteTenant(ctx context.Context, tenantID string) error {
	// First, get all snapshots for this tenant so we can enqueue cleanup jobs
	snapshots, err := s.repo.ListSnapshotsByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list snapshots for cleanup: %w", err)
	}

	// Enqueue delete_snapshot jobs for all snapshots (to clean up worker storage)
	// Use existing EnqueueDeleteJob method
	var enqueueErrors []error
	for _, snapshot := range snapshots {
		if snapshot.WorkerID != nil && *snapshot.WorkerID != "" {
			// Enqueue delete job using existing method
			_, err := s.EnqueueDeleteJob(ctx, snapshot.ID)
			if err != nil {
				// Log but continue with other snapshots
				log.Printf("warning: failed to enqueue delete job for snapshot %s: %v", snapshot.ID, err)
				enqueueErrors = append(enqueueErrors, err)
			}
		}
	}

	// Finally, delete the tenant (database cascade handles the rest)
	if err := s.repo.DeleteTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	log.Printf("deleted tenant %s and enqueued %d snapshot cleanup jobs (%d failed)", tenantID, len(snapshots), len(enqueueErrors))
	return nil
}

// ========== Admin Source Management ==========

// ListAllSources returns all sources across all tenants (admin only)
func (s *Service) ListAllSources(ctx context.Context) ([]*repository.Source, error) {
	return s.repo.ListAllSources(ctx)
}

// CreateSourceAdminRequest is the request to create a source as admin
type CreateSourceAdminRequest struct {
	TenantID   string          `json:"tenant_id"`
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Config     json.RawMessage `json:"config"`
	Credential string          `json:"credential"` // Base64-encoded credential (password or private key)
}

// CreateSourceAdmin creates a source with credential for a tenant (admin only)
func (s *Service) CreateSourceAdmin(ctx context.Context, req CreateSourceAdminRequest) (*repository.Source, error) {
	// Validate source type
	validTypes := map[string]bool{"ssh": true, "sftp": true, "ftp": true, "mysql": true, "postgresql": true}
	if !validTypes[req.Type] {
		return nil, fmt.Errorf("invalid source type: must be ssh, sftp, ftp, mysql, or postgresql")
	}

	// Verify tenant exists
	_, err := s.repo.GetTenant(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	// Create credential for the source
	cred, err := s.CreateCredential(ctx, CreateCredentialRequest{
		TenantID:  req.TenantID,
		Kind:      "source",
		Plaintext: req.Credential, // Already base64-encoded from frontend
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// Create source with the new credential
	source, err := s.repo.CreateSource(ctx, req.TenantID, req.Type, req.Name, cred.ID, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return source, nil
}

// UpdateSourceAdminRequest is the request to update a source as admin
type UpdateSourceAdminRequest struct {
	Name       string          `json:"name,omitempty"`
	Status     string          `json:"status,omitempty"` // "active" or "disabled"
	Config     json.RawMessage `json:"config,omitempty"`
	Credential string          `json:"credential,omitempty"` // Base64-encoded new credential (if rotating)
}

// UpdateSourceAdmin updates a source (admin only)
func (s *Service) UpdateSourceAdmin(ctx context.Context, sourceID string, req UpdateSourceAdminRequest) (*repository.Source, error) {
	// Get existing source
	source, err := s.repo.GetSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	// Update name/status if provided
	name := source.Name
	status := source.Status
	if req.Name != "" {
		name = req.Name
	}
	if req.Status != "" {
		if req.Status != "active" && req.Status != "disabled" {
			return nil, fmt.Errorf("invalid status: must be active or disabled")
		}
		status = req.Status
	}

	// Update the source
	source, err = s.repo.UpdateSource(ctx, sourceID, name, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update source: %w", err)
	}

	// Update config if provided
	if len(req.Config) > 0 {
		if err := s.repo.UpdateSourceConfig(ctx, sourceID, req.Config); err != nil {
			return nil, fmt.Errorf("failed to update source config: %w", err)
		}
	}

	// Rotate credential if provided
	if req.Credential != "" {
		// Create new credential
		cred, err := s.CreateCredential(ctx, CreateCredentialRequest{
			TenantID:  source.TenantID,
			Kind:      "source",
			Plaintext: req.Credential,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create new credential: %w", err)
		}

		// Update source to use new credential
		if err := s.repo.UpdateSourceCredential(ctx, sourceID, cred.ID); err != nil {
			return nil, fmt.Errorf("failed to update source credential: %w", err)
		}
	}

	// Fetch updated source
	return s.repo.GetSource(ctx, sourceID)
}

// DeleteSource deletes a source (admin only)
func (s *Service) DeleteSource(ctx context.Context, sourceID string) error {
	return s.repo.DeleteSource(ctx, sourceID)
}

// ========== Connection Testing ==========

// TestConnectionRequest is the request to test a source connection
type TestConnectionRequest struct {
	Type          string `json:"type"`               // ssh, sftp, ftp, mysql, postgresql
	Host          string `json:"host"`               // Hostname or IP
	Port          int    `json:"port"`               // Port number
	Username      string `json:"username"`           // Username for connection
	Credential    string `json:"credential"`         // Base64-encoded password or private key
	UsePrivateKey bool   `json:"use_private_key"`    // True if credential is a private key
	Database      string `json:"database,omitempty"` // Database name (for mysql/postgresql)
}

// TestConnectionResult is the result of a connection test
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TestConnection tests connectivity to a source
func (s *Service) TestConnection(ctx context.Context, req TestConnectionRequest) (*TestConnectionResult, error) {
	// Decode credential from base64
	credBytes, err := base64.StdEncoding.DecodeString(req.Credential)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Invalid credential encoding",
			Details: err.Error(),
		}, nil
	}
	credential := string(credBytes)

	switch req.Type {
	case "ssh", "sftp":
		return s.testSSHConnection(ctx, req, credential)
	case "ftp":
		return s.testFTPConnection(ctx, req, credential)
	case "mysql":
		return s.testMySQLConnection(ctx, req, credential)
	case "postgresql":
		return s.testPostgreSQLConnection(ctx, req, credential)
	default:
		return &TestConnectionResult{
			Success: false,
			Message: "Unsupported source type",
			Details: fmt.Sprintf("Type '%s' is not supported", req.Type),
		}, nil
	}
}

// testSSHConnection tests SSH/SFTP connectivity
func (s *Service) testSSHConnection(ctx context.Context, req TestConnectionRequest, credential string) (*TestConnectionResult, error) {
	// Build SSH config
	sshConfig := &ssh.ClientConfig{
		User:            req.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Make configurable
		Timeout:         10 * time.Second,
	}

	// Add authentication method
	if req.UsePrivateKey {
		signer, err := ssh.ParsePrivateKey([]byte(credential))
		if err != nil {
			return &TestConnectionResult{
				Success: false,
				Message: "Failed to parse private key",
				Details: err.Error(),
			}, nil
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	} else {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(credential))
	}

	// Attempt connection with context timeout
	address := fmt.Sprintf("%s:%d", req.Host, req.Port)

	// Use a dialer with timeout
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Failed to connect to host",
			Details: err.Error(),
		}, nil
	}
	defer conn.Close()

	// Create SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "SSH authentication failed",
			Details: err.Error(),
		}, nil
	}

	sshClient := ssh.NewClient(sshConn, chans, reqs)
	defer sshClient.Close()

	// For SFTP, also test SFTP subsystem
	if req.Type == "sftp" {
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return &TestConnectionResult{
				Success: false,
				Message: "SSH connected but SFTP subsystem failed",
				Details: err.Error(),
			}, nil
		}
		sftpClient.Close()

		return &TestConnectionResult{
			Success: true,
			Message: "SFTP connection successful",
			Details: fmt.Sprintf("Connected to %s as %s", address, req.Username),
		}, nil
	}

	return &TestConnectionResult{
		Success: true,
		Message: "SSH connection successful",
		Details: fmt.Sprintf("Connected to %s as %s", address, req.Username),
	}, nil
}

// testFTPConnection tests FTP connectivity
func (s *Service) testFTPConnection(ctx context.Context, req TestConnectionRequest, credential string) (*TestConnectionResult, error) {
	// Simple TCP connection test for FTP
	address := fmt.Sprintf("%s:%d", req.Host, req.Port)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Failed to connect to FTP server",
			Details: err.Error(),
		}, nil
	}
	conn.Close()

	// Note: Full FTP authentication test would require an FTP library
	// For now, we just verify the port is reachable
	return &TestConnectionResult{
		Success: true,
		Message: "FTP port is reachable",
		Details: fmt.Sprintf("Connected to %s (authentication not fully tested - install FTP client library for full test)", address),
	}, nil
}

// testMySQLConnection tests MySQL connectivity
func (s *Service) testMySQLConnection(ctx context.Context, req TestConnectionRequest, credential string) (*TestConnectionResult, error) {
	// Simple TCP connection test for MySQL
	address := fmt.Sprintf("%s:%d", req.Host, req.Port)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Failed to connect to MySQL server",
			Details: err.Error(),
		}, nil
	}
	conn.Close()

	// Note: Full MySQL authentication test would require the MySQL driver
	// For now, we verify the port is reachable
	return &TestConnectionResult{
		Success: true,
		Message: "MySQL port is reachable",
		Details: fmt.Sprintf("Connected to %s (authentication not fully tested - requires MySQL driver)", address),
	}, nil
}

// testPostgreSQLConnection tests PostgreSQL connectivity
func (s *Service) testPostgreSQLConnection(ctx context.Context, req TestConnectionRequest, credential string) (*TestConnectionResult, error) {
	// We already have lib/pq, so we can do a full connection test
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
		req.Host, req.Port, req.Username, credential, req.Database)

	if req.Database == "" {
		connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable connect_timeout=10",
			req.Host, req.Port, req.Username, credential)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Failed to create PostgreSQL connection",
			Details: err.Error(),
		}, nil
	}
	defer db.Close()

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "PostgreSQL connection failed",
			Details: err.Error(),
		}, nil
	}

	return &TestConnectionResult{
		Success: true,
		Message: "PostgreSQL connection successful",
		Details: fmt.Sprintf("Connected to %s:%d as %s", req.Host, req.Port, req.Username),
	}, nil
}

// ========== Admin Backup Trigger ==========

// TriggerBackupAdmin triggers a manual backup for a source (admin only)
func (s *Service) TriggerBackupAdmin(ctx context.Context, sourceID string) (*repository.Job, error) {
	// Get source to verify it exists
	source, err := s.repo.GetSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	// Build job payload
	payload := types.JobPayload{
		SourceID:     sourceID,
		CredentialID: source.CredentialID,
		SourceConfig: source.Config,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create job record with high priority (manual trigger)
	priority := 10

	job, err := s.repo.CreateJob(ctx, source.TenantID, types.JobTypeBackup, &sourceID, payloadJSON, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Enqueue to Redis
	jobMsg := map[string]any{
		"job_id":     job.ID,
		"tenant_id":  source.TenantID,
		"type":       "backup",
		"priority":   priority,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}

	jobMsgJSON, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job message: %w", err)
	}

	if err := s.redis.LPush(ctx, JobQueueKey, jobMsgJSON).Err(); err != nil {
		s.LogSystemError(ctx, "Redis: failed to enqueue admin-triggered backup job", err, map[string]any{
			"job_id":    job.ID,
			"source_id": sourceID,
		})
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("admin triggered backup for source %s, job_id=%s", sourceID, job.ID)
	return job, nil
}

// ========== Admin Schedule Management ==========

// ListAllSchedulesAdmin returns all schedules across all tenants (admin only)
func (s *Service) ListAllSchedulesAdmin(ctx context.Context) ([]*repository.Schedule, error) {
	return s.repo.ListAllSchedulesAdmin(ctx)
}

// CreateScheduleAdmin creates a schedule for any source (admin only)
func (s *Service) CreateScheduleAdmin(ctx context.Context, req CreateScheduleRequest) (*repository.Schedule, error) {
	// Validate source exists
	source, err := s.repo.GetSource(ctx, req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	// Use source's tenant_id
	req.TenantID = source.TenantID

	// Delegate to existing CreateSchedule
	return s.CreateSchedule(ctx, req)
}

// DeleteSchedule deletes a schedule (admin only)
func (s *Service) DeleteSchedule(ctx context.Context, scheduleID string) error {
	return s.repo.DeleteSchedule(ctx, scheduleID)
}

// ========== Admin Snapshot Management ==========

// ListAllSnapshotsAdmin returns all snapshots across all tenants with source/tenant info (admin only)
func (s *Service) ListAllSnapshotsAdmin(ctx context.Context, limit int) ([]*repository.AdminSnapshot, error) {
	return s.repo.ListAllSnapshotsAdmin(ctx, limit)
}

// ListAllSnapshotsAndJobsAdmin returns all snapshots and in-progress/failed jobs (admin only)
// This shows the complete status including queued, running, and failed backups
func (s *Service) ListAllSnapshotsAndJobsAdmin(ctx context.Context, limit int) ([]*repository.AdminSnapshot, error) {
	return s.repo.ListAllSnapshotsAndJobsAdmin(ctx, limit)
}

// ========== Backup Scheduler ==========

// ProcessDueSchedules checks for schedules that are due to run and enqueues backup jobs
func (s *Service) ProcessDueSchedules(ctx context.Context) (int, error) {
	now := time.Now()

	// Get all schedules that are due
	schedules, err := s.repo.GetDueSchedules(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("failed to get due schedules: %w", err)
	}

	jobsCreated := 0
	for _, schedule := range schedules {
		// Skip if source is not active
		source, err := s.repo.GetSource(ctx, schedule.SourceID)
		if err != nil {
			log.Printf("scheduler: failed to get source %s for schedule %s: %v", schedule.SourceID, schedule.ID, err)
			continue
		}
		if source.Status != "active" {
			log.Printf("scheduler: skipping schedule %s - source %s is not active", schedule.ID, schedule.SourceID)
			continue
		}

		// Enqueue backup job
		job, err := s.EnqueueScheduledBackup(ctx, schedule)
		if err != nil {
			log.Printf("scheduler: failed to enqueue backup for schedule %s: %v", schedule.ID, err)
			continue
		}

		log.Printf("scheduler: enqueued backup job %s for schedule %s (source: %s)", job.ID, schedule.ID, schedule.SourceID)
		jobsCreated++

		// Calculate next run time
		nextRun, err := s.CalculateNextRun(schedule, now)
		if err != nil {
			log.Printf("scheduler: failed to calculate next run for schedule %s: %v", schedule.ID, err)
			continue
		}

		// Update schedule with last and next run times
		if err := s.repo.UpdateScheduleRunTimes(ctx, schedule.ID, now, nextRun); err != nil {
			log.Printf("scheduler: failed to update run times for schedule %s: %v", schedule.ID, err)
		}
	}

	return jobsCreated, nil
}

// EnqueueScheduledBackup creates and enqueues a backup job for a scheduled backup
func (s *Service) EnqueueScheduledBackup(ctx context.Context, schedule *repository.Schedule) (*repository.Job, error) {
	source, err := s.repo.GetSource(ctx, schedule.SourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	// Build job payload
	payload := types.JobPayload{
		SourceID:     schedule.SourceID,
		CredentialID: source.CredentialID,
		SourceConfig: source.Config,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Scheduled jobs have normal priority (5)
	priority := 5

	job, err := s.repo.CreateJob(ctx, schedule.TenantID, types.JobTypeBackup, &schedule.SourceID, payloadJSON, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Enqueue to Redis
	jobMsg := map[string]any{
		"job_id":      job.ID,
		"tenant_id":   schedule.TenantID,
		"type":        "backup",
		"priority":    priority,
		"schedule_id": schedule.ID,
		"created_at":  job.CreatedAt.Format(time.RFC3339),
	}

	jobMsgJSON, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job message: %w", err)
	}

	if err := s.redis.LPush(ctx, JobQueueKey, jobMsgJSON).Err(); err != nil {
		s.LogSystemError(ctx, "Redis: failed to enqueue scheduled backup job", err, map[string]any{
			"job_id":      job.ID,
			"schedule_id": schedule.ID,
			"source_id":   schedule.SourceID,
		})
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return job, nil
}

// CalculateNextRun calculates the next run time for a schedule based on cron or interval
func (s *Service) CalculateNextRun(schedule *repository.Schedule, from time.Time) (time.Time, error) {
	// Load timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}
	fromInTz := from.In(loc)

	// If using interval_minutes
	if schedule.IntervalMinutes != nil && *schedule.IntervalMinutes > 0 {
		return fromInTz.Add(time.Duration(*schedule.IntervalMinutes) * time.Minute).UTC(), nil
	}

	// If using cron
	if schedule.Cron != nil && *schedule.Cron != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		cronSchedule, err := parser.Parse(*schedule.Cron)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
		}
		return cronSchedule.Next(fromInTz).UTC(), nil
	}

	return time.Time{}, fmt.Errorf("schedule has neither cron nor interval_minutes")
}

// ========== Log Management ==========

// CreateLogRequest is the request to create a log entry
type CreateLogRequest struct {
	Level      string          `json:"level"` // debug, info, warn, error
	Message    string          `json:"message"`
	WorkerID   *string         `json:"worker_id,omitempty"`
	JobID      *string         `json:"job_id,omitempty"`
	SnapshotID *string         `json:"snapshot_id,omitempty"`
	SourceID   *string         `json:"source_id,omitempty"`
	ScheduleID *string         `json:"schedule_id,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

// CreateLog creates a new log entry
func (s *Service) CreateLog(ctx context.Context, req CreateLogRequest) error {
	detailsJSON := json.RawMessage("{}")
	if req.Details != nil {
		detailsJSON = req.Details
	}
	return s.repo.CreateLog(ctx, req.Level, req.Message, req.WorkerID, req.JobID, req.SnapshotID, req.SourceID, req.ScheduleID, detailsJSON)
}

// GetLogsForSnapshot retrieves logs for a specific snapshot
// It queries by both snapshot_id and the snapshot's job_id to get all related logs
// For failed jobs (which don't have snapshot records), it queries by job_id directly
func (s *Service) GetLogsForSnapshot(ctx context.Context, snapshotID string, limit int) ([]*repository.LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	// Get the snapshot to find its job_id
	snapshot, err := s.repo.GetSnapshot(ctx, snapshotID)
	if err != nil {
		// Snapshot not found - this might be a failed job that never created a snapshot.
		// In this case, the provided "snapshotID" is actually a job_id.
		// Query logs by treating the ID as a job_id.
		logs, logErr := s.repo.ListLogsForJob(ctx, snapshotID, limit)
		if logErr != nil {
			return nil, fmt.Errorf("failed to get logs for snapshot/job: %w", logErr)
		}
		return logs, nil
	}

	// Query logs by both snapshot_id and job_id
	logs, err := s.repo.ListLogsForSnapshotWithJob(ctx, snapshotID, snapshot.JobID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for snapshot: %w", err)
	}
	return logs, nil
}

// GetLogsForSource retrieves logs for a specific source
// It queries by source_id and also by related jobs and snapshots
func (s *Service) GetLogsForSource(ctx context.Context, sourceID string, limit int) ([]*repository.LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	// Query logs by source_id and related jobs/snapshots
	logs, err := s.repo.ListLogsForSourceWithJobs(ctx, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for source: %w", err)
	}
	return logs, nil
}

// ListAllLogsAdminParams contains parameters for listing all logs
type ListAllLogsAdminParams struct {
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Level      string `json:"level"`       // Filter by level: debug, info, warn, error, all
	Search     string `json:"search"`      // Search in message
	WorkerID   string `json:"worker_id"`   // Filter by worker_id
	JobID      string `json:"job_id"`      // Filter by job_id
	SnapshotID string `json:"snapshot_id"` // Filter by snapshot_id
	SourceID   string `json:"source_id"`   // Filter by source_id
	ScheduleID string `json:"schedule_id"` // Filter by schedule_id
}

// ListAllLogsAdminResult contains the result of listing all logs
type ListAllLogsAdminResult struct {
	Logs   []*repository.LogEntry `json:"logs"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// ListAllLogsAdmin retrieves all system logs with filtering and pagination (admin only)
func (s *Service) ListAllLogsAdmin(ctx context.Context, params ListAllLogsAdminParams) (*ListAllLogsAdminResult, error) {
	repoParams := repository.ListAllLogsAdminParams{
		Limit:      params.Limit,
		Offset:     params.Offset,
		Level:      params.Level,
		Search:     params.Search,
		WorkerID:   params.WorkerID,
		JobID:      params.JobID,
		SnapshotID: params.SnapshotID,
		SourceID:   params.SourceID,
		ScheduleID: params.ScheduleID,
	}

	result, err := s.repo.ListAllLogsAdmin(ctx, repoParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list all logs: %w", err)
	}

	return &ListAllLogsAdminResult{
		Logs:   result.Logs,
		Total:  result.Total,
		Limit:  params.Limit,
		Offset: params.Offset,
	}, nil
}

// ==================== AUDIT EVENTS ====================

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionCreateSource   AuditAction = "create_source"
	AuditActionUpdateSource   AuditAction = "update_source"
	AuditActionDeleteSource   AuditAction = "delete_source"
	AuditActionCreateSchedule AuditAction = "create_schedule"
	AuditActionUpdateSchedule AuditAction = "update_schedule"
	AuditActionDeleteSchedule AuditAction = "delete_schedule"
	AuditActionDeleteSnapshot AuditAction = "delete_snapshot"
	AuditActionTriggerBackup  AuditAction = "trigger_backup"
	AuditActionCreateTenant   AuditAction = "create_tenant"
	AuditActionDeleteTenant   AuditAction = "delete_tenant"
	AuditActionCreateUser     AuditAction = "create_user"
	AuditActionUpdateUser     AuditAction = "update_user"
	AuditActionDeleteUser     AuditAction = "delete_user"
	AuditActionUpdateSetting  AuditAction = "update_setting"
	AuditActionLogin          AuditAction = "login"
	AuditActionLogout         AuditAction = "logout"
)

// AuditTargetType represents the type of resource being audited
type AuditTargetType string

const (
	AuditTargetSource   AuditTargetType = "source"
	AuditTargetSchedule AuditTargetType = "schedule"
	AuditTargetSnapshot AuditTargetType = "snapshot"
	AuditTargetTenant   AuditTargetType = "tenant"
	AuditTargetUser     AuditTargetType = "user"
	AuditTargetSetting  AuditTargetType = "setting"
)

// CreateAuditEventRequest contains parameters for creating an audit event
type CreateAuditEventRequest struct {
	TenantID    *string         `json:"tenant_id,omitempty"`
	ActorUserID *string         `json:"actor_user_id,omitempty"`
	Action      AuditAction     `json:"action"`
	TargetType  AuditTargetType `json:"target_type"`
	TargetID    string          `json:"target_id"`
	TargetName  string          `json:"target_name"` // Human-readable name for display
	Details     json.RawMessage `json:"details,omitempty"`
	IPAddress   string          `json:"ip_address,omitempty"`
}

// CreateAuditEvent creates a new audit event
func (s *Service) CreateAuditEvent(ctx context.Context, req CreateAuditEventRequest) error {
	var ipPtr *string
	if req.IPAddress != "" {
		ipPtr = &req.IPAddress
	}

	action := string(req.Action)
	targetType := string(req.TargetType)
	targetName := req.TargetName

	_, err := s.repo.CreateAuditEvent(ctx, req.TenantID, req.ActorUserID, action, &targetType, &req.TargetID, &targetName, req.Details, ipPtr)
	if err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	return nil
}

// ListAuditEventsParams contains parameters for listing audit events
type ListAuditEventsParams struct {
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Action     string `json:"action"`      // Filter by action
	TargetType string `json:"target_type"` // Filter by target type
	ActorID    string `json:"actor_id"`    // Filter by actor user ID
	TenantID   string `json:"tenant_id"`   // Filter by tenant ID
	Search     string `json:"search"`      // Search in action
}

// ListAuditEventsResult contains the result of listing audit events
type ListAuditEventsResult struct {
	Events []*repository.AuditEvent `json:"events"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// ListAuditEventsAdmin retrieves all audit events with filtering and pagination (admin only)
func (s *Service) ListAuditEventsAdmin(ctx context.Context, params ListAuditEventsParams) (*ListAuditEventsResult, error) {
	repoParams := repository.ListAuditEventsParams{
		Limit:      params.Limit,
		Offset:     params.Offset,
		Action:     params.Action,
		TargetType: params.TargetType,
		ActorID:    params.ActorID,
		TenantID:   params.TenantID,
		Search:     params.Search,
	}

	result, err := s.repo.ListAuditEventsAdmin(ctx, repoParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	return &ListAuditEventsResult{
		Events: result.Events,
		Total:  result.Total,
		Limit:  params.Limit,
		Offset: params.Offset,
	}, nil
}
