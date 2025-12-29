package service

import (
	"context"
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

	// If snapshot was created, store it
	if req.Snapshot != nil {
		// Get job details to find tenant_id and source_id
		job, err := s.repo.GetJob(ctx, jobID) // Need to add this method to repository
		if err != nil {
			return fmt.Errorf("failed to get job details: %w", err)
		}

		_, err = s.repo.CreateSnapshot(ctx, job.TenantID, *job.SourceID, jobID, *req.Snapshot)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
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
