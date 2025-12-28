package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"xvault/internal/worker/orchestrator"
	"xvault/internal/worker/client"
)

func main() {
	workerID := mustGetenv("WORKER_ID")
	hubBaseURL := mustGetenv("HUB_BASE_URL")
	storageBase := getenv("WORKER_STORAGE_BASE", "/var/lib/xvault/backups")

	log.Printf("worker starting: worker_id=%s hub=%s storage=%s", workerID, hubBaseURL, storageBase)

	// Create Hub client
	hubClient := client.NewHubClient(hubBaseURL)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(workerID, hubClient, storageBase)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- orch.Run(ctx)
	}()

	// Wait for signal or error
	select {
	case <-sigChan:
		log.Printf("worker %s received shutdown signal", workerID)
		cancel()
		if err := orch.Shutdown(context.Background()); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			log.Fatalf("worker error: %v", err)
		}
	}

	log.Printf("worker %s stopped", workerID)
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
