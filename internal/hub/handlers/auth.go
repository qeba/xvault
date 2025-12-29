package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"xvault/internal/hub/middleware"
	"xvault/internal/hub/service"
)

// AuthHandlers wraps the auth service for HTTP handlers
type AuthHandlers struct {
	authService *service.AuthService
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(authService *service.AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

// Helper to get IP address and user agent
func getRequestMetadata(c *fiber.Ctx) (ipAddress, userAgent string) {
	ipAddress = c.IP()
	userAgent = c.Get("User-Agent")
	if ipAddress == "" {
		ipAddress = c.Get("X-Forwarded-For")
		if ipAddress == "" {
			ipAddress = c.Get("X-Real-IP")
		}
	}
	return ipAddress, userAgent
}

// HandleRegister handles POST /api/v1/auth/register
func (h *AuthHandlers) HandleRegister(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return sendError(c, fiber.StatusBadRequest, nil, "Name, email, and password are required")
	}

	// Validate password length
	if len(req.Password) < 8 {
		return sendError(c, fiber.StatusBadRequest, nil, "Password must be at least 8 characters")
	}

	ipAddress, userAgent := getRequestMetadata(c)

	resp, err := h.authService.Register(ctx, req, ipAddress, userAgent)
	if err != nil {
		log.Printf("failed to register user: %v", err)
		if err.Error() == "user with this email already exists" {
			return sendError(c, fiber.StatusConflict, err, "Email already registered")
		}
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to register user")
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// HandleLogin handles POST /api/v1/auth/login
func (h *AuthHandlers) HandleLogin(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return sendError(c, fiber.StatusBadRequest, nil, "Email and password are required")
	}

	ipAddress, userAgent := getRequestMetadata(c)

	resp, err := h.authService.Login(ctx, req, ipAddress, userAgent)
	if err != nil {
		log.Printf("failed to login user: %v", err)
		if err.Error() == "invalid email or password" {
			return sendError(c, fiber.StatusUnauthorized, err, "Invalid email or password")
		}
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to login")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// HandleRefresh handles POST /api/v1/auth/refresh
func (h *AuthHandlers) HandleRefresh(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	var req service.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return sendError(c, fiber.StatusBadRequest, err, "Invalid request body")
	}

	// Validate input
	if req.RefreshToken == "" {
		return sendError(c, fiber.StatusBadRequest, nil, "Refresh token is required")
	}

	ipAddress, userAgent := getRequestMetadata(c)

	resp, err := h.authService.Refresh(ctx, req, ipAddress, userAgent)
	if err != nil {
		log.Printf("failed to refresh token: %v", err)
		if err.Error() == "refresh token not found or expired" || err.Error() == "refresh token has been revoked" {
			return sendError(c, fiber.StatusUnauthorized, err, "Invalid or expired refresh token")
		}
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to refresh token")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// HandleLogout handles POST /api/v1/auth/logout
func (h *AuthHandlers) HandleLogout(c *fiber.Ctx) error {
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()

	// Get access token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return sendError(c, fiber.StatusBadRequest, nil, "Authorization header required")
	}

	// Extract Bearer token
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return sendError(c, fiber.StatusBadRequest, nil, "Invalid authorization header")
	}

	accessToken := authHeader[7:]

	// Get refresh token from request body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		// Refresh token is optional for logout
		req.RefreshToken = ""
	}

	err := h.authService.Logout(ctx, accessToken, req.RefreshToken)
	if err != nil {
		log.Printf("failed to logout: %v", err)
		return sendError(c, fiber.StatusInternalServerError, err, "Failed to logout")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Successfully logged out",
	})
}

// HandleMe handles GET /api/v1/auth/me
func (h *AuthHandlers) HandleMe(c *fiber.Ctx) error {
	// Get user info from context (set by JWT middleware)
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Not authenticated")
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Not authenticated")
	}

	email, err := middleware.GetEmail(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Not authenticated")
	}

	role, err := middleware.GetRole(c)
	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, err, "Not authenticated")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user_id":   userID,
		"tenant_id": tenantID,
		"email":     email,
		"role":      role,
	})
}
