package connector

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SSHConfig represents SSH connection configuration
type SSHConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string // Optional, use Key instead
	Key        string // Private key content
	Paths      []string
}

// SFTPConnector handles SSH/SFTP connections for file downloads
type SFTPConnector struct {
	config *SSHConfig
}

// NewSFTPConnector creates a new SFTP connector
func NewSFTPConnector(config *SSHConfig) *SFTPConnector {
	return &SFTPConnector{
		config: config,
	}
}

// Connect establishes an SSH connection and returns an SFTP client
func (c *SFTPConnector) Connect() (*sftp.Client, *ssh.Client, error) {
	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: c.config.Username,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Make this configurable for production
	}

	// Add authentication method
	if c.config.Key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(c.config.Key))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	} else if c.config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(c.config.Password))
	} else {
		return nil, nil, fmt.Errorf("no authentication method provided")
	}

	// Connect to SSH server
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	sshClient, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return sftpClient, sshClient, nil
}

// PullFiles downloads files from remote paths to a local temporary directory
func (c *SFTPConnector) PullFiles(sftpClient *sftp.Client, destDir string) (*PullStats, error) {
	stats := &PullStats{
		FilesDownloaded: 0,
		TotalBytes:      0,
	}

	for _, path := range c.config.Paths {
		fileStats, err := c.pullPath(sftpClient, path, destDir)
		if err != nil {
			return stats, fmt.Errorf("failed to pull path %s: %w", path, err)
		}
		stats.FilesDownloaded += fileStats.FilesDownloaded
		stats.TotalBytes += fileStats.TotalBytes
	}

	return stats, nil
}

// pullPath recursively downloads a file or directory
func (c *SFTPConnector) pullPath(sftpClient *sftp.Client, remotePath, destDir string) (*PullStats, error) {
	stats := &PullStats{
		FilesDownloaded: 0,
		TotalBytes:      0,
	}

	// Check if remote path exists
	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		return stats, fmt.Errorf("failed to stat remote path: %w", err)
	}

	// Get the base name for the local path
	baseName := filepath.Base(remotePath)
	localPath := filepath.Join(destDir, baseName)

	if info.IsDir() {
		// Recursively pull directory
		walker := sftpClient.Walk(remotePath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				return stats, fmt.Errorf("walk error: %w", err)
			}

			// Calculate relative path from remotePath
			relPath, err := filepath.Rel(remotePath, walker.Path())
			if err != nil {
				return stats, fmt.Errorf("failed to get relative path: %w", err)
			}
			localFilePath := filepath.Join(destDir, baseName, relPath)

			if walker.Stat().IsDir() {
				// Create local directory
				if err := os.MkdirAll(localFilePath, 0755); err != nil {
					return stats, fmt.Errorf("failed to create directory: %w", err)
				}
			} else {
				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
					return stats, fmt.Errorf("failed to create parent directory: %w", err)
				}

				// Download file
				size, err := c.downloadFile(sftpClient, walker.Path(), localFilePath)
				if err != nil {
					return stats, fmt.Errorf("failed to download file %s: %w", walker.Path(), err)
				}
				stats.FilesDownloaded++
				stats.TotalBytes += size
			}
		}
	} else {
		// Pull single file
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return stats, fmt.Errorf("failed to create destination directory: %w", err)
		}

		size, err := c.downloadFile(sftpClient, remotePath, localPath)
		if err != nil {
			return stats, fmt.Errorf("failed to download file: %w", err)
		}
		stats.FilesDownloaded++
		stats.TotalBytes += size
	}

	return stats, nil
}

// downloadFile downloads a single file from SFTP
func (c *SFTPConnector) downloadFile(sftpClient *sftp.Client, remotePath, localPath string) (int64, error) {
	// Open remote file
	srcFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open remote file: %w", err)
	}
	defer srcFile.Close()

	// Create local file
	dstFile, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create local file: %w", err)
	}
	defer dstFile.Close()

	// Copy file content
	size, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("failed to copy file: %w", err)
	}

	return size, nil
}

// PullStats contains statistics about the pulled files
type PullStats struct {
	FilesDownloaded int
	TotalBytes      int64
}
