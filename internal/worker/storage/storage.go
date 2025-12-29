package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Storage handles local storage of backup artifacts
type Storage struct {
	basePath string
}

// NewStorage creates a new storage manager
func NewStorage(basePath string) *Storage {
	return &Storage{
		basePath: basePath,
	}
}

// SnapshotPath returns the path for a snapshot
func (s *Storage) SnapshotPath(tenantID, sourceID, snapshotID string) string {
	return filepath.Join(s.basePath, "tenants", tenantID, "sources", sourceID, "snapshots", snapshotID)
}

// GetSnapshotPath is an alias for SnapshotPath for clarity
func (s *Storage) GetSnapshotPath(tenantID, sourceID, snapshotID string) (string, error) {
	path := s.SnapshotPath(tenantID, sourceID, snapshotID)
	// Verify the path exists
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("snapshot not found: %w", err)
	}
	return path, nil
}

// WriteSnapshot writes the encrypted backup artifact and metadata to disk
func (s *Storage) WriteSnapshot(tenantID, sourceID, snapshotID string, artifact []byte, manifest []byte) (string, int64, error) {
	snapshotPath := s.SnapshotPath(tenantID, sourceID, snapshotID)

	// Create the snapshot directory
	if err := os.MkdirAll(snapshotPath, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Write the encrypted artifact
	artifactPath := filepath.Join(snapshotPath, "backup.tar.zst.enc")
	if err := os.WriteFile(artifactPath, artifact, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write artifact: %w", err)
	}

	// Write the manifest
	manifestPath := filepath.Join(snapshotPath, "manifest.json")
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Write meta.json
	meta := Meta{
		TenantID:  tenantID,
		SourceID:  sourceID,
		SnapshotID: snapshotID,
	}
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal meta: %w", err)
	}
	metaPath := filepath.Join(snapshotPath, "meta.json")
	if err := os.WriteFile(metaPath, metaJSON, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write meta: %w", err)
	}

	return snapshotPath, int64(len(artifact)), nil
}

// DeleteSnapshot removes a snapshot from local storage
func (s *Storage) DeleteSnapshot(tenantID, sourceID, snapshotID string) error {
	snapshotPath := s.SnapshotPath(tenantID, sourceID, snapshotID)
	if err := os.RemoveAll(snapshotPath); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	return nil
}

// CleanupTempDir removes a temporary directory
func (s *Storage) CleanupTempDir(tempDir string) error {
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to cleanup temp directory: %w", err)
	}
	return nil
}

// CreateTempDir creates a temporary directory for a job
func (s *Storage) CreateTempDir(jobID string) (string, error) {
	tempDir := filepath.Join("/tmp", "gobackup", jobID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create source-mirror subdirectory
	mirrorDir := filepath.Join(tempDir, "source-mirror")
	if err := os.MkdirAll(mirrorDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create mirror directory: %w", err)
	}

	return tempDir, nil
}

// GenerateSnapshotID generates a new snapshot ID
func GenerateSnapshotID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Meta represents the meta.json file stored with each snapshot
type Meta struct {
	TenantID   string `json:"tenant_id"`
	SourceID   string `json:"source_id"`
	SnapshotID string `json:"snapshot_id"`
	JobID      string `json:"job_id,omitempty"`
	WorkerID   string `json:"worker_id,omitempty"`
}
