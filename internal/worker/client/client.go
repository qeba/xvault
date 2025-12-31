package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HubClient is the HTTP client for communicating with the Hub
type HubClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHubClient creates a new Hub API client
func NewHubClient(baseURL string) *HubClient {
	return &HubClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ClaimJob claims the next available job from the Hub
func (c *HubClient) ClaimJob(ctx context.Context, workerID string) (*JobClaimResponse, error) {
	reqBody := JobClaimRequest{
		WorkerID: workerID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/jobs/claim", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claim job failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var claimResp JobClaimResponse
	if err := json.NewDecoder(resp.Body).Decode(&claimResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &claimResp, nil
}

// CompleteJob reports job completion to the Hub
func (c *HubClient) CompleteJob(ctx context.Context, jobID string, req JobCompleteRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/jobs/%s/complete", c.baseURL, jobID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("complete job failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetCredential fetches an encrypted credential from the Hub
func (c *HubClient) GetCredential(ctx context.Context, credentialID string) (*CredentialResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/internal/credentials/"+credentialID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get credential failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var credResp CredentialResponse
	if err := json.NewDecoder(resp.Body).Decode(&credResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &credResp, nil
}

// GetTenantPublicKey fetches a tenant's public key from the Hub
func (c *HubClient) GetTenantPublicKey(ctx context.Context, tenantID string) (*TenantKeyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/internal/tenants/"+tenantID+"/public-key", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant public key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get tenant public key failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var keyResp TenantKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &keyResp, nil
}

// GetTenantPrivateKey fetches and decrypts a tenant's private key from the Hub
// This is used for restore operations (decrypting backup artifacts)
func (c *HubClient) GetTenantPrivateKey(ctx context.Context, tenantID string) (*TenantPrivateKeyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/internal/tenants/"+tenantID+"/private-key", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant private key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get tenant private key failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var keyResp TenantPrivateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &keyResp, nil
}

// RegisterWorker registers this worker with the Hub
func (c *HubClient) RegisterWorker(ctx context.Context, req WorkerRegisterRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/workers/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register worker failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendHeartbeat sends a heartbeat to the Hub
func (c *HubClient) SendHeartbeat(ctx context.Context, req WorkerHeartbeatRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/workers/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CreateLog creates a log entry in the Hub
func (c *HubClient) CreateLog(ctx context.Context, req LogRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal log request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/logs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create log request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create log failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Request/Response types matching Hub API

type JobClaimRequest struct {
	WorkerID string `json:"worker_id"`
}

type JobClaimResponse struct {
	JobID          string     `json:"job_id"`
	TenantID       string     `json:"tenant_id"`
	SourceID       string     `json:"source_id,omitempty"`
	Type           string     `json:"type"`
	Payload        JobPayload `json:"payload"`
	LeaseExpiresAt string     `json:"lease_expires_at"`
}

type JobPayload struct {
	SourceID          string          `json:"source_id"`
	CredentialID      string          `json:"credential_id"`
	SourceConfig      json.RawMessage `json:"source_config"`
	RestoreSnapshotID *string         `json:"restore_snapshot_id,omitempty"`
	DeleteSnapshotID  *string         `json:"delete_snapshot_id,omitempty"`
}

type JobCompleteRequest struct {
	WorkerID string          `json:"worker_id"`
	Status   string          `json:"status"`
	Error    string          `json:"error,omitempty"`
	Snapshot *SnapshotResult `json:"snapshot,omitempty"`
	Restore  *RestoreResult  `json:"restore,omitempty"`
}

type RestoreResult struct {
	RestoreID     string `json:"restore_id"`
	SnapshotID    string `json:"snapshot_id"`
	Status        string `json:"status"`
	DownloadURL   string `json:"download_url,omitempty"`
	DownloadToken string `json:"download_token,omitempty"`
	SizeBytes     int64  `json:"size_bytes"`
	ExpiresAt     string `json:"expires_at"`
}

type SnapshotResult struct {
	SnapshotID          string          `json:"snapshot_id"`
	Status              string          `json:"status"`
	SizeBytes           int64           `json:"size_bytes"`
	StartedAt           string          `json:"started_at"`
	FinishedAt          string          `json:"finished_at"`
	DurationMs          int64           `json:"duration_ms"`
	ManifestJSON        json.RawMessage `json:"manifest_json"`
	EncryptionAlgorithm string          `json:"encryption_algorithm"`
	Locator             SnapshotLocator `json:"locator"`
}

type SnapshotLocator struct {
	StorageBackend string `json:"storage_backend"`
	WorkerID       string `json:"worker_id,omitempty"`
	LocalPath      string `json:"local_path,omitempty"`
	Bucket         string `json:"bucket,omitempty"`
	ObjectKey      string `json:"object_key,omitempty"`
	ETag           string `json:"etag,omitempty"`
}

type CredentialResponse struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id"`
	Kind       string `json:"kind"`
	Ciphertext string `json:"ciphertext"`
	KeyID      string `json:"key_id"`
}

type TenantKeyResponse struct {
	ID                  string `json:"id"`
	TenantID            string `json:"tenant_id"`
	Algorithm           string `json:"algorithm"`
	PublicKey           string `json:"public_key"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
	KeyStatus           string `json:"key_status"`
}

type TenantPrivateKeyResponse struct {
	TenantID   string `json:"tenant_id"`
	PrivateKey string `json:"private_key"`
}

type WorkerRegisterRequest struct {
	WorkerID        string         `json:"worker_id"`
	Name            string         `json:"name"`
	StorageBasePath string         `json:"storage_base_path"`
	Capabilities    map[string]any `json:"capabilities"`
}

type WorkerHeartbeatRequest struct {
	WorkerID string `json:"worker_id"`
	Status   string `json:"status"`
}

type LogRequest struct {
	Level       string          `json:"level"`
	Message     string          `json:"message"`
	WorkerID    *string         `json:"worker_id,omitempty"`
	JobID       *string         `json:"job_id,omitempty"`
	SnapshotID  *string         `json:"snapshot_id,omitempty"`
	SourceID    *string         `json:"source_id,omitempty"`
	ScheduleID  *string         `json:"schedule_id,omitempty"`
	Details     json.RawMessage `json:"details,omitempty"`
}
