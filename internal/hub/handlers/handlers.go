package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	middlewarepkg "xvault/internal/hub/middleware"
	"xvault/internal/hub/repository"
	"xvault/internal/hub/service"
	"xvault/pkg/types"

	"github.com/gofiber/fiber/v2"
)

// Handlers wraps the service for HTTP handlers
type Handlers struct {
	service *service.Service
}

// NewHandlers creates a new handlers instance
func NewHandlers(service *service.Service) *Handlers {
	return &Handlers{service: service}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// sendError sends a JSON error response
func sendError(c *fiber.Ctx, status int, err error, message string) error {
	resp := ErrorResponse{
		Error: message,
	}
	if err != nil {
		resp.Details = err.Error()
	}
	return c.Status(status).JSON(resp)
}

// contextWithTimeout creates a context with timeout
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Tenant handlersdv

// HandleCreateTenant handles POST /api/v1/tenants
func (h *Handlers) HandleCreateTenant(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.Name == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("name is required"), "Validation failed")
	}

	resp, err := h.service.CreateTenant(ctx, req)
	if err != nil {
		log.Printf("failed to create tenant: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create tenant")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionCreateTenant, service.AuditTargetTenant, resp.Tenant.ID, resp.Tenant.Name, &resp.Tenant.ID, nil)

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// HandleGetTenant handles GET /api/v1/tenants/:id
func (h *Handlers) HandleGetTenant(c *fiber.Ctx) error {
	// For now, return not implemented
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"message": "Not implemented yet"})
}

// Credential handlers

// HandleCreateCredential handles POST /api/v1/credentials
func (h *Handlers) HandleCreateCredential(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateCredentialRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.TenantID == "" || req.Kind == "" || req.Plaintext == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("missing required fields"), "Validation failed")
	}

	cred, err := h.service.CreateCredential(ctx, req)
	if err != nil {
		log.Printf("failed to create credential: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create credential")
	}

	return c.Status(fiber.StatusCreated).JSON(cred)
}

// Source handlers

// HandleCreateSource handles POST /api/v1/sources
func (h *Handlers) HandleCreateSource(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateSourceRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.TenantID == "" || req.Type == "" || req.Name == "" || req.CredentialID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("missing required fields"), "Validation failed")
	}

	source, err := h.service.CreateSource(ctx, req)
	if err != nil {
		log.Printf("failed to create source: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create source")
	}

	return c.Status(fiber.StatusCreated).JSON(source)
}

// HandleListSources handles GET /api/v1/sources
func (h *Handlers) HandleListSources(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	// Get tenant_id from JWT context
	tenantID, err := middlewarepkg.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Authentication required")
	}

	sources, err := h.service.ListSources(ctx, tenantID)
	if err != nil {
		log.Printf("failed to list sources: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list sources")
	}

	return c.JSON(fiber.Map{"sources": sources})
}

// HandleGetSource handles GET /api/v1/sources/:id
func (h *Handlers) HandleGetSource(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	sourceID := c.Params("id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	source, err := h.service.GetSource(ctx, sourceID)
	if err != nil {
		log.Printf("failed to get source: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Source not found")
	}

	return c.JSON(source)
}

// Job handlers

// HandleEnqueueBackupJob handles POST /api/v1/jobs
func (h *Handlers) HandleEnqueueBackupJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.EnqueueBackupJobRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.SourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	// Get tenant_id from JWT context
	tenantID, err := middlewarepkg.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Authentication required")
	}

	job, err := h.service.EnqueueBackupJob(ctx, tenantID, req)
	if err != nil {
		log.Printf("failed to enqueue job: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to enqueue job")
	}

	return c.Status(fiber.StatusCreated).JSON(job)
}

// Internal/Worker handlers

// HandleClaimJob handles POST /internal/jobs/claim
func (h *Handlers) HandleClaimJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	var req types.JobClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.WorkerID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("worker_id is required"), "Validation failed")
	}

	resp, err := h.service.ClaimJob(ctx, req)
	if err != nil {
		// Don't log "no jobs available" - it's expected when queue is empty
		if err != sql.ErrNoRows {
			log.Printf("failed to claim job: %v", err)
		}
		return sendError(c, fiber.StatusNotFound, err, "No jobs available")
	}

	return c.JSON(resp)
}

// HandleCompleteJob handles POST /internal/jobs/:id/complete
func (h *Handlers) HandleCompleteJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	jobID := c.Params("id")
	if jobID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("job_id is required"), "Validation failed")
	}

	var req types.JobCompleteRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.WorkerID == "" || req.Status == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("worker_id and status are required"), "Validation failed")
	}

	if err := h.service.CompleteJob(ctx, jobID, req); err != nil {
		log.Printf("failed to complete job: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to complete job")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
}

// HandleGetCredential handles GET /internal/credentials/:id
func (h *Handlers) HandleGetCredential(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	credentialID := c.Params("id")
	if credentialID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("credential_id is required"), "Validation failed")
	}

	cred, err := h.service.GetCredentialForWorker(ctx, credentialID)
	if err != nil {
		log.Printf("failed to get credential: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Credential not found")
	}

	return c.JSON(cred)
}

// HandleGetTenantPublicKey handles GET /internal/tenants/:id/public-key
func (h *Handlers) HandleGetTenantPublicKey(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	tenantID := c.Params("id")
	if tenantID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id is required"), "Validation failed")
	}

	key, err := h.service.GetTenantPublicKeyForWorker(ctx, tenantID)
	if err != nil {
		log.Printf("failed to get tenant key: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Tenant key not found")
	}

	// Return only the public key
	return c.JSON(fiber.Map{
		"tenant_id":  key.TenantID,
		"public_key": key.PublicKey,
		"algorithm":  key.Algorithm,
	})
}

// HandleGetTenantPrivateKey handles GET /internal/tenants/:id/private-key
// This returns the DECRYPTED private key for restore operations
// For v0, this is only accessible via internal API (worker to hub)
func (h *Handlers) HandleGetTenantPrivateKey(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	tenantID := c.Params("id")
	if tenantID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id is required"), "Validation failed")
	}

	privateKey, err := h.service.GetTenantPrivateKeyForWorker(ctx, tenantID)
	if err != nil {
		log.Printf("failed to get tenant private key: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Tenant key not found")
	}

	// Return the decrypted private key
	return c.JSON(fiber.Map{
		"tenant_id":   tenantID,
		"private_key": privateKey,
	})
}

// HandleRegisterWorker handles POST /internal/workers/register
func (h *Handlers) HandleRegisterWorker(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req types.WorkerRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.WorkerID == "" || req.Name == "" || req.StorageBasePath == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("worker_id, name, and storage_base_path are required"), "Validation failed")
	}

	if req.Capabilities == nil {
		req.Capabilities = make(map[string]any)
	}

	worker, err := h.service.RegisterWorker(ctx, req)
	if err != nil {
		log.Printf("failed to register worker: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to register worker")
	}

	return c.Status(fiber.StatusCreated).JSON(worker)
}

// HandleWorkerHeartbeat handles POST /internal/workers/heartbeat
func (h *Handlers) HandleWorkerHeartbeat(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req types.WorkerHeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.WorkerID == "" || req.Status == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("worker_id and status are required"), "Validation failed")
	}

	if err := h.service.WorkerHeartbeat(ctx, req); err != nil {
		log.Printf("failed to update heartbeat: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update heartbeat")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
}

// Snapshot handlers

// HandleListSnapshots handles GET /api/v1/snapshots
func (h *Handlers) HandleListSnapshots(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	// Get tenant_id from JWT context
	tenantID, err := middlewarepkg.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Authentication required")
	}

	sourceID := c.Query("source_id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	limit := 50 // Default limit

	snapshots, err := h.service.ListSnapshots(ctx, tenantID, sourceID, limit)
	if err != nil {
		log.Printf("failed to list snapshots: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list snapshots")
	}

	return c.JSON(snapshots)
}

// HandleGetSnapshot handles GET /api/v1/snapshots/:id
func (h *Handlers) HandleGetSnapshot(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	snapshotID := c.Params("id")
	if snapshotID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("snapshot_id is required"), "Validation failed")
	}

	snapshot, err := h.service.GetSnapshot(ctx, snapshotID)
	if err != nil {
		log.Printf("failed to get snapshot: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Snapshot not found")
	}

	return c.JSON(snapshot)
}

// Admin / Retention handlers

// HandleRunRetentionForAllSources handles POST /api/v1/admin/retention/run
// This manually triggers retention evaluation for all enabled sources
func (h *Handlers) HandleRunRetentionForAllSources(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(30 * time.Second)
	defer cancel()

	results, err := h.service.RunRetentionEvaluationForAllSources(ctx)
	if err != nil {
		log.Printf("failed to run retention evaluation: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to run retention evaluation")
	}

	// Build summary
	var totalSnapshots, totalToKeep, totalToDelete, totalJobsEnqueued int
	for _, result := range results {
		if result.EvaluationResult != nil {
			totalSnapshots += len(result.EvaluationResult.SnapshotsToKeep) + len(result.EvaluationResult.SnapshotsToDelete)
			totalToKeep += len(result.EvaluationResult.SnapshotsToKeep)
			totalToDelete += len(result.EvaluationResult.SnapshotsToDelete)
		}
		totalJobsEnqueued += result.JobsEnqueued
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"results": results,
		"summary": fiber.Map{
			"sources_evaluated": len(results),
			"total_snapshots":   totalSnapshots,
			"total_to_keep":     totalToKeep,
			"total_to_delete":   totalToDelete,
			"jobs_enqueued":     totalJobsEnqueued,
		},
	})
}

// HandleRunRetentionForSource handles POST /api/v1/admin/retention/run/:source_id
// This manually triggers retention evaluation for a specific source
func (h *Handlers) HandleRunRetentionForSource(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	sourceID := c.Params("id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	result, err := h.service.RunRetentionEvaluationForSource(ctx, sourceID)
	if err != nil {
		log.Printf("failed to run retention evaluation for source %s: %v", sourceID, err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to run retention evaluation")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"source_id": sourceID,
		"result":    result,
	})
}

// HandleGetSourceRetentionPolicy handles GET /api/v1/sources/:id/retention
// Returns the retention policy for a specific source
func (h *Handlers) HandleGetSourceRetentionPolicy(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	sourceID := c.Params("id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	schedule, err := h.service.GetScheduleForSource(ctx, sourceID)
	if err != nil {
		log.Printf("failed to get schedule for source %s: %v", sourceID, err)
		return sendError(c, fiber.StatusNotFound, err, "Schedule not found for this source")
	}

	// Parse and return the retention policy
	var policy types.RetentionPolicy
	if len(schedule.RetentionPolicy) > 0 {
		if err := json.Unmarshal(schedule.RetentionPolicy, &policy); err != nil {
			return sendError(c, fiber.StatusInternalServerError, err, "Failed to parse retention policy")
		}
	}

	return c.JSON(fiber.Map{
		"source_id":        sourceID,
		"schedule_id":      schedule.ID,
		"retention_policy": policy,
	})
}

// HandleUpdateSourceRetentionPolicy handles PUT /api/v1/sources/:id/retention
// Updates the retention policy for a specific source
func (h *Handlers) HandleUpdateSourceRetentionPolicy(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	sourceID := c.Params("id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	var req service.UpdateSourceRetentionPolicyRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	schedule, err := h.service.UpdateSourceRetentionPolicy(ctx, sourceID, req)
	if err != nil {
		log.Printf("failed to update retention policy for source %s: %v", sourceID, err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update retention policy")
	}

	// Parse and return the updated retention policy
	var policy types.RetentionPolicy
	json.Unmarshal(schedule.RetentionPolicy, &policy)

	return c.JSON(fiber.Map{
		"source_id":        sourceID,
		"schedule_id":      schedule.ID,
		"retention_policy": policy,
	})
}

// Schedule handlers

// HandleCreateSchedule handles POST /api/v1/schedules
func (h *Handlers) HandleCreateSchedule(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.TenantID == "" || req.SourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id and source_id are required"), "Validation failed")
	}

	schedule, err := h.service.CreateSchedule(ctx, req)
	if err != nil {
		log.Printf("failed to create schedule: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create schedule")
	}

	return c.Status(fiber.StatusCreated).JSON(schedule)
}

// HandleGetSchedule handles GET /api/v1/schedules/:id
func (h *Handlers) HandleGetSchedule(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	scheduleID := c.Params("id")
	if scheduleID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("schedule_id is required"), "Validation failed")
	}

	schedule, err := h.service.GetSchedule(ctx, scheduleID)
	if err != nil {
		log.Printf("failed to get schedule: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Schedule not found")
	}

	return c.JSON(schedule)
}

// HandleUpdateSchedule handles PUT /api/v1/schedules/:id
func (h *Handlers) HandleUpdateSchedule(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	scheduleID := c.Params("id")
	if scheduleID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("schedule_id is required"), "Validation failed")
	}

	var req service.UpdateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	schedule, err := h.service.UpdateSchedule(ctx, scheduleID, req)
	if err != nil {
		log.Printf("failed to update schedule: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update schedule")
	}

	return c.JSON(schedule)
}

// HandleListSchedules handles GET /api/v1/schedules
func (h *Handlers) HandleListSchedules(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	// Get tenant_id from JWT context
	tenantID, err := middlewarepkg.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Authentication required")
	}

	schedules, err := h.service.ListSchedules(ctx, tenantID)
	if err != nil {
		log.Printf("failed to list schedules: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list schedules")
	}

	return c.JSON(fiber.Map{"schedules": schedules})
}

// Restore handlers

// HandleEnqueueRestoreJob handles POST /api/v1/snapshots/:id/restore
// This creates a restore job and returns a job ID
func (h *Handlers) HandleEnqueueRestoreJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	snapshotID := c.Params("id")
	if snapshotID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("snapshot_id is required"), "Validation failed")
	}

	// Get tenant_id from JWT context
	tenantID, err := middlewarepkg.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Authentication required")
	}

	job, err := h.service.EnqueueRestoreJob(ctx, tenantID, snapshotID)
	if err != nil {
		log.Printf("failed to enqueue restore job: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to enqueue restore job")
	}

	return c.Status(fiber.StatusCreated).JSON(job)
}

// Internal/Restore Service handlers

// HandleClaimRestoreJob handles POST /internal/restore-jobs/claim
// Restore service claims the next available restore job
func (h *Handlers) HandleClaimRestoreJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	var req service.RestoreJobClaimRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.ServiceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("service_id is required"), "Validation failed")
	}

	resp, err := h.service.ClaimRestoreJob(ctx, req)
	if err != nil {
		// Don't log "no jobs available" - it's expected when queue is empty
		if err != sql.ErrNoRows {
			log.Printf("failed to claim restore job: %v", err)
		}
		return sendError(c, fiber.StatusNotFound, err, "No restore jobs available")
	}

	return c.JSON(resp)
}

// HandleCompleteRestoreJob handles POST /internal/restore-jobs/:id/complete
// Restore service reports job completion
func (h *Handlers) HandleCompleteRestoreJob(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	jobID := c.Params("id")
	if jobID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("job_id is required"), "Validation failed")
	}

	var req service.RestoreJobCompleteRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.ServiceID == "" || req.Status == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("service_id and status are required"), "Validation failed")
	}

	if err := h.service.CompleteRestoreJob(ctx, jobID, req); err != nil {
		log.Printf("failed to complete restore job: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to complete restore job")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
}

// HandleRegisterRestoreService handles POST /internal/services/register
// Restore service registration
func (h *Handlers) HandleRegisterRestoreService(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.RegisterServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.ServiceID == "" || req.Type == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("service_id and type are required"), "Validation failed")
	}

	service, err := h.service.RegisterRestoreService(ctx, req)
	if err != nil {
		log.Printf("failed to register restore service: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to register service")
	}

	return c.Status(fiber.StatusCreated).JSON(service)
}

// HandleRestoreServiceHeartbeat handles POST /internal/services/heartbeat
func (h *Handlers) HandleRestoreServiceHeartbeat(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.ServiceHeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.ServiceID == "" || req.Status == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("service_id and status are required"), "Validation failed")
	}

	if err := h.service.RestoreServiceHeartbeat(ctx, req); err != nil {
		log.Printf("failed to update heartbeat: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update heartbeat")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
}

// Admin / Settings handlers

// HandleListSettings handles GET /api/v1/admin/settings
// Returns all system settings
func (h *Handlers) HandleListSettings(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	settings, err := h.service.ListSettings(ctx)
	if err != nil {
		log.Printf("failed to list settings: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list settings")
	}

	return c.JSON(fiber.Map{"settings": settings})
}

// HandleGetSetting handles GET /api/v1/admin/settings/:key
// Returns a specific system setting
func (h *Handlers) HandleGetSetting(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	key := c.Params("key")
	if key == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("key is required"), "Validation failed")
	}

	setting, err := h.service.GetSetting(ctx, key)
	if err != nil {
		log.Printf("failed to get setting: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Setting not found")
	}

	return c.JSON(setting)
}

// HandleUpdateSetting handles PUT /api/v1/admin/settings/:key
// Updates a system setting
func (h *Handlers) HandleUpdateSetting(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	key := c.Params("key")
	if key == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("key is required"), "Validation failed")
	}

	var req service.UpdateSettingRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.Value == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("value is required"), "Validation failed")
	}

	setting, err := h.service.UpdateSetting(ctx, key, req)
	if err != nil {
		log.Printf("failed to update setting: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update setting")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionUpdateSetting, service.AuditTargetSetting, setting.Key, setting.Key, nil, nil)

	return c.JSON(setting)
}

// Admin / User handlers

// HandleListUsers handles GET /api/v1/admin/users
// Returns all users (admin only)
func (h *Handlers) HandleListUsers(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	users, err := h.service.ListUsers(ctx)
	if err != nil {
		log.Printf("failed to list users: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list users")
	}

	return c.JSON(fiber.Map{"users": users})
}

// HandleGetUser handles GET /api/v1/admin/users/:id
// Returns a specific user (admin only)
func (h *Handlers) HandleGetUser(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	user, err := h.service.GetUser(ctx, id)
	if err != nil {
		log.Printf("failed to get user: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "User not found")
	}

	return c.JSON(user)
}

// HandleCreateUser handles POST /api/v1/admin/users
// Creates a new user with tenant (admin only)
func (h *Handlers) HandleCreateUser(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateUserAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("email and password are required"), "Validation failed")
	}

	if len(req.Password) < 8 {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("password must be at least 8 characters"), "Validation failed")
	}

	user, err := h.service.CreateUserAdmin(ctx, req)
	if err != nil {
		log.Printf("failed to create user: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create user")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionCreateUser, service.AuditTargetUser, user.ID, user.Email, &user.TenantID, nil)

	return c.Status(fiber.StatusCreated).JSON(user)
}

// HandleUpdateUser handles PUT /api/v1/admin/users/:id
// Updates a user (admin only)
func (h *Handlers) HandleUpdateUser(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	var req service.UpdateUserAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	user, err := h.service.UpdateUserAdmin(ctx, id, req)
	if err != nil {
		log.Printf("failed to update user: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update user")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionUpdateUser, service.AuditTargetUser, user.ID, user.Email, &user.TenantID, nil)

	return c.JSON(user)
}

// HandleDeleteUser handles DELETE /api/v1/admin/users/:id
// Deletes a user (admin only)
func (h *Handlers) HandleDeleteUser(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	// Get user info for audit log before deleting
	user, _ := h.service.GetUser(ctx, id)
	userEmail := id
	var tenantID *string
	if user != nil {
		userEmail = user.Email
		tenantID = &user.TenantID
	}

	if err := h.service.DeleteUser(ctx, id); err != nil {
		log.Printf("failed to delete user: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to delete user")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionDeleteUser, service.AuditTargetUser, id, userEmail, tenantID, nil)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Admin / Tenant handlers

// HandleListTenants handles GET /api/v1/admin/tenants
// Returns all tenants (admin only)
func (h *Handlers) HandleListTenants(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	tenants, err := h.service.ListTenants(ctx)
	if err != nil {
		log.Printf("failed to list tenants: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list tenants")
	}

	return c.JSON(fiber.Map{"tenants": tenants})
}

// HandleGetTenantAdmin handles GET /api/v1/admin/tenants/:id
// Returns a specific tenant (admin only)
func (h *Handlers) HandleGetTenantAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	tenant, err := h.service.GetTenant(ctx, id)
	if err != nil {
		log.Printf("failed to get tenant: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Tenant not found")
	}

	return c.JSON(tenant)
}

// HandleDeleteTenant handles DELETE /api/v1/admin/tenants/:id
// Deletes a tenant and all associated data (admin only)
// This will enqueue delete_snapshot jobs for all snapshots before deleting the tenant
func (h *Handlers) HandleDeleteTenant(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(30 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	// Get tenant name for audit log before deleting
	tenant, _ := h.service.GetTenant(ctx, id)
	tenantName := id
	if tenant != nil {
		tenantName = tenant.Name
	}

	if err := h.service.DeleteTenant(ctx, id); err != nil {
		log.Printf("failed to delete tenant: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to delete tenant")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionDeleteTenant, service.AuditTargetTenant, id, tenantName, &id, nil)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Internal/Settings handlers (for restore service)

// HandleGetDownloadExpiration handles GET /internal/settings/download-expiration
// Returns the download expiration setting in hours (for restore service)
func (h *Handlers) HandleGetDownloadExpiration(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	hours, err := h.service.GetDownloadExpirationHours(ctx)
	if err != nil {
		log.Printf("failed to get download expiration: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to get download expiration")
	}

	return c.JSON(fiber.Map{
		"hours": hours,
	})
}

// Admin / Source handlers

// HandleListSourcesAdmin handles GET /api/v1/admin/sources
// Returns all sources across all tenants (admin only)
func (h *Handlers) HandleListSourcesAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	sources, err := h.service.ListAllSources(ctx)
	if err != nil {
		log.Printf("failed to list all sources: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list sources")
	}

	return c.JSON(fiber.Map{"sources": sources})
}

// HandleGetSourceAdmin handles GET /api/v1/admin/sources/:id
// Returns a specific source (admin only)
func (h *Handlers) HandleGetSourceAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	source, err := h.service.GetSource(ctx, id)
	if err != nil {
		log.Printf("failed to get source: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Source not found")
	}

	return c.JSON(source)
}

// HandleCreateSourceAdmin handles POST /api/v1/admin/sources
// Creates a new source with credential (admin only)
func (h *Handlers) HandleCreateSourceAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateSourceAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.TenantID == "" || req.Type == "" || req.Name == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id, type, and name are required"), "Validation failed")
	}

	if req.Credential == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("credential is required"), "Validation failed")
	}

	source, err := h.service.CreateSourceAdmin(ctx, req)
	if err != nil {
		log.Printf("failed to create source: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create source")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionCreateSource, service.AuditTargetSource, source.ID, source.Name, &source.TenantID, nil)

	return c.Status(fiber.StatusCreated).JSON(source)
}

// HandleUpdateSourceAdmin handles PUT /api/v1/admin/sources/:id
// Updates a source (admin only)
func (h *Handlers) HandleUpdateSourceAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	var req service.UpdateSourceAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	source, err := h.service.UpdateSourceAdmin(ctx, id, req)
	if err != nil {
		log.Printf("failed to update source: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update source")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionUpdateSource, service.AuditTargetSource, id, source.Name, &source.TenantID, nil)

	return c.JSON(source)
}

// HandleDeleteSourceAdmin handles DELETE /api/v1/admin/sources/:id
// Deletes a source (admin only)
func (h *Handlers) HandleDeleteSourceAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	// Get source info before deletion for audit log
	source, _ := h.service.GetSource(ctx, id)
	var sourceName string
	var tenantID *string
	if source != nil {
		sourceName = source.Name
		tenantID = &source.TenantID
	}

	if err := h.service.DeleteSource(ctx, id); err != nil {
		log.Printf("failed to delete source: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to delete source")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionDeleteSource, service.AuditTargetSource, id, sourceName, tenantID, nil)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// HandleTestConnection handles POST /api/v1/admin/sources/test-connection
// Tests connectivity to a source before creating it
func (h *Handlers) HandleTestConnection(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(15 * time.Second)
	defer cancel()

	var req service.TestConnectionRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.Type == "" || req.Host == "" || req.Port == 0 || req.Username == "" || req.Credential == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("type, host, port, username, and credential are required"), "Validation failed")
	}

	result, err := h.service.TestConnection(ctx, req)
	if err != nil {
		log.Printf("failed to test connection: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to test connection")
	}

	return c.JSON(result)
}

// HandleTriggerBackupAdmin handles POST /api/v1/admin/sources/:id/backup
// Triggers a manual backup for a source (admin only)
func (h *Handlers) HandleTriggerBackupAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	// Get source info for audit log
	source, _ := h.service.GetSource(ctx, id)
	var sourceName string
	var tenantID *string
	if source != nil {
		sourceName = source.Name
		tenantID = &source.TenantID
	}

	job, err := h.service.TriggerBackupAdmin(ctx, id)
	if err != nil {
		log.Printf("failed to trigger backup: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to trigger backup")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionTriggerBackup, service.AuditTargetSource, id, sourceName, tenantID, nil)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Backup job created",
		"job":     job,
	})
}

// Admin / Schedule handlers

// HandleListSchedulesAdmin handles GET /api/v1/admin/schedules
// Returns all schedules across all tenants (admin only)
func (h *Handlers) HandleListSchedulesAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	schedules, err := h.service.ListAllSchedulesAdmin(ctx)
	if err != nil {
		log.Printf("failed to list all schedules: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list schedules")
	}

	return c.JSON(fiber.Map{"schedules": schedules})
}

// HandleGetScheduleAdmin handles GET /api/v1/admin/schedules/:id
// Returns a specific schedule (admin only)
func (h *Handlers) HandleGetScheduleAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	schedule, err := h.service.GetSchedule(ctx, id)
	if err != nil {
		log.Printf("failed to get schedule: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Schedule not found")
	}

	return c.JSON(schedule)
}

// HandleCreateScheduleAdmin handles POST /api/v1/admin/schedules
// Creates a new schedule for a source (admin only)
func (h *Handlers) HandleCreateScheduleAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.SourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	schedule, err := h.service.CreateScheduleAdmin(ctx, req)
	if err != nil {
		log.Printf("failed to create schedule: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create schedule")
	}

	// Audit log
	scheduleName := "Schedule for " + req.SourceID[:8]
	h.createAuditEvent(ctx, c, service.AuditActionCreateSchedule, service.AuditTargetSchedule, schedule.ID, scheduleName, &schedule.TenantID, nil)

	return c.Status(fiber.StatusCreated).JSON(schedule)
}

// HandleUpdateScheduleAdmin handles PUT /api/v1/admin/schedules/:id
// Updates a schedule (admin only)
func (h *Handlers) HandleUpdateScheduleAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	var req service.UpdateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	schedule, err := h.service.UpdateSchedule(ctx, id, req)
	if err != nil {
		log.Printf("failed to update schedule: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to update schedule")
	}

	// Audit log
	scheduleName := "Schedule " + id[:8]
	h.createAuditEvent(ctx, c, service.AuditActionUpdateSchedule, service.AuditTargetSchedule, id, scheduleName, &schedule.TenantID, nil)

	return c.JSON(schedule)
}

// HandleDeleteScheduleAdmin handles DELETE /api/v1/admin/schedules/:id
// Deletes a schedule (admin only)
func (h *Handlers) HandleDeleteScheduleAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	// Get schedule info before deletion for audit log
	schedule, _ := h.service.GetSchedule(ctx, id)
	var tenantID *string
	scheduleName := "Schedule " + id[:8]
	if schedule != nil {
		tenantID = &schedule.TenantID
	}

	if err := h.service.DeleteSchedule(ctx, id); err != nil {
		log.Printf("failed to delete schedule: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to delete schedule")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionDeleteSchedule, service.AuditTargetSchedule, id, scheduleName, tenantID, nil)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Admin / Snapshot handlers

// HandleListSnapshotsAdmin handles GET /api/v1/admin/snapshots
// Returns all snapshots and in-progress jobs across all tenants with source/tenant info (admin only)
func (h *Handlers) HandleListSnapshotsAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	limit := c.QueryInt("limit", 200)
	if limit > 500 {
		limit = 500
	}

	// Use the new method that includes both snapshots and jobs
	snapshots, err := h.service.ListAllSnapshotsAndJobsAdmin(ctx, limit)
	if err != nil {
		log.Printf("failed to list all snapshots: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list snapshots")
	}

	return c.JSON(fiber.Map{"snapshots": snapshots})
}

// HandleGetSnapshotAdmin handles GET /api/v1/admin/snapshots/:id
// Returns a specific snapshot (admin only)
func (h *Handlers) HandleGetSnapshotAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("id is required"), "Validation failed")
	}

	snapshot, err := h.service.GetSnapshot(ctx, id)
	if err != nil {
		log.Printf("failed to get snapshot: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Snapshot not found")
	}

	return c.JSON(snapshot)
}

// HandleGetLogsForSnapshot handles GET /api/v1/admin/snapshots/:id/logs
// Returns logs for a specific snapshot (admin only)
func (h *Handlers) HandleGetLogsForSnapshot(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	snapshotID := c.Params("id")
	if snapshotID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("snapshot_id is required"), "Validation failed")
	}

	limit := c.QueryInt("limit", 100)
	if limit > 500 {
		limit = 500
	}

	logs, err := h.service.GetLogsForSnapshot(ctx, snapshotID, limit)
	if err != nil {
		log.Printf("failed to get logs for snapshot: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to get logs")
	}

	return c.JSON(fiber.Map{"logs": logs})
}

// HandleDeleteSnapshotAdmin handles DELETE /api/v1/admin/snapshots/:id
// Deletes a snapshot by enqueuing a delete_snapshot job to clean up worker storage
// and remove the database record (admin only)
func (h *Handlers) HandleDeleteSnapshotAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	snapshotID := c.Params("id")
	if snapshotID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("snapshot_id is required"), "Validation failed")
	}

	// Get snapshot info for audit log
	snapshot, _ := h.service.GetSnapshot(ctx, snapshotID)
	var tenantID *string
	snapshotName := "Snapshot " + snapshotID[:8]
	if snapshot != nil {
		tenantID = &snapshot.TenantID
	}

	// Enqueue delete job (worker will clean up storage, then DB record is removed)
	job, err := h.service.EnqueueDeleteJob(ctx, snapshotID)
	if err != nil {
		log.Printf("failed to enqueue delete job for snapshot: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to enqueue delete job")
	}

	// Audit log
	h.createAuditEvent(ctx, c, service.AuditActionDeleteSnapshot, service.AuditTargetSnapshot, snapshotID, snapshotName, tenantID, nil)

	log.Printf("enqueued delete job %s for snapshot %s", job.ID, snapshotID)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":     "Delete job enqueued successfully",
		"job_id":      job.ID,
		"snapshot_id": snapshotID,
		"worker_id":   job.TargetWorkerID,
	})
}

// HandleGetLogsForSource handles GET /api/v1/admin/sources/:id/logs
// Returns logs for a specific source (admin only)
func (h *Handlers) HandleGetLogsForSource(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	sourceID := c.Params("id")
	if sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("source_id is required"), "Validation failed")
	}

	limit := c.QueryInt("limit", 100)
	if limit > 500 {
		limit = 500
	}

	logs, err := h.service.GetLogsForSource(ctx, sourceID, limit)
	if err != nil {
		log.Printf("failed to get logs for source: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to get logs")
	}

	return c.JSON(fiber.Map{"logs": logs})
}

// HandleListAllLogsAdmin handles GET /api/v1/admin/logs
// Returns all system logs with filtering and pagination (admin only)
func (h *Handlers) HandleListAllLogsAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	// Parse query parameters
	params := service.ListAllLogsAdminParams{
		Limit:      c.QueryInt("limit", 100),
		Offset:     c.QueryInt("offset", 0),
		Level:      c.Query("level", ""),
		Search:     c.Query("search", ""),
		WorkerID:   c.Query("worker_id", ""),
		JobID:      c.Query("job_id", ""),
		SnapshotID: c.Query("snapshot_id", ""),
		SourceID:   c.Query("source_id", ""),
		ScheduleID: c.Query("schedule_id", ""),
	}

	// Validate and cap limit
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	result, err := h.service.ListAllLogsAdmin(ctx, params)
	if err != nil {
		log.Printf("failed to list all logs: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list logs")
	}

	return c.JSON(result)
}

// HandleCreateLog handles POST /internal/logs
// Creates a new log entry (for workers)
func (h *Handlers) HandleCreateLog(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.CreateLogRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	if req.Level == "" || req.Message == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("level and message are required"), "Validation failed")
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[req.Level] {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("invalid log level: %s", req.Level), "Validation failed")
	}

	if err := h.service.CreateLog(ctx, req); err != nil {
		log.Printf("failed to create log: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to create log")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true})
}

// HandleListAuditEventsAdmin handles GET /api/v1/admin/audit
// Returns all audit events with filtering and pagination (admin only)
func (h *Handlers) HandleListAuditEventsAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	// Parse query parameters
	params := service.ListAuditEventsParams{
		Limit:      c.QueryInt("limit", 100),
		Offset:     c.QueryInt("offset", 0),
		Action:     c.Query("action", ""),
		TargetType: c.Query("target_type", ""),
		ActorID:    c.Query("actor_id", ""),
		TenantID:   c.Query("tenant_id", ""),
		Search:     c.Query("search", ""),
	}

	// Validate and cap limit
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	result, err := h.service.ListAuditEventsAdmin(ctx, params)
	if err != nil {
		log.Printf("failed to list audit events: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list audit events")
	}

	return c.JSON(result)
}

// createAuditEvent is a helper to create audit events in handlers
func (h *Handlers) createAuditEvent(ctx context.Context, c *fiber.Ctx, action service.AuditAction, targetType service.AuditTargetType, targetID, targetName string, tenantID *string, details json.RawMessage) {
	// Get user ID from context
	userID, _ := middlewarepkg.GetUserID(c)
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Get IP address
	ip := c.IP()

	req := service.CreateAuditEventRequest{
		TenantID:    tenantID,
		ActorUserID: userIDPtr,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetName:  targetName,
		Details:     details,
		IPAddress:   ip,
	}

	// Create audit event in background - don't block the response
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.service.CreateAuditEvent(bgCtx, req); err != nil {
			log.Printf("failed to create audit event: %v", err)
		}
	}()
}

// HandleListWorkersAdmin handles GET /api/v1/admin/workers
// Returns all workers with their status and system metrics (admin only)
func (h *Handlers) HandleListWorkersAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	workers, err := h.service.ListWorkers(ctx)
	if err != nil {
		log.Printf("failed to list workers: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list workers")
	}

	// Build response with computed health status
	type workerResponse struct {
		ID              string          `json:"id"`
		Name            string          `json:"name"`
		Status          string          `json:"status"`
		Health          string          `json:"health"` // healthy, warning, critical, offline
		Capabilities    json.RawMessage `json:"capabilities"`
		StorageBasePath string          `json:"storage_base_path"`
		SystemMetrics   json.RawMessage `json:"system_metrics,omitempty"`
		LastSeenAt      *time.Time      `json:"last_seen_at,omitempty"`
		CreatedAt       time.Time       `json:"created_at"`
		UpdatedAt       time.Time       `json:"updated_at"`
	}

	result := make([]workerResponse, len(workers))
	for i, w := range workers {
		// Compute health status based on last_seen and metrics
		health := computeWorkerHealth(w)

		result[i] = workerResponse{
			ID:              w.ID,
			Name:            w.Name,
			Status:          w.Status,
			Health:          health,
			Capabilities:    w.Capabilities,
			StorageBasePath: w.StorageBasePath,
			SystemMetrics:   w.SystemMetrics,
			LastSeenAt:      w.LastSeenAt,
			CreatedAt:       w.CreatedAt,
			UpdatedAt:       w.UpdatedAt,
		}
	}

	return c.JSON(fiber.Map{
		"workers": result,
		"total":   len(result),
	})
}

// HandleGetWorkerAdmin handles GET /api/v1/admin/workers/:id
// Returns a single worker with its status and system metrics (admin only)
func (h *Handlers) HandleGetWorkerAdmin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	workerID := c.Params("id")
	if workerID == "" {
		return sendError(c, fiber.StatusBadRequest, nil, "Worker ID is required")
	}

	worker, err := h.service.GetWorker(ctx, workerID)
	if err != nil {
		log.Printf("failed to get worker: %v", err)
		return sendError(c, fiber.StatusNotFound, err, "Worker not found")
	}

	health := computeWorkerHealth(worker)

	return c.JSON(fiber.Map{
		"id":                worker.ID,
		"name":              worker.Name,
		"status":            worker.Status,
		"health":            health,
		"capabilities":      worker.Capabilities,
		"storage_base_path": worker.StorageBasePath,
		"system_metrics":    worker.SystemMetrics,
		"last_seen_at":      worker.LastSeenAt,
		"created_at":        worker.CreatedAt,
		"updated_at":        worker.UpdatedAt,
	})
}

// computeWorkerHealth calculates the health status of a worker based on last_seen and metrics
func computeWorkerHealth(w *repository.Worker) string {
	// Check if worker is offline (no heartbeat in 2 minutes)
	if w.LastSeenAt == nil || time.Since(*w.LastSeenAt) > 2*time.Minute {
		return "offline"
	}

	// If we have system metrics, check them for warning/critical thresholds
	if len(w.SystemMetrics) > 0 {
		var metrics struct {
			CPUPercent    float64 `json:"cpu_percent"`
			MemoryPercent float64 `json:"memory_percent"`
			DiskPercent   float64 `json:"disk_percent"`
		}
		if err := json.Unmarshal(w.SystemMetrics, &metrics); err == nil {
			// Critical if any resource is above 95%
			if metrics.CPUPercent > 95 || metrics.MemoryPercent > 95 || metrics.DiskPercent > 95 {
				return "critical"
			}
			// Warning if any resource is above 80%
			if metrics.CPUPercent > 80 || metrics.MemoryPercent > 80 || metrics.DiskPercent > 80 {
				return "warning"
			}
		}
	}

	return "healthy"
}
