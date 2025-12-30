package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"xvault/internal/hub/service"
)

// JWTMiddleware creates a Fiber middleware for JWT authentication
func JWTMiddleware(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token := parts[1]

		// Parse and validate token
		claims, err := authService.ParseAccessToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Check if token is blacklisted
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		isBlacklisted, err := authService.IsTokenBlacklisted(ctx, claims.TokenID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to verify token status",
			})
		}

		if isBlacklisted {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token has been revoked",
			})
		}

		// Store user info in context for downstream handlers
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("jti", claims.TokenID)

		return c.Next()
	}
}

// OptionalJWTMiddleware creates a Fiber middleware for optional JWT authentication
// It doesn't return 401 if token is missing, but still validates if provided
func OptionalJWTMiddleware(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without auth
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without auth
			return c.Next()
		}

		token := parts[1]

		// Try to parse and validate token
		claims, err := authService.ParseAccessToken(token)
		if err != nil {
			// Invalid token, continue without auth
			return c.Next()
		}

		// Check if token is blacklisted
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		isBlacklisted, err := authService.IsTokenBlacklisted(ctx, claims.TokenID)
		if err != nil || isBlacklisted {
			// Token invalid or blacklisted, continue without auth
			return c.Next()
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("jti", claims.TokenID)

		return c.Next()
	}
}

// RequireAdmin checks if the authenticated user has admin role
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		// Allow both "admin" and "owner" roles for admin access
		if role != "admin" && role != "owner" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}

// GetUserID retrieves the user ID from context
func GetUserID(c *fiber.Ctx) (string, error) {
	userID := c.Locals("user_id")
	if userID == nil {
		return "", errors.New("user not authenticated")
	}
	return userID.(string), nil
}

// GetTenantID retrieves the tenant ID from context
func GetTenantID(c *fiber.Ctx) (string, error) {
	tenantID := c.Locals("tenant_id")
	if tenantID == nil {
		return "", errors.New("user not authenticated")
	}
	return tenantID.(string), nil
}

// GetEmail retrieves the email from context
func GetEmail(c *fiber.Ctx) (string, error) {
	email := c.Locals("email")
	if email == nil {
		return "", errors.New("user not authenticated")
	}
	return email.(string), nil
}

// GetRole retrieves the role from context
func GetRole(c *fiber.Ctx) (string, error) {
	role := c.Locals("role")
	if role == nil {
		return "", errors.New("user not authenticated")
	}
	return role.(string), nil
}

const defaultTimeout = 5 * 1000 * 1000000 // 5 seconds in nanoseconds
