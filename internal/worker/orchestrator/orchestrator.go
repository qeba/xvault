package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"xvault/internal/worker/client"
	"xvault/internal/worker/connector"
	"xvault/internal/worker/metrics"
	"xvault/internal/worker/packager"
	"xvault/internal/worker/storage"
	"xvault/pkg/crypto"
	"xvault/pkg/types"
)

// Orchestrator manages the worker job execution loop
type Orchestrator struct {
	workerID         string
	hubClient        *client.HubClient
	storage          *storage.Storage
	encryptionKEK    string
	pollInterval     time.Duration
	metricsCollector *metrics.Collector
	activeJobs       int32
	storageBasePath  string
}

// NewOrchestrator creates a new worker orchestrator
func NewOrchestrator(workerID string, hubClient *client.HubClient, storageBase, encryptionKEK string) *Orchestrator {
	o := &Orchestrator{
		workerID:        workerID,
		hubClient:       hubClient,
		storage:         storage.NewStorage(storageBase),
		encryptionKEK:   encryptionKEK,
		pollInterval:    5 * time.Second,
		storageBasePath: storageBase,
	}
	// Initialize metrics collector with pointer to active jobs counter
	activeJobsPtr := new(int)
	o.metricsCollector = metrics.NewCollector(storageBase, activeJobsPtr)
	return o
}

// logToHub sends a log entry to the hub
func (o *Orchestrator) logToHub(ctx context.Context, level, message string, jobID, snapshotID, sourceID, scheduleID *string, details map[string]any) {
	detailsJSON, _ := json.Marshal(details)
	req := client.LogRequest{
		Level:      level,
		Message:    message,
		WorkerID:   &o.workerID,
		JobID:      jobID,
		SnapshotID: snapshotID,
		SourceID:   sourceID,
		ScheduleID: scheduleID,
		Details:    detailsJSON,
	}
	if err := o.hubClient.CreateLog(ctx, req); err != nil {
		log.Printf("failed to send log to hub: %v", err)
	}
}

// Run starts the worker job loop
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Printf("worker %s starting job loop", o.workerID)

	// Register worker with hub
	if err := o.registerWorker(ctx); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Send initial heartbeat
	if err := o.sendHeartbeat(ctx, "online"); err != nil {
		log.Printf("initial heartbeat failed: %v", err)
	}

	// Start heartbeat in separate goroutine so it continues during job processing
	go func() {
		heartbeatTicker := time.NewTicker(30 * time.Second)
		defer heartbeatTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				if err := o.sendHeartbeat(ctx, "online"); err != nil {
					log.Printf("heartbeat failed: %v", err)
				}
			}
		}
	}()

	// Start job poll ticker
	pollTicker := time.NewTicker(o.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %s shutting down", o.workerID)
			return nil

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

// sendHeartbeat sends a heartbeat to the hub with system metrics
func (o *Orchestrator) sendHeartbeat(ctx context.Context, status string) error {
	// Collect system metrics
	sysMetrics := o.metricsCollector.Collect()
	// Update active jobs count from atomic counter
	sysMetrics.ActiveJobs = int(atomic.LoadInt32(&o.activeJobs))

	req := client.WorkerHeartbeatRequest{
		WorkerID:      o.workerID,
		Status:        status,
		SystemMetrics: sysMetrics,
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

	// Increment active jobs counter
	atomic.AddInt32(&o.activeJobs, 1)
	defer atomic.AddInt32(&o.activeJobs, -1)

	log.Printf("worker %s claimed job %s (type: %s, tenant: %s)", o.workerID, claimResp.JobID, claimResp.Type, claimResp.TenantID)
	o.logToHub(ctx, "info", fmt.Sprintf("claimed job %s (type: %s)", claimResp.JobID, claimResp.Type), &claimResp.JobID, nil, &claimResp.SourceID, nil, map[string]any{
		"tenant_id": claimResp.TenantID,
		"job_type":  claimResp.Type,
	})

	// Process the job
	var completeReq client.JobCompleteRequest

	switch claimResp.Type {
	case "backup":
		completeReq, err = o.processBackupJob(ctx, claimResp)
	case "delete_snapshot":
		completeReq, err = o.processDeleteSnapshotJob(ctx, claimResp)
	case "restore":
		// Restore jobs are handled by the separate restore service
		completeReq = client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    "restore jobs must be handled by restore service, not worker",
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

	// Log job completion with error details if failed
	if completeReq.Error != "" {
		log.Printf("worker %s completed job %s with status: %s, error: %s", o.workerID, claimResp.JobID, completeReq.Status, completeReq.Error)
		o.logToHub(ctx, "error", fmt.Sprintf("completed job %s with error: %s", claimResp.JobID, completeReq.Error), &claimResp.JobID, nil, nil, nil, map[string]any{
			"status": completeReq.Status,
			"error":  completeReq.Error,
		})
	} else {
		log.Printf("worker %s completed job %s with status: %s", o.workerID, claimResp.JobID, completeReq.Status)
		o.logToHub(ctx, "info", fmt.Sprintf("completed job %s successfully", claimResp.JobID), &claimResp.JobID, nil, nil, nil, map[string]any{
			"status": completeReq.Status,
		})
	}
	return nil
}

// processBackupJob processes a backup job
func (o *Orchestrator) processBackupJob(ctx context.Context, job *client.JobClaimResponse) (client.JobCompleteRequest, error) {
	startTime := time.Now()

	// Generate snapshot ID
	snapshotID, err := storage.GenerateSnapshotID()
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to generate snapshot ID: %v", err), &job.JobID, nil, nil, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to generate snapshot ID: %v", err),
		}, err
	}

	// Create temp directory
	tempDir, err := o.storage.CreateTempDir(job.JobID)
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to create temp directory: %v", err), &job.JobID, nil, nil, nil, nil)
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
		o.logToHub(ctx, "error", fmt.Sprintf("failed to parse source config: %v", err), &job.JobID, nil, nil, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to parse source config: %v", err),
		}, err
	}

	// Fetch credential from hub
	credResp, err := o.hubClient.GetCredential(ctx, job.Payload.CredentialID)
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to get credential: %v", err), &job.JobID, nil, nil, nil, nil)
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
		o.logToHub(ctx, "error", fmt.Sprintf("failed to decrypt credential: %v", err), &job.JobID, nil, nil, nil, nil)
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
		o.logToHub(ctx, "error", fmt.Sprintf("failed to get tenant public key: %v", err), &job.JobID, nil, nil, nil, nil)
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
		o.logToHub(ctx, "error", fmt.Sprintf("failed to connect: %v", err), &job.JobID, nil, nil, nil, nil)
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
		o.logToHub(ctx, "error", fmt.Sprintf("failed to pull files: %v", err), &job.JobID, nil, nil, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to pull files: %v", err),
		}, err
	}

	log.Printf("pulled %d files (%d bytes) from source", stats.FilesDownloaded, stats.TotalBytes)
	o.logToHub(ctx, "info", fmt.Sprintf("pulled %d files (%d bytes) from source", stats.FilesDownloaded, stats.TotalBytes), &job.JobID, nil, &job.SourceID, nil, map[string]any{
		"files_downloaded": stats.FilesDownloaded,
		"total_bytes":      stats.TotalBytes,
	})

	// Package and encrypt
	pkg := packager.NewPackager(keyResp.PublicKey)
	pkgResult, err := pkg.PackageBackup(mirrorDir, snapshotID, job.TenantID, job.SourceID, job.JobID, o.workerID)
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to package backup: %v", err), &job.JobID, &snapshotID, nil, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to package backup: %v", err),
		}, err
	}

	// Write to local storage
	localPath, sizeBytes, err := o.storage.WriteSnapshot(job.TenantID, job.SourceID, snapshotID, pkgResult.Artifact, pkgResult.Manifest)
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to write snapshot: %v", err), &job.JobID, &snapshotID, &job.SourceID, nil, map[string]any{
			"local_path": localPath,
			"size_bytes": sizeBytes,
		})
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to write snapshot: %v", err),
		}, err
	}

	log.Printf("snapshot %s written to %s (%d bytes)", snapshotID, localPath, sizeBytes)
	o.logToHub(ctx, "info", fmt.Sprintf("snapshot %s written to %s (%d bytes)", snapshotID, localPath, sizeBytes), &job.JobID, &snapshotID, &job.SourceID, nil, map[string]any{
		"local_path": localPath,
		"size_bytes": sizeBytes,
	})

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

// processDeleteSnapshotJob processes a delete_snapshot job
func (o *Orchestrator) processDeleteSnapshotJob(ctx context.Context, job *client.JobClaimResponse) (client.JobCompleteRequest, error) {
	// Extract snapshot ID from payload
	if job.Payload.DeleteSnapshotID == nil || *job.Payload.DeleteSnapshotID == "" {
		o.logToHub(ctx, "error", "delete_snapshot_id is required in payload", &job.JobID, nil, nil, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    "delete_snapshot_id is required in payload",
		}, fmt.Errorf("missing delete_snapshot_id")
	}

	snapshotID := *job.Payload.DeleteSnapshotID

	log.Printf("worker %s deleting snapshot %s", o.workerID, snapshotID)
	o.logToHub(ctx, "info", fmt.Sprintf("deleting snapshot %s", snapshotID), &job.JobID, &snapshotID, &job.SourceID, nil, nil)

	// Delete snapshot from local storage
	err := o.storage.DeleteSnapshot(job.TenantID, job.SourceID, snapshotID)
	if err != nil {
		o.logToHub(ctx, "error", fmt.Sprintf("failed to delete snapshot: %v", err), &job.JobID, &snapshotID, &job.SourceID, nil, nil)
		return client.JobCompleteRequest{
			WorkerID: o.workerID,
			Status:   "failed",
			Error:    fmt.Sprintf("failed to delete snapshot: %v", err),
		}, err
	}

	log.Printf("worker %s successfully deleted snapshot %s", o.workerID, snapshotID)
	o.logToHub(ctx, "info", fmt.Sprintf("successfully deleted snapshot %s", snapshotID), &job.JobID, &snapshotID, &job.SourceID, nil, nil)

	return client.JobCompleteRequest{
		WorkerID: o.workerID,
		Status:   "completed",
	}, nil
}

// Shutdown gracefully shuts down the worker
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	log.Printf("worker %s shutting down...", o.workerID)
	return o.sendHeartbeat(ctx, "draining")
}
