package middleware

import (
	"strings"

	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// ContextKeyUserID is the key for storing user ID in gin context
	ContextKeyUserID = "user_id"

	// ContextKeyUserRole is the key for storing user role in gin context
	ContextKeyUserRole = "user_role"

	// authHeaderParts is the number of parts in Bearer token header
	authHeaderParts = 2 // "Bearer <token>"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	blacklistRepo repository.TokenBlacklistRepository
	jwtSecret     string
	logger        *logger.Logger
}

// NewAuthMiddleware creates a new AuthMiddleware
func NewAuthMiddleware(
	blacklistRepo repository.TokenBlacklistRepository,
	jwtSecret string,
	logger *logger.Logger,
) *AuthMiddleware {
	return &AuthMiddleware{
		blacklistRepo: blacklistRepo,
		jwtSecret:     jwtSecret,
		logger:        logger,
	}
}

// Authenticate is a middleware that validates JWT tokens and sets user context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		token := extractBearerToken(c)
		if token == "" {
			m.logger.WithContext(c.Request.Context()).Warn("missing authorization token")
			response.ProblemFromError(c, apperrors.Unauthorized("missing authorization token"))
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := crypto.ParseToken(token, m.jwtSecret)
		if err != nil {
			if err == crypto.ErrExpiredToken {
				m.logger.WithContext(c.Request.Context()).Warn("expired token")
				response.ProblemFromError(c, apperrors.Unauthorized("token has expired"))
				c.Abort()
				return
			}
			m.logger.WithContext(c.Request.Context()).Warn("invalid token", zap.Error(err))
			response.ProblemFromError(c, apperrors.Unauthorized("invalid token"))
			c.Abort()
			return
		}

		// Verify token type is access token
		if claims.TokenType != crypto.TokenTypeAccess {
			m.logger.WithContext(c.Request.Context()).Warn("invalid token type")
			response.ProblemFromError(c, apperrors.Unauthorized("invalid token type"))
			c.Abort()
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := m.blacklistRepo.IsBlacklisted(c.Request.Context(), token)
		if err != nil {
			m.logger.WithContext(c.Request.Context()).Error("failed to check token blacklist", zap.Error(err))
			response.ProblemFromError(c, apperrors.Internal("failed to validate token"))
			c.Abort()
			return
		}
		if isBlacklisted {
			m.logger.WithContext(c.Request.Context()).Warn("blacklisted token used")
			response.ProblemFromError(c, apperrors.Unauthorized("token has been revoked"))
			c.Abort()
			return
		}

		// Set user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserRole, claims.Role)

		// Continue to next handler
		c.Next()
	}
}

// OptionalAuth is a middleware that validates JWT tokens if present but doesn't require them
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		token := extractBearerToken(c)
		if token == "" {
			// No token present, continue without authentication
			c.Next()
			return
		}

		// Parse and validate token
		claims, err := crypto.ParseToken(token, m.jwtSecret)
		if err != nil {
			// Invalid token, but don't abort (optional auth)
			m.logger.WithContext(c.Request.Context()).Debug("invalid optional auth token", zap.Error(err))
			c.Next()
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := m.blacklistRepo.IsBlacklisted(c.Request.Context(), token)
		if err != nil {
			m.logger.WithContext(c.Request.Context()).Warn("failed to check token blacklist", zap.Error(err))
			c.Next()
			return
		}
		if isBlacklisted {
			// Blacklisted token, don't set user context
			c.Next()
			return
		}

		// Set user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUserRole, claims.Role)

		c.Next()
	}
}

// RequireRole is a middleware that checks if the authenticated user has the required role
// Must be used after Authenticate middleware
func (m *AuthMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user role from context (set by Authenticate middleware)
		roleValue, exists := c.Get(ContextKeyUserRole)
		if !exists {
			m.logger.WithContext(c.Request.Context()).Error("role not found in context")
			response.ProblemFromError(c, apperrors.Unauthorized("authentication required"))
			c.Abort()
			return
		}

		userRole, ok := roleValue.(string)
		if !ok {
			m.logger.WithContext(c.Request.Context()).Error("invalid role type in context")
			response.ProblemFromError(c, apperrors.Internal("authentication error"))
			c.Abort()
			return
		}

		// Check if user role is in allowed roles
		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		// User doesn't have required role
		m.logger.WithContext(c.Request.Context()).Warn("insufficient permissions")
		response.ProblemFromError(c, apperrors.Forbidden("insufficient permissions"))
		c.Abort()
	}
}

// extractBearerToken extracts the Bearer token from Authorization header
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Authorization header format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", authHeaderParts)
	if len(parts) != authHeaderParts || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
