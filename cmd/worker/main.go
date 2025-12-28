package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	workerID := mustGetenv("WORKER_ID")
	redisURL := mustGetenv("REDIS_URL")
	storageBase := getenv("WORKER_STORAGE_BASE", "/var/lib/xvault/backups")

	log.Printf("worker starting: worker_id=%s storage_base=%s", workerID, storageBase)

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}

	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	log.Printf("worker ready (placeholder). next: job loop + connectors + packaging")
	select {}
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
