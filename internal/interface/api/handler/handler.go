package handler

import (
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
)

// Handler implements the generated.ServerInterface by composing individual handlers.
// This pattern allows us to organize handlers by domain while still satisfying
// the OpenAPI-generated interface.
type Handler struct {
	*HealthHandler
	*AuthHandler
}

// Compile-time check to ensure Handler implements ServerInterface
var _ generated.ServerInterface = (*Handler)(nil)

// NewHandler creates a new combined handler
func NewHandler(health *HealthHandler, auth *AuthHandler) *Handler {
	return &Handler{
		HealthHandler: health,
		AuthHandler:   auth,
	}
}

// Health endpoints are implemented by HealthHandler
// GetHealth, GetHealthLive, GetHealthReady are embedded

// Auth endpoints are implemented by AuthHandler
// RegisterUser, LoginUser, RefreshToken, LogoutUser are embedded

// Placeholder methods for future endpoints - these will be implemented in subsequent tasks
// For now, they return 501 Not Implemented to satisfy the interface

// Example placeholder for future event endpoints:
// These will be removed when actual handlers are implemented

// Note: The generated ServerInterface may include other endpoints from the OpenAPI spec.
// Each endpoint should be implemented in its domain-specific handler and embedded here.
