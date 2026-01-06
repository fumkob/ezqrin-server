package handler

import (
	"net/http"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
)

const (
	authHeaderParts = 2 // Bearer token format: "Bearer <token>"
)

// AuthHandler handles authentication endpoints.
// Implements generated.ServerInterface for OpenAPI compliance.
type AuthHandler struct {
	registerUC     *auth.RegisterUseCase
	loginUC        *auth.LoginUseCase
	refreshTokenUC *auth.RefreshTokenUseCase
	logoutUC       *auth.LogoutUseCase
	logger         *logger.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(
	registerUC *auth.RegisterUseCase,
	loginUC *auth.LoginUseCase,
	refreshTokenUC *auth.RefreshTokenUseCase,
	logoutUC *auth.LogoutUseCase,
	logger *logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUC:     registerUC,
		loginUC:        loginUC,
		refreshTokenUC: refreshTokenUC,
		logoutUC:       logoutUC,
		logger:         logger,
	}
}

// RegisterUser handles user registration (POST /auth/register).
// Implements generated.ServerInterface.RegisterUser
func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req generated.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	// Execute use case
	result, err := h.registerUC.Execute(c.Request.Context(), &auth.RegisterRequest{
		Email:    string(req.Email),
		Password: req.Password,
		Name:     req.Name,
		Role:     string(req.Role),
	})
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Map to generated response type
	authResponse := h.toAuthResponse(result)
	response.Data(c, http.StatusCreated, authResponse)
}

// LoginUser handles user login (POST /auth/login).
// Implements generated.ServerInterface.LoginUser
func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req generated.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	// Execute use case
	result, err := h.loginUC.Execute(c.Request.Context(), &auth.LoginRequest{
		Email:    string(req.Email),
		Password: req.Password,
	})
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Map to generated response type
	authResponse := h.toAuthResponse(result)
	response.Data(c, http.StatusOK, authResponse)
}

// RefreshToken handles refresh token rotation (POST /auth/refresh).
// Implements generated.ServerInterface.RefreshToken
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req generated.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(c.Request.Context()).Warn("invalid request body", zap.Error(err))
		response.ProblemFromError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	// Execute use case
	result, err := h.refreshTokenUC.Execute(c.Request.Context(), &auth.RefreshRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Map to generated response type
	authResponse := h.toAuthResponse(result)
	response.Data(c, http.StatusOK, authResponse)
}

// LogoutUser handles user logout (POST /auth/logout).
// Implements generated.ServerInterface.LogoutUser
func (h *AuthHandler) LogoutUser(c *gin.Context) {
	// Extract tokens from request
	accessToken := extractBearerToken(c)

	// Refresh token might be in request body or header
	var refreshToken string
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err == nil {
		refreshToken = req["refresh_token"]
	}

	// Execute use case
	result, err := h.logoutUC.Execute(c.Request.Context(), &auth.LogoutRequest{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		response.ProblemFromError(c, err)
		return
	}

	// Return logout response
	logoutResponse := generated.LogoutResponse{
		Message: result.Message,
	}
	response.Data(c, http.StatusOK, logoutResponse)
}

// toAuthResponse maps use case AuthResponse to generated AuthResponse
func (h *AuthHandler) toAuthResponse(result *auth.AuthResponse) generated.AuthResponse {
	userID := openapi_types.UUID(result.User.ID)
	userEmail := openapi_types.Email(result.User.Email)

	return generated.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		User: generated.User{
			Id:        &userID,
			Email:     userEmail,
			Name:      result.User.Name,
			Role:      generated.UserRole(result.User.Role),
			CreatedAt: &result.User.CreatedAt,
			UpdatedAt: &result.User.UpdatedAt,
		},
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
