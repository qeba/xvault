package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"xvault/internal/hub/repository"
	"xvault/pkg/crypto"
)

// JWT claims structures

// AccessTokenClaims represents JWT access token claims
type AccessTokenClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TokenID  string `json:"jti"` // Unique token identifier
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents JWT refresh token claims
type RefreshTokenClaims struct {
	UserID  string `json:"user_id"`
	TokenID string `json:"jti"`
	jwt.RegisteredClaims
}

// AuthConfig holds JWT configuration
type AuthConfig struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	JWTSecret           string
	EncryptionKEK       string // Key-encryption-key for tenant private keys
}

// DefaultAuthConfig returns default auth configuration
func DefaultAuthConfig(jwtSecret string) AuthConfig {
	return AuthConfig{
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour, // 7 days
		JWTSecret:           jwtSecret,
	}
}

// AuthService handles authentication operations
type AuthService struct {
	repo   *repository.Repository
	config AuthConfig
}

// NewAuthService creates a new auth service instance
func NewAuthService(repo *repository.Repository, config AuthConfig) *AuthService {
	return &AuthService{
		repo:   repo,
		config: config,
	}
}

// Helper: hash token for storage
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(h[:])
}

// Helper: generate JWT ID
func generateJTI() string {
	return uuid.New().String()
}

// Helper: hash password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// Helper: verify password
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateAccessToken generates a JWT access token
func (s *AuthService) GenerateAccessToken(userID, tenantID, email, role string) (string, string, error) {
	tokenID := generateJTI()
	now := time.Now()
	expiresAt := now.Add(s.config.AccessTokenDuration)

	claims := AccessTokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "xvault",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateRefreshToken generates a JWT refresh token
func (s *AuthService) GenerateRefreshToken(userID string) (string, string, error) {
	tokenID := generateJTI()
	now := time.Now()
	expiresAt := now.Add(s.config.RefreshTokenDuration)

	claims := RefreshTokenClaims{
		UserID:  userID,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "xvault",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, tokenID, nil
}

// ParseAccessToken parses and validates an access token
func (s *AuthService) ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ParseRefreshToken parses and validates a refresh token
func (s *AuthService) ParseRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	return claims, nil
}

// RegisterRequest is the request to register a new user
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse is the response after successful registration
type RegisterResponse struct {
	User         *repository.User `json:"user"`
	Tenant       *repository.Tenant `json:"tenant"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresAt    time.Time         `json:"expires_at"`
}

// Register registers a new user with a tenant
func (s *AuthService) Register(ctx context.Context, req RegisterRequest, ipAddress, userAgent string) (*RegisterResponse, error) {
	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("name, email, and password are required")
	}

	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create tenant
	tenant, err := s.repo.CreateTenant(ctx, req.Name+"'s Workspace")
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Generate encryption keypair for tenant
	publicKey, privateKey, err := crypto.GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Encrypt private key with platform KEK
	encryptedPrivateKey, err := crypto.EncryptForStorage([]byte(privateKey), s.config.EncryptionKEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Store tenant key
	_, err = s.repo.CreateTenantKey(ctx, tenant.ID, "age-x25519", publicKey, encryptedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to store tenant key: %w", err)
	}

	// Create user
	user, err := s.repo.CreateUser(ctx, tenant.ID, req.Email, hashedPassword, "owner")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, _, err := s.GenerateAccessToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	_, err = s.repo.CreateRefreshToken(ctx, user.ID, hashToken(refreshToken), time.Now().Add(s.config.RefreshTokenDuration), ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &RegisterResponse{
		User:         user,
		Tenant:       tenant,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.config.AccessTokenDuration),
	}, nil
}

// LoginRequest is the request to login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response after successful login
type LoginResponse struct {
	User         *repository.User `json:"user"`
	Tenant       *repository.Tenant `json:"tenant"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresAt    time.Time         `json:"expires_at"`
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if err := VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Get tenant
	tenant, err := s.repo.GetTenant(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Generate tokens
	accessToken, _, err := s.GenerateAccessToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	_, err = s.repo.CreateRefreshToken(ctx, user.ID, hashToken(refreshToken), time.Now().Add(s.config.RefreshTokenDuration), ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &LoginResponse{
		User:         user,
		Tenant:       tenant,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.config.AccessTokenDuration),
	}, nil
}

// RefreshRequest is the request to refresh tokens
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse is the response after successful token refresh
type RefreshResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Refresh refreshes access token using refresh token
func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest, ipAddress, userAgent string) (*RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	// Parse refresh token
	claims, err := s.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if refresh token exists and is not revoked
	refreshToken, err := s.repo.GetRefreshTokenByHash(ctx, hashToken(req.RefreshToken))
	if err != nil {
		return nil, errors.New("refresh token not found or expired")
	}

	if refreshToken.RevokedAt != nil {
		return nil, errors.New("refresh token has been revoked")
	}

	// Get user
	user, err := s.repo.GetUser(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Get tenant
	tenant, err := s.repo.GetTenant(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Generate new tokens
	accessToken, _, err := s.GenerateAccessToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, _, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke old refresh token
	err = s.repo.RevokeRefreshToken(ctx, refreshToken.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Store new refresh token
	_, err = s.repo.CreateRefreshToken(ctx, user.ID, hashToken(newRefreshToken), time.Now().Add(s.config.RefreshTokenDuration), ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(s.config.AccessTokenDuration),
	}, nil
}

// Logout revokes the refresh token and optionally adds access token to blacklist
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Parse access token to get JTI for blacklisting
	accessClaims, err := s.ParseAccessToken(accessToken)
	if err == nil {
		// Add access token JTI to blacklist
		expiresAt := time.Unix(accessClaims.ExpiresAt.Unix(), 0)
		err := s.repo.AddTokenToBlacklist(ctx, accessClaims.TokenID, expiresAt)
		if err != nil {
			return fmt.Errorf("failed to blacklist access token: %w", err)
		}
	}

	// Revoke refresh token
	if refreshToken != "" {
		refreshTokenRecord, err := s.repo.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
		if err == nil {
			err = s.repo.RevokeRefreshToken(ctx, refreshTokenRecord.ID)
			if err != nil {
				return fmt.Errorf("failed to revoke refresh token: %w", err)
			}
		}
	}

	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (s *AuthService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.repo.IsTokenBlacklisted(ctx, jti)
}
