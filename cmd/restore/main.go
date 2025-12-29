package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xvault/internal/restore/client"
	"xvault/internal/restore/download"
	"xvault/internal/restore/orchestrator"
)

func main() {
	serviceID := mustGetenv("RESTORE_SERVICE_ID")
	hubBaseURL := mustGetenv("HUB_BASE_URL")
	workerStorage := getenv("WORKER_STORAGE_BASE", "/var/lib/xvault/backups") // Shared volume (read-only)
	downloadListenAddr := getenv("RESTORE_DOWNLOAD_ADDR", ":8082")
	downloadBaseURL := getenv("RESTORE_DOWNLOAD_BASE_URL", "http://"+downloadListenAddr)

	log.Printf("restore service starting: service_id=%s hub=%s worker_storage=%s", serviceID, hubBaseURL, workerStorage)

	// Create Hub client
	hubClient := client.NewHubClient(hubBaseURL)

	// Setup context with cancellation (used for both initial setup and main loop)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch download expiration setting from Hub
	downloadExpirationHours := 1 // Default
	expirationCtx, expirationCancel := context.WithTimeout(ctx, 5*time.Second)
	defer expirationCancel()

	expirationResp, err := hubClient.GetDownloadExpiration(expirationCtx)
	if err != nil {
		log.Printf("failed to fetch download expiration setting, using default (1 hour): %v", err)
	} else {
		downloadExpirationHours = expirationResp.Hours
		log.Printf("download expiration set to %d hours", downloadExpirationHours)
	}

	// Create download server with configurable expiration
	downloadsDir := "/var/lib/xvault/downloads"
	downloadSrv := download.NewServer(downloadListenAddr, downloadsDir, downloadExpirationHours)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(serviceID, hubClient, workerStorage, downloadBaseURL, downloadSrv)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start restore service in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- orch.Run(ctx)
	}()

	// Wait for signal or error
	select {
	case <-sigChan:
		log.Printf("restore service %s received shutdown signal", serviceID)
		cancel()
		if err := orch.Shutdown(context.Background()); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			log.Fatalf("restore service error: %v", err)
		}
	}

	log.Printf("restore service %s stopped", serviceID)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}
