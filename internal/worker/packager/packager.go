package packager

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"strconv"

	"github.com/klauspost/compress/zstd"
	"xvault/pkg/crypto"
	"xvault/pkg/types"
)

// Packager handles backup packaging, compression, and encryption
type Packager struct {
	tenantPublicKey string
}

// NewPackager creates a new packager for a tenant
func NewPackager(tenantPublicKey string) *Packager {
	return &Packager{
		tenantPublicKey: tenantPublicKey,
	}
}

// PackageBackup creates an encrypted backup artifact from a source directory
func (p *Packager) PackageBackup(sourceDir, snapshotID, tenantID, sourceID, jobID, workerID string) (*PackageResult, error) {
	startTime := time.Now()

	// Create a buffer for the tar archive
	var tarBuf bytes.Buffer

	// Calculate total size and count files
	fileCount, _, err := p.walkSourceDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to walk source directory: %w", err)
	}

	// Create tar archive
	tarSize, err := p.createTarArchive(sourceDir, &tarBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar archive: %w", err)
	}

	// Compress with zstd
	compressed, err := p.compressZstd(tarBuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to compress: %w", err)
	}

	// Encrypt with Age
	encrypted, err := crypto.EncryptToPublicKey(compressed, p.tenantPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	// Calculate SHA256 of encrypted artifact
	hash := sha256.Sum256(encrypted)
	sha256Hash := hex.EncodeToString(hash[:])

	finishTime := time.Now()
	durationMs := finishTime.Sub(startTime).Milliseconds()

	// Create manifest
	manifest := types.SnapshotManifest{
		TenantID:   tenantID,
		SourceID:   sourceID,
		SnapshotID: snapshotID,
		JobID:      jobID,
		WorkerID:   workerID,
		StartedAt:  startTime.Format(time.RFC3339),
		FinishedAt: finishTime.Format(time.RFC3339),
		DurationMs: durationMs,
		SizeBytes:  int64(len(encrypted)),
		SHA256:     sha256Hash,
		EncryptionAlgorithm: "age-x25519",
		EncryptionKeyID:     p.tenantPublicKey[:16], // First 16 chars of public key as ID
		EncryptionRecipient: p.tenantPublicKey,
		ContentSummary: types.ContentSummary{
			Type:      "files",
			FileCount: fileCount,
		},
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return &PackageResult{
		Artifact:         encrypted,
		Manifest:         manifestJSON,
		ManifestObj:      manifest,
		UncompressedSize: tarSize,
		CompressedSize:   int64(len(compressed)),
		EncryptedSize:    int64(len(encrypted)),
		SHA256:           sha256Hash,
	}, nil
}

// walkSourceDir walks the source directory and counts files/bytes
func (p *Packager) walkSourceDir(sourceDir string) (fileCount int, totalBytes int64, err error) {
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			totalBytes += info.Size()
		}
		return nil
	})
	return fileCount, totalBytes, err
}

// createTarArchive creates a tar archive of the source directory
func (p *Packager) createTarArchive(sourceDir string, w io.Writer) (int64, error) {
	// Use a simple tar implementation
	// For production, consider using archive/tar for better cross-platform support
	return createSimpleTar(sourceDir, w)
}

// compressZstd compresses data with zstandard
func (p *Packager) compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, nil)
	return compressed, nil
}

// PackageResult contains the result of packaging a backup
type PackageResult struct {
	Artifact         []byte
	Manifest         []byte
	ManifestObj      types.SnapshotManifest
	UncompressedSize int64
	CompressedSize   int64
	EncryptedSize    int64
	SHA256           string
}

// createSimpleTar creates a simple tar archive
// For v0, this is a simplified implementation. For production, use archive/tar.
func createSimpleTar(sourceDir string, w io.Writer) (int64, error) {
	sourceDir = filepath.Clean(sourceDir)

	var totalSize int64
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourceDir {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if info.IsDir() {
			// For directories, we could create directory entries
			// For simplicity, we'll rely on file paths to imply structure
			return nil
		}

		// Read file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Write a simple tar header (ustar format)
		header := makeTarHeader(relPath, info.Size(), info.Mode())
		if _, err := w.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}

		// Write file data
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write file data: %w", err)
		}

		// Write padding to 512-byte boundary
		padding := (512 - (info.Size() % 512)) % 512
		if padding > 0 {
			if _, err := w.Write(make([]byte, padding)); err != nil {
				return fmt.Errorf("failed to write padding: %w", err)
			}
		}

		totalSize += info.Size()
		return nil
	})

	if err != nil {
		return 0, err
	}

	// Write two 512-byte zero blocks to end the archive
	endBlocks := make([]byte, 1024)
	if _, err := w.Write(endBlocks); err != nil {
		return 0, fmt.Errorf("failed to write end blocks: %w", err)
	}

	return totalSize, nil
}

// makeTarHeader creates a simple tar header
func makeTarHeader(name string, size int64, mode os.FileMode) []byte {
	header := make([]byte, 512)

	// Name (100 bytes)
	copy(header[0:100], name)
	// Mode (8 bytes) - octal
	modeStr := fmt.Sprintf("%06o ", mode&0777)
	copy(header[100:108], modeStr)
	// UID (8 bytes)
	copy(header[108:116], "0000000 ")
	// GID (8 bytes)
	copy(header[116:124], "0000000 ")
	// Size (12 bytes) - octal
	sizeStr := fmt.Sprintf("%011o ", size)
	copy(header[124:136], sizeStr)
	// Mtime (12 bytes) - octal (use 0 for simplicity)
	copy(header[136:148], "00000000000 ")
	// Type flag (1 byte) - regular file
	header[156] = 0x30 // '0'
	// Magic (6 bytes) + version (2 bytes)
	copy(header[257:263], "ustar")
	header[263] = 0x30 // '0'
	header[264] = 0x30 // '0'
	// Prefix (155 bytes) - empty for simplicity

	// Calculate checksum
	sum := checksum(header)
	sumStr := fmt.Sprintf("%06o", sum)
	copy(header[148:155], sumStr)
	header[155] = ' ' // null terminated

	return header
}

// checksum calculates the tar header checksum
func checksum(header []byte) int64 {
	var sum int64
	for i := 0; i < 512; i++ {
		// Skip the checksum field itself (bytes 148-155)
		if i >= 148 && i < 156 {
			sum += int64(' ')
		} else {
			sum += int64(header[i])
		}
	}
	return sum
}

// parseOctal parses an octal string from a byte slice
func parseOctal(b []byte) int64 {
	// Find null terminator
	end := len(b)
	for i, c := range b {
		if c == 0 || c == ' ' {
			end = i
			break
		}
	}
	s := string(b[:end])
	val, err := strconv.ParseInt(s, 8, 64)
	if err != nil {
		return 0
	}
	return val
}
