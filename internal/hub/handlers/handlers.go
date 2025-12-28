package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"xvault/internal/hub/service"
	"xvault/pkg/types"
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
		Error:   message,
		Details: err.Error(),
	}
	return c.Status(status).JSON(resp)
}

// contextWithTimeout creates a context with timeout
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Tenant handlers

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

	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id query parameter is required"), "Validation failed")
	}

	sources, err := h.service.ListSources(ctx, tenantID)
	if err != nil {
		log.Printf("failed to list sources: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to list sources")
	}

	return c.JSON(sources)
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

	// For v0, we'll use a default tenant_id from header or query
	// In production, this would come from JWT auth
	tenantID := c.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = c.Query("tenant_id")
	}
	if tenantID == "" {
		return sendError(c, fiber.StatusUnauthorized, fmt.Errorf("tenant_id is required"), "Authentication required")
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
		log.Printf("failed to claim job: %v", err)
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

	tenantID := c.Query("tenant_id")
	sourceID := c.Query("source_id")

	if tenantID == "" || sourceID == "" {
		return sendError(c, fiber.StatusBadRequest, fmt.Errorf("tenant_id and source_id are required"), "Validation failed")
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
