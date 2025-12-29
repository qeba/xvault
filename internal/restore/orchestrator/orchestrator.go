package orchestrator

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xvault/internal/restore/client"
	"xvault/internal/restore/download"
	"xvault/pkg/crypto"
)

// Orchestrator manages the restore service job execution loop
type Orchestrator struct {
	serviceID      string
	hubClient      *client.HubClient
	downloadServer *download.Server
	workerStorage  string // Path to worker storage (read-only shared volume)
	downloadBaseURL string
	pollInterval   time.Duration
}

// NewOrchestrator creates a new restore service orchestrator
func NewOrchestrator(serviceID string, hubClient *client.HubClient, workerStorage, downloadBaseURL string, downloadSrv *download.Server) *Orchestrator {
	return &Orchestrator{
		serviceID:       serviceID,
		hubClient:       hubClient,
		downloadServer:  downloadSrv,
		workerStorage:   workerStorage, // e.g., "/var/lib/xvault/backups" (shared volume)
		downloadBaseURL: downloadBaseURL,
		pollInterval:    5 * time.Second,
	}
}

// Run starts the restore service job loop
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("restore service %s starting job loop", o.serviceID)

	// Register restore service with hub
	if err := o.registerService(ctx); err != nil {
		return fmt.Errorf("failed to register restore service: %w", err)
	}

	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Start job poll ticker
	pollTicker := time.NewTicker(o.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("restore service %s shutting down", o.serviceID)
			return nil

		case <-heartbeatTicker.C:
			if err := o.sendHeartbeat(ctx, "online"); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}

		case <-pollTicker.C:
			// Try to claim and process a restore job
			if err := o.processNextJob(ctx); err != nil {
				log.Printf("job processing error: %v", err)
			}
		}
	}
}

// registerService registers this restore service with the hub
func (o *Orchestrator) registerService(ctx context.Context) error {
	req := client.RegisterServiceRequest{
		ServiceID: o.serviceID,
		Type:      "restore",
		Name:      fmt.Sprintf("Restore Service %s", o.serviceID),
		Capabilities: map[string]any{
			"storage": "local_fs",
		},
	}

	return o.hubClient.RegisterService(ctx, req)
}

// sendHeartbeat sends a heartbeat to the hub
func (o *Orchestrator) sendHeartbeat(ctx context.Context, status string) error {
	req := client.ServiceHeartbeatRequest{
		ServiceID: o.serviceID,
		Status:    status,
	}
	return o.hubClient.SendHeartbeat(ctx, req)
}

// processNextJob attempts to claim and process the next available restore job
func (o *Orchestrator) processNextJob(ctx context.Context) error {
	// Claim a restore job
	claimResp, err := o.hubClient.ClaimRestoreJob(ctx, o.serviceID)
	if err != nil {
		// No job available is not an error
		return nil
	}

	log.Printf("restore service %s claimed job %s (snapshot: %s)", o.serviceID, claimResp.JobID, claimResp.SnapshotID)

	// Process the restore job
	completeReq, err := o.processRestoreJob(ctx, claimResp)

	// Report job completion
	if err := o.hubClient.CompleteRestoreJob(ctx, claimResp.JobID, completeReq); err != nil {
		log.Printf("failed to complete restore job %s: %v", claimResp.JobID, err)
		return err
	}

	// Log completion
	if completeReq.Error != "" {
		log.Printf("restore service %s completed job %s with status: %s, error: %s", o.serviceID, claimResp.JobID, completeReq.Status, completeReq.Error)
	} else {
		log.Printf("restore service %s completed job %s with status: %s, download_url: %s", o.serviceID, claimResp.JobID, completeReq.Status, completeReq.DownloadURL)
	}

	return nil
}

// processRestoreJob processes a restore job - decrypts the backup and creates a downloadable ZIP
func (o *Orchestrator) processRestoreJob(ctx context.Context, job *client.RestoreJobClaimResponse) (client.RestoreJobCompleteRequest, error) {
	startTime := time.Now()

	log.Printf("restore service %s processing restore for snapshot %s", o.serviceID, job.SnapshotID)

	// Use the LocalPath from the claim response (contains actual directory name)
	// Fallback to constructed path if LocalPath is empty
	var snapshotPath string
	if job.LocalPath != "" {
		snapshotPath = job.LocalPath
	} else {
		snapshotPath = filepath.Join(o.workerStorage, "tenants", job.TenantID, "sources", job.SourceID, "snapshots", job.SnapshotID)
	}
	backupPath := filepath.Join(snapshotPath, "backup.tar.zst.enc")

	// Read the manifest to verify encryption info
	manifestPath := filepath.Join(snapshotPath, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to read manifest: %v", err),
		}, err
	}

	var manifest struct {
		EncryptionAlgorithm string `json:"encryption_algorithm"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to parse manifest: %v", err),
		}, err
	}

	// Get tenant private key for decryption
	keyResp, err := o.hubClient.GetTenantPrivateKey(ctx, job.TenantID)
	if err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to get tenant private key: %v", err),
		}, err
	}

	// Read the encrypted backup file
	encryptedData, err := os.ReadFile(backupPath)
	if err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to read encrypted backup: %v", err),
		}, err
	}

	// Decrypt the backup using Age
	decryptedData, err := crypto.DecryptWithPrivateKey(encryptedData, keyResp.PrivateKey)
	if err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to decrypt backup: %v", err),
		}, err
	}

	log.Printf("decrypted backup for snapshot %s (%d bytes)", job.SnapshotID, len(decryptedData))

	// Create temp directory for processing
	tempDir := filepath.Join(os.TempDir(), "restore-"+job.SnapshotID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to create temp directory: %v", err),
		}, err
	}
	defer os.RemoveAll(tempDir)

	// Write the decrypted tar.zst to temp
	decryptedPath := filepath.Join(tempDir, "backup.tar.zst")
	if err := os.WriteFile(decryptedPath, decryptedData, 0644); err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to write decrypted data: %v", err),
		}, err
	}

	// Create ZIP file from the decrypted archive
	zipPath := filepath.Join(tempDir, "restore-"+job.SnapshotID+".zip")
	if err := o.createZipFromDecrypted(decryptedPath, zipPath); err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to create zip: %v", err),
		}, err
	}

	// Get file size for response
	zipInfo, _ := os.Stat(zipPath)
	zipSize := int64(0)
	if zipInfo != nil {
		zipSize = zipInfo.Size()
	}

	// Move zip to downloads directory
	downloadsDir := "/var/lib/xvault/downloads"
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to create downloads directory: %v", err),
		}, err
	}

	finalZipPath := filepath.Join(downloadsDir, "restore-"+job.SnapshotID+".zip")
	// Move zip to downloads directory
	// Try rename first (fast, works on same filesystem)
	err = os.Rename(zipPath, finalZipPath)
	if err != nil {
		// Cross-device link error: fall back to copy
		// This happens when /tmp and /var/lib/xvault/downloads are on different filesystems
		if err.Error() == "invalid cross-device link" || strings.Contains(err.Error(), "cross-device") {
			log.Printf("restore service %s: cross-device link detected, copying file instead", o.serviceID)
			if copyErr := copyFile(zipPath, finalZipPath); copyErr != nil {
				return client.RestoreJobCompleteRequest{
					ServiceID: o.serviceID,
					Status:    "failed",
					Error:     fmt.Sprintf("failed to copy zip to downloads: %v", copyErr),
				}, copyErr
			}
			// Remove source file after successful copy
			os.Remove(zipPath)
		} else {
			return client.RestoreJobCompleteRequest{
				ServiceID: o.serviceID,
				Status:    "failed",
				Error:     fmt.Sprintf("failed to move zip to downloads: %v", err),
			}, err
		}
	}

	// Register download and get token
	token, expiresAt, err := o.downloadServer.RegisterDownload(job.SnapshotID, finalZipPath)
	if err != nil {
		return client.RestoreJobCompleteRequest{
			ServiceID: o.serviceID,
			Status:    "failed",
			Error:     fmt.Sprintf("failed to register download: %v", err),
		}, err
	}

	downloadURL := o.downloadServer.GetDownloadURL(token, o.downloadBaseURL)

	log.Printf("restore service %s completed restore for snapshot %s: download URL: %s", o.serviceID, job.SnapshotID, downloadURL)

	// Build success response
	finishTime := time.Now()
	durationMs := finishTime.Sub(startTime).Milliseconds()

	return client.RestoreJobCompleteRequest{
		ServiceID:     o.serviceID,
		Status:        "completed",
		DownloadURL:   downloadURL,
		DownloadToken: token,
		SizeBytes:     zipSize,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		DurationMs:    durationMs,
	}, nil
}

// createZipFromDecrypted creates a ZIP file from the decrypted tar.zst archive
func (o *Orchestrator) createZipFromDecrypted(decryptedPath, zipPath string) error {
	// Create a new ZIP file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Open the decrypted file
	srcFile, err := os.Open(decryptedPath)
	if err != nil {
		return fmt.Errorf("failed to open decrypted file: %w", err)
	}
	defer srcFile.Close()

	// Get file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat decrypted file: %w", err)
	}

	// Create a file in the ZIP
	header := &zip.FileHeader{
		Name:   filepath.Base(decryptedPath),
		Method: zip.Deflate,
	}
	header.SetModTime(time.Now())
	header.SetMode(0644)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	// Copy the file content
	if _, err := io.Copy(writer, srcFile); err != nil {
		return fmt.Errorf("failed to write to zip: %w", err)
	}

	log.Printf("created zip file: %s (size: %d bytes)", zipPath, srcInfo.Size())
	return nil
}

// Shutdown gracefully shuts down the restore service
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	log.Printf("restore service %s shutting down...", o.serviceID)
	return o.sendHeartbeat(ctx, "draining")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
