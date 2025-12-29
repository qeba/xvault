package download

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Server handles HTTP downloads for restored backups
type Server struct {
	app         *fiber.App
	downloadsDir string
	tokens      map[string]*tokenInfo
	mu          sync.RWMutex
	downloadExpirationHours int // Configurable expiration time in hours
}

// tokenInfo holds information about a download token
type tokenInfo struct {
	filePath  string
	createdAt time.Time
	expiresAt time.Time
	snapshotID string
}

// NewServer creates a new download server
func NewServer(listenAddr, downloadsDir string, downloadExpirationHours int) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             100 * 1024 * 1024 * 1024, // 100GB max download
	})

	s := &Server{
		app:         app,
		downloadsDir: downloadsDir,
		tokens:      make(map[string]*tokenInfo),
		downloadExpirationHours: downloadExpirationHours,
	}

	// Setup routes
	s.setupRoutes()

	// Start cleanup goroutine
	go s.cleanupExpiredTokens()

	// Start server in background
	go func() {
		log.Printf("restore download server listening on %s", listenAddr)
		if err := app.Listen(listenAddr); err != nil {
			log.Printf("restore download server error: %v", err)
		}
	}()

	return s
}

// setupRoutes configures the download routes
func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	// Download endpoint with token
	s.app.Get("/download/:token", func(c *fiber.Ctx) error {
		token := c.Params("token")
		if token == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token is required"})
		}

		s.mu.RLock()
		info, exists := s.tokens[token]
		s.mu.RUnlock()

		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		// Check if token has expired
		if time.Now().After(info.expiresAt) {
			s.mu.Lock()
			delete(s.tokens, token)
			s.mu.Unlock()
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "download token has expired"})
		}

		// Serve the file
		return c.Download(info.filePath, filepath.Base(info.filePath))
	})
}

// RegisterDownload registers a file for download and returns a token
func (s *Server) RegisterDownload(snapshotID, filePath string) (token string, expiresAt time.Time, err error) {
	// Generate a unique token
	token = generateToken()
	expiresAt = time.Now().Add(time.Duration(s.downloadExpirationHours) * time.Hour)

	s.mu.Lock()
	s.tokens[token] = &tokenInfo{
		filePath:   filePath,
		createdAt:  time.Now(),
		expiresAt:  expiresAt,
		snapshotID: snapshotID,
	}
	s.mu.Unlock()

	log.Printf("[restore] registered download for snapshot %s: token=%s, expires=%s (hours=%d)", snapshotID, token, expiresAt.Format(time.RFC3339), s.downloadExpirationHours)

	return token, expiresAt, nil
}

// GetDownloadURL returns the download URL for a given token
func (s *Server) GetDownloadURL(token string, baseURL string) string {
	return baseURL + "/download/" + token
}

// cleanupExpiredTokens removes expired tokens from memory
func (s *Server) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, info := range s.tokens {
			if now.After(info.expiresAt) {
				log.Printf("[restore] cleaning up expired token: %s (snapshot: %s)", token, info.snapshotID)
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

// Shutdown stops the download server
func (s *Server) Shutdown() error {
	log.Println("shutting down restore download server")
	return s.app.Shutdown()
}

// generateToken generates a random token for download access
func generateToken() string {
	// Use timestamp + random string for simplicity
	// For production, use a more secure random generator
	return time.Now().Format("20060102150405") + "-" + randomString(16)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		// Simple random generation using time nanos (not cryptographically secure but sufficient for v0)
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
