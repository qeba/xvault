package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"xvault/internal/worker/client"
	"xvault/internal/worker/connector"
	"xvault/internal/worker/packager"
	"xvault/internal/worker/storage"
	"xvault/pkg/crypto"
	"xvault/pkg/types"
)

// Orchestrator manages the worker job execution loop
type Orchestrator struct {
	workerID      string
	hubClient     *client.HubClient
	storage       *storage.Storage
	encryptionKEK string
	pollInterval  time.Duration
}

// NewOrchestrator creates a new worker orchestrator
func NewOrchestrator(workerID string, hubClient *client.HubClient, storageBase, encryptionKEK string) *Orchestrator {
	return &Orchestrator{
		workerID:      workerID,
		hubClient:     hubClient,
		storage:       storage.NewStorage(storageBase),
		encryptionKEK: encryptionKEK,
		pollInterval:  5 * time.Second,
	}
}

// Run starts the worker job loop
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("worker %s starting job loop", o.workerID)

	// Register worker with hub
	if err := o.registerWorker(ctx); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
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
			log.Printf("worker %s shutting down", o.workerID)
			return nil

		case <-heartbeatTicker.C:
			if err := o.sendHeartbeat(ctx, "online"); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}

		case <-pollTicker.C:
			// Try to claim and process a job
			if err := o.processNextJob(ctx); err != nil {
				log.Printf("job processing error: %v", err)
			}
		}
	}
}

// registerWorker registers this worker with the hub
func (o *Orchestrator) registerWorker(ctx context.Context) error {
	req := client.WorkerRegisterRequest{
		WorkerID:        o.workerID,
		Name:            fmt.Sprintf("Worker %s", o.workerID),
		StorageBasePath: o.storage.SnapshotPath("", "", ""), // Get base path
		Capabilities: map[string]any{
			"connectors": []string{"ssh", "sftp"},
			"storage":    []string{"local_fs"},
		},
	}

	// Extract base path from storage
	req.StorageBasePath = "/var/lib/xvault/backups" // Default from env

	return o.hubClient.RegisterWorker(ctx, req)
}

// sendHeartbeat sends a heartbeat to the hub
func (o *Orchestrator) sendHeartbeat(ctx context.Context, status string) error {
	req := client.WorkerHeartbeatRequest{
		WorkerID: o.workerID,
		Status:   status,
	}
	return o.hubClient.SendHeartbeat(ctx, req)
}

// processNextJob attempts to claim and process the next available job
func (o *Orchestrator) processNextJob(ctx context.Context) error {
	// Claim a job
	claimResp, err := o.hubClient.ClaimJob(ctx, o.workerID)
	if err != nil {
		// No job available is not an error
		return nil
	}

	log.Printf("worker %s claimed job %s (type: %s, tenant: %s)", o.workerID, claimResp.JobID, claimResp.Type, claimResp.TenantID)

	// Process the job
	var completeReq client.JobCompleteRequest

	switch claimResp.Type {
	case "backup":
		completeReq, err = o.processBackupJob(ctx, claimResp)
	case "restore", "delete_snapshot":
		completeReq = client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("job type %s not yet implemented", claimResp.Type),
		}
	default:
		completeReq = client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("unknown job type: %s", claimResp.Type),
		}
	}

	// Report job completion
	if err := o.hubClient.CompleteJob(ctx, claimResp.JobID, completeReq); err != nil {
		log.Printf("failed to complete job %s: %v", claimResp.JobID, err)
		return err
	}

	log.Printf("worker %s completed job %s with status: %s", o.workerID, claimResp.JobID, completeReq.Status)
	return nil
}

// processBackupJob processes a backup job
func (o *Orchestrator) processBackupJob(ctx context.Context, job *client.JobClaimResponse) (client.JobCompleteRequest, error) {
	startTime := time.Now()

	// Generate snapshot ID
	snapshotID, err := storage.GenerateSnapshotID()
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to generate snapshot ID: %v", err),
		}, err
	}

	// Create temp directory
	tempDir, err := o.storage.CreateTempDir(job.JobID)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to create temp directory: %v", err),
		}, err
	}
	defer o.storage.CleanupTempDir(tempDir)

	// Parse source config
	var sourceConfig types.SourceConfigSSH
	if err := json.Unmarshal(job.Payload.SourceConfig, &sourceConfig); err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to parse source config: %v", err),
		}, err
	}

	// Fetch credential from hub
	credResp, err := o.hubClient.GetCredential(ctx, job.Payload.CredentialID)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to get credential: %v", err),
		}, err
	}

	// Decrypt credential using platform KEK
	// For v0, credentials are encrypted with platform KEK so workers can decrypt them
	plaintext, err := crypto.DecryptFromStorage(credResp.Ciphertext, o.encryptionKEK)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to decrypt credential: %v", err),
		}, err
	}

	password := string(plaintext)

	// Fetch tenant public key for encrypting the backup
	keyResp, err := o.hubClient.GetTenantPublicKey(ctx, job.TenantID)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to get tenant public key: %v", err),
		}, err
	}

	// Create SSH connector
	sshConfig := &connector.SSHConfig{
		Host:     sourceConfig.Host,
		Port:     sourceConfig.Port,
		Username: sourceConfig.Username,
		Password: password,
		Paths:    sourceConfig.Paths,
	}
	sftpConn := connector.NewSFTPConnector(sshConfig)

	// Connect and pull files
	sftpClient, sshClient, err := sftpConn.Connect()
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to connect: %v", err),
		}, err
	}
	defer sftpClient.Close()
	defer sshClient.Close()

	mirrorDir := tempDir + "/source-mirror"
	stats, err := sftpConn.PullFiles(sftpClient, mirrorDir)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to pull files: %v", err),
		}, err
	}

	log.Printf("pulled %d files (%d bytes) from source", stats.FilesDownloaded, stats.TotalBytes)

	// Package and encrypt
	pkg := packager.NewPackager(keyResp.PublicKey)
	pkgResult, err := pkg.PackageBackup(mirrorDir, snapshotID, job.TenantID, job.SourceID, job.JobID, o.workerID)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to package backup: %v", err),
		}, err
	}

	// Write to local storage
	localPath, sizeBytes, err := o.storage.WriteSnapshot(job.TenantID, job.SourceID, snapshotID, pkgResult.Artifact, pkgResult.Manifest)
	if err != nil {
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to write snapshot: %v", err),
		}, err
	}

	log.Printf("snapshot %s written to %s (%d bytes)", snapshotID, localPath, sizeBytes)

	// Build success response
	finishTime := time.Now()
	durationMs := finishTime.Sub(startTime).Milliseconds()

	return client.JobCompleteRequest{
		WorkerID: o.workerID,
		Status:   "completed",
		Snapshot: &client.SnapshotResult{
			SnapshotID:          snapshotID,
			Status:              "completed",
			SizeBytes:           sizeBytes,
			StartedAt:           startTime.Format(time.RFC3339),
			FinishedAt:          finishTime.Format(time.RFC3339),
			DurationMs:          durationMs,
			ManifestJSON:        pkgResult.Manifest,
			EncryptionAlgorithm: "age-x25519",
			Locator: client.SnapshotLocator{
				StorageBackend: "local_fs",
				WorkerID:       o.workerID,
				LocalPath:      localPath,
			},
		},
	}, nil
}

// Shutdown gracefully shuts down the worker
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	log.Printf("worker %s shutting down...", o.workerID)
	return o.sendHeartbeat(ctx, "draining")
}
