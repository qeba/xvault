package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"xvault/internal/hub/repository"
	"xvault/pkg/crypto"
	"xvault/pkg/types"
)

const (
	// JobQueueKey is the Redis key for the job queue
	JobQueueKey = "xvault:jobs:queue"
	// JobLeaseDuration is how long a worker has to complete a job
	JobLeaseDuration = 30 * time.Minute
)

// Service handles business logic for the Hub
type Service struct {
	repo       *repository.Repository
	redis      *redis.Client
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

// CreateTenantRequest is the request to create a tenant
type CreateTenantRequest struct {
	Name string `json:"name"`
}

// CreateTenantResponse is the response when creating a tenant
type CreateTenantResponse struct {
	Tenant    *repository.Tenant  `json:"tenant"`
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
	TenantID   string `json:"tenant_id"`
	Kind       string `json:"kind"` // "source" or "storage"
	Plaintext  string `json:"plaintext"` // Base64-encoded plaintext secret
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
	if err := s.repo.UpdateWorkerHeartbeat(ctx, req.WorkerID, req.Status); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
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
	var maxAgeTime time.Time
	if policy.MaxAgeDays != nil {
		maxAge := time.Duration(*policy.MaxAgeDays) * 24 * time.Hour
		maxAgeTime = now.Add(-maxAge)
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

	// Evaluate policy
	result, err := s.EvaluateRetentionPolicy(ctx, schedule.TenantID, sourceID, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate retention policy: %w", err)
	}

	// Enqueue delete jobs for snapshots to delete
	for _, snapshotID := range result.SnapshotsToDelete {
		_, err := s.EnqueueDeleteJob(ctx, snapshotID)
		if err != nil {
			// Log error but continue with other snapshots
			// TODO: Add proper logging
			_ = err
		}
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
			SourceID:          schedule.SourceID,
			EvaluationResult:  result,
			JobsEnqueued:      len(result.SnapshotsToDelete),
		})
	}

	return results, nil
}

// SourceRetentionResult represents the result of running retention evaluation for a source
type SourceRetentionResult struct {
	SourceID         string                              `json:"source_id"`
	EvaluationResult *types.RetentionEvaluationResult    `json:"evaluation_result,omitempty"`
	JobsEnqueued     int                                 `json:"jobs_enqueued"`
	Error            string                              `json:"error,omitempty"`
}

// Schedule management

// CreateScheduleRequest is the request to create a schedule
type CreateScheduleRequest struct {
	TenantID         string             `json:"tenant_id"`
	SourceID         string             `json:"source_id"`
	Cron             *string            `json:"cron,omitempty"`
	IntervalMinutes  *int               `json:"interval_minutes,omitempty"`
	Timezone         string             `json:"timezone,omitempty"` // Default "UTC"
	RetentionPolicy  types.RetentionPolicy `json:"retention_policy"`
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

	return schedule, nil
}

// UpdateScheduleRequest is the request to update a schedule
type UpdateScheduleRequest struct {
	Cron            *string            `json:"cron,omitempty"`
	IntervalMinutes *int               `json:"interval_minutes,omitempty"`
	Timezone        *string            `json:"timezone,omitempty"`
	Status          *string            `json:"status,omitempty"` // "enabled" or "disabled"
	RetentionPolicy  *types.RetentionPolicy `json:"retention_policy,omitempty"`
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
	if req.Timezone != nil && *req.Timezone != "" {
		timezone = *req.Timezone
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
