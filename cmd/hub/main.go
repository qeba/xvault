package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"xvault/internal/hub/database"
	"xvault/internal/hub/handlers"
	"xvault/internal/hub/repository"
	"xvault/internal/hub/service"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
	migrateStatus := flag.Bool("migrate-status", false, "show migration status and exit")
	flag.Parse()

	// Get configuration
	addr := getenv("HUB_LISTEN_ADDR", ":8080")
	databaseURL := mustGetenv("DATABASE_URL")
	redisURL := mustGetenv("REDIS_URL")
	encryptionKEK := mustGetenv("HUB_ENCRYPTION_KEK")

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection established")

	// Connect to Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(opt)

	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}
	log.Println("redis connection established")

	// Handle migration commands
	if *migrateOnly {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		os.Exit(0)
	}

	if *migrateStatus {
		if err := database.MigrateStatus(db); err != nil {
			log.Fatalf("migration status failed: %v", err)
		}
		os.Exit(0)
	}

	// Auto-migrate on startup if flag is set
	if getenv("HUB_AUTO_MIGRATE", "false") == "true" {
		log.Println("running auto-migration on startup...")
		if err := database.Migrate(db); err != nil {
			log.Fatalf("auto-migration failed: %v", err)
		}
	}

	// Initialize repository, service, and handlers
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, rdb, encryptionKEK)
	h := handlers.NewHandlers(svc)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Health check
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("xVault Hub API")
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Tenant routes
	api.Post("/tenants", h.HandleCreateTenant)
	api.Get("/tenants/:id", h.HandleGetTenant)

	// Credential routes
	api.Post("/credentials", h.HandleCreateCredential)

	// Source routes
	api.Post("/sources", h.HandleCreateSource)
	api.Get("/sources", h.HandleListSources)
	api.Get("/sources/:id", h.HandleGetSource)

	// Job routes
	api.Post("/jobs", h.HandleEnqueueBackupJob)

	// Snapshot routes
	api.Get("/snapshots", h.HandleListSnapshots)
	api.Get("/snapshots/:id", h.HandleGetSnapshot)

	// Internal/Worker routes
	internal := app.Group("/internal")

	// Job management
	internal.Post("/jobs/claim", h.HandleClaimJob)
	internal.Post("/jobs/:id/complete", h.HandleCompleteJob)

	// Credential fetching
	internal.Get("/credentials/:id", h.HandleGetCredential)

	// Tenant keys
	internal.Get("/tenants/:id/public-key", h.HandleGetTenantPublicKey)

	// Worker management
	internal.Post("/workers/register", h.HandleRegisterWorker)
	internal.Post("/workers/heartbeat", h.HandleWorkerHeartbeat)

	log.Printf("hub listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("hub server error: %v", err)
	}
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

func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// parseRedisAddr parses a Redis URL to get the host:port
// Supports: redis://localhost:6379/0 or redis://:password@localhost:6379/0
func parseRedisAddr(redisURL string) string {
	// Simple parser - just extract host:port
	// For production, use a proper URL parser
	if len(redisURL) > 9 && redisURL[:9] == "redis://" {
		rest := redisURL[9:]
		// Find the end of host:port (before / or @)
		for i, c := range rest {
			if c == '/' || c == '@' {
				return rest[:i]
			}
		}
		return rest
	}
	return redisURL
}
