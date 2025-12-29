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

// HubClient is the HTTP client for restore service to communicate with the Hub
type HubClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHubClient creates a new Hub API client for restore service
func NewHubClient(baseURL string) *HubClient {
	return &HubClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for restore operations
		},
	}
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

// RegisterServiceRequest is the request to register a restore service
type RegisterServiceRequest struct {
	ServiceID    string         `json:"service_id"`
	Type         string         `json:"type"` // "restore"
	Name         string         `json:"name"`
	Capabilities map[string]any `json:"capabilities"`
}

// ServiceHeartbeatRequest is the request for service heartbeat
type ServiceHeartbeatRequest struct {
	ServiceID string `json:"service_id"`
	Status    string `json:"status"` // "online", "offline", "draining"
}

// TenantPrivateKeyResponse is the response for tenant private key
type TenantPrivateKeyResponse struct {
	TenantID   string `json:"tenant_id"`
	PrivateKey string `json:"private_key"`
}

// ClaimRestoreJob claims the next available restore job from the Hub
func (c *HubClient) ClaimRestoreJob(ctx context.Context, serviceID string) (*RestoreJobClaimResponse, error) {
	reqBody := map[string]string{
		"service_id": serviceID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/restore-jobs/claim", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to claim restore job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claim restore job failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var claimResp RestoreJobClaimResponse
	if err := json.NewDecoder(resp.Body).Decode(&claimResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &claimResp, nil
}

// CompleteRestoreJob reports restore job completion to the Hub
func (c *HubClient) CompleteRestoreJob(ctx context.Context, jobID string, req RestoreJobCompleteRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/internal/restore-jobs/%s/complete", c.baseURL, jobID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to complete restore job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("complete restore job failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RegisterService registers this restore service with the Hub
func (c *HubClient) RegisterService(ctx context.Context, req RegisterServiceRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/services/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register service failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendHeartbeat sends a heartbeat to the Hub
func (c *HubClient) SendHeartbeat(ctx context.Context, req ServiceHeartbeatRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/services/heartbeat", bytes.NewReader(body))
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

// GetTenantPrivateKey fetches and decrypts a tenant's private key from the Hub
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

// DownloadExpirationResponse is the response for download expiration setting
type DownloadExpirationResponse struct {
	Hours int `json:"hours"`
}

// GetDownloadExpiration fetches the download expiration setting from the Hub
func (c *HubClient) GetDownloadExpiration(ctx context.Context) (*DownloadExpirationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/internal/settings/download-expiration", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get download expiration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get download expiration failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var expirationResp DownloadExpirationResponse
	if err := json.NewDecoder(resp.Body).Decode(&expirationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &expirationResp, nil
}
