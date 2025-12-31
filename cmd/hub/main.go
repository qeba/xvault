package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	"xvault/internal/hub/database"
	"xvault/internal/hub/handlers"
	middlewarepkg "xvault/internal/hub/middleware"
	"xvault/internal/hub/repository"
	"xvault/internal/hub/service"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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
	jwtSecret := mustGetenv("HUB_JWT_SECRET")

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

	// Initialize auth service and handlers
	authConfig := service.DefaultAuthConfig(jwtSecret)
	authConfig.EncryptionKEK = encryptionKEK
	authService := service.NewAuthService(repo, authConfig)
	authHandlers := handlers.NewAuthHandlers(authService)

	// JWT middleware
	jwtMiddleware := middlewarepkg.JWTMiddleware(authService)

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

	// Auth routes (public - no JWT required)
	auth := api.Group("/auth")
	auth.Post("/register", authHandlers.HandleRegister)
	auth.Post("/login", authHandlers.HandleLogin)
	auth.Post("/refresh", authHandlers.HandleRefresh)
	auth.Post("/logout", jwtMiddleware, authHandlers.HandleLogout)
	auth.Get("/me", jwtMiddleware, authHandlers.HandleMe)

	// Protected routes (JWT required)
	// Tenant routes
	api.Post("/tenants", jwtMiddleware, h.HandleCreateTenant)
	api.Get("/tenants/:id", jwtMiddleware, h.HandleGetTenant)

	// Credential routes
	api.Post("/credentials", jwtMiddleware, h.HandleCreateCredential)

	// Source routes
	api.Post("/sources", jwtMiddleware, h.HandleCreateSource)
	api.Get("/sources", jwtMiddleware, h.HandleListSources)
	api.Get("/sources/:id", jwtMiddleware, h.HandleGetSource)

	// Schedule routes
	api.Post("/schedules", jwtMiddleware, h.HandleCreateSchedule)
	api.Get("/schedules", jwtMiddleware, h.HandleListSchedules)
	api.Get("/schedules/:id", jwtMiddleware, h.HandleGetSchedule)
	api.Put("/schedules/:id", jwtMiddleware, h.HandleUpdateSchedule)

	// Job routes
	api.Post("/jobs", jwtMiddleware, h.HandleEnqueueBackupJob)

	// Snapshot routes
	api.Get("/snapshots", jwtMiddleware, h.HandleListSnapshots)
	api.Get("/snapshots/:id", jwtMiddleware, h.HandleGetSnapshot)

	// Restore routes
	api.Post("/snapshots/:id/restore", jwtMiddleware, h.HandleEnqueueRestoreJob)

	// Source retention policy routes
	api.Get("/sources/:id/retention", jwtMiddleware, h.HandleGetSourceRetentionPolicy)
	api.Put("/sources/:id/retention", jwtMiddleware, h.HandleUpdateSourceRetentionPolicy)

	// Admin routes (JWT + admin role required)
	admin := api.Group("/admin")
	admin.Use(jwtMiddleware)
	admin.Use(middlewarepkg.RequireAdmin())

	// Retention management
	admin.Post("/retention/run", h.HandleRunRetentionForAllSources)
	admin.Post("/retention/run/:id", h.HandleRunRetentionForSource)

	// Settings management
	admin.Get("/settings", h.HandleListSettings)
	admin.Get("/settings/:key", h.HandleGetSetting)
	admin.Put("/settings/:key", h.HandleUpdateSetting)

	// User management (admin only)
	admin.Get("/users", h.HandleListUsers)
	admin.Get("/users/:id", h.HandleGetUser)
	admin.Post("/users", h.HandleCreateUser)
	admin.Put("/users/:id", h.HandleUpdateUser)
	admin.Delete("/users/:id", h.HandleDeleteUser)

	// Tenant management (admin only)
	admin.Get("/tenants", h.HandleListTenants)
	admin.Get("/tenants/:id", h.HandleGetTenantAdmin)
	admin.Delete("/tenants/:id", h.HandleDeleteTenant)

	// Source management (admin only)
	admin.Post("/sources/test-connection", h.HandleTestConnection) // Must be before :id routes
	admin.Get("/sources", h.HandleListSourcesAdmin)
	admin.Get("/sources/:id", h.HandleGetSourceAdmin)
	admin.Post("/sources", h.HandleCreateSourceAdmin)
	admin.Put("/sources/:id", h.HandleUpdateSourceAdmin)
	admin.Delete("/sources/:id", h.HandleDeleteSourceAdmin)
	admin.Post("/sources/:id/backup", h.HandleTriggerBackupAdmin)
	admin.Get("/sources/:id/logs", h.HandleGetLogsForSource)

	// Schedule management (admin only)
	admin.Get("/schedules", h.HandleListSchedulesAdmin)
	admin.Get("/schedules/:id", h.HandleGetScheduleAdmin)
	admin.Post("/schedules", h.HandleCreateScheduleAdmin)
	admin.Put("/schedules/:id", h.HandleUpdateScheduleAdmin)
	admin.Delete("/schedules/:id", h.HandleDeleteScheduleAdmin)

	// Snapshot management (admin only)
	admin.Get("/snapshots", h.HandleListSnapshotsAdmin)
	admin.Get("/snapshots/:id", h.HandleGetSnapshotAdmin)
	admin.Get("/snapshots/:id/logs", h.HandleGetLogsForSnapshot)
	admin.Delete("/snapshots/:id", h.HandleDeleteSnapshotAdmin)

	// Internal/Worker routes
	internal := app.Group("/internal")

	// Job management
	internal.Post("/jobs/claim", h.HandleClaimJob)
	internal.Post("/jobs/:id/complete", h.HandleCompleteJob)

	// Credential fetching
	internal.Get("/credentials/:id", h.HandleGetCredential)

	// Tenant keys
	internal.Get("/tenants/:id/public-key", h.HandleGetTenantPublicKey)
	internal.Get("/tenants/:id/private-key", h.HandleGetTenantPrivateKey)

	// Restore service management
	internal.Post("/restore-jobs/claim", h.HandleClaimRestoreJob)
	internal.Post("/restore-jobs/:id/complete", h.HandleCompleteRestoreJob)
	internal.Post("/services/register", h.HandleRegisterRestoreService)
	internal.Post("/services/heartbeat", h.HandleRestoreServiceHeartbeat)

	// Worker management
	internal.Post("/workers/register", h.HandleRegisterWorker)
	internal.Post("/workers/heartbeat", h.HandleWorkerHeartbeat)

	// Internal settings (for restore service)
	internal.Get("/settings/download-expiration", h.HandleGetDownloadExpiration)

	// Internal logs (for workers)
	internal.Post("/logs", h.HandleCreateLog)

	// Start retention scheduler in background
	retentionIntervalHours := getenv("RETENTION_EVALUATION_INTERVAL_HOURS", "6")
	intervalHours, err := time.ParseDuration(retentionIntervalHours + "h")
	if err != nil {
		log.Printf("invalid RETENTION_EVALUATION_INTERVAL_HOURS, using default 6h: %v", err)
		intervalHours = 6 * time.Hour
	}
	go startRetentionScheduler(svc, intervalHours)

	// Start backup scheduler in background
	backupSchedulerInterval := getenv("BACKUP_SCHEDULER_INTERVAL_SECONDS", "60")
	schedulerInterval, err := time.ParseDuration(backupSchedulerInterval + "s")
	if err != nil {
		log.Printf("invalid BACKUP_SCHEDULER_INTERVAL_SECONDS, using default 60s: %v", err)
		schedulerInterval = 60 * time.Second
	}
	go startBackupScheduler(svc, schedulerInterval)

	log.Printf("hub listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("hub server error: %v", err)
	}
}

// startBackupScheduler runs periodic backup schedule processing
func startBackupScheduler(svc *service.Service, interval time.Duration) {
	log.Printf("starting backup scheduler (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once on startup after a short delay
	time.Sleep(10 * time.Second)
	runBackupScheduler(svc)

	for range ticker.C {
		runBackupScheduler(svc)
	}
}

// runBackupScheduler processes due schedules and enqueues backup jobs
func runBackupScheduler(svc *service.Service) {
	ctx, cancel := contextWithTimeout(1 * time.Minute)
	defer cancel()

	jobsCreated, err := svc.ProcessDueSchedules(ctx)
	if err != nil {
		log.Printf("backup scheduler error: %v", err)
		return
	}

	if jobsCreated > 0 {
		log.Printf("backup scheduler: created %d backup job(s)", jobsCreated)
	}
}

// startRetentionScheduler runs periodic retention evaluation
func startRetentionScheduler(svc *service.Service, interval time.Duration) {
	log.Printf("starting retention scheduler (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once on startup after a short delay
	time.Sleep(30 * time.Second)
	runRetentionEvaluation(svc)

	for range ticker.C {
		runRetentionEvaluation(svc)
	}
}

// runRetentionEvaluation runs retention evaluation and logs results
func runRetentionEvaluation(svc *service.Service) {
	ctx, cancel := contextWithTimeout(5 * time.Minute)
	defer cancel()

	log.Println("running retention evaluation...")
	results, err := svc.RunRetentionEvaluationForAllSources(ctx)
	if err != nil {
		log.Printf("retention evaluation failed: %v", err)
		return
	}

	var totalEvaluated, totalKept, totalDeleted, totalJobsEnqueued int
	for _, r := range results {
		totalEvaluated++
		if r.EvaluationResult != nil {
			totalKept += len(r.EvaluationResult.SnapshotsToKeep)
			totalDeleted += len(r.EvaluationResult.SnapshotsToDelete)
		}
		totalJobsEnqueued += r.JobsEnqueued
	}

	log.Printf("retention evaluation complete: sources=%d, kept=%d, to_delete=%d, jobs_enqueued=%d",
		totalEvaluated, totalKept, totalDeleted, totalJobsEnqueued)
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
