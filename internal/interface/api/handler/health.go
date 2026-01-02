// Package handler provides HTTP request handlers for the API.
//
// Health handlers implement the generated.ServerInterface for OpenAPI compliance.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	readinessCheckTimeout = 5 * time.Second
)

// HealthHandler handles health check endpoints.
// Implements generated.ServerInterface for OpenAPI compliance.
type HealthHandler struct {
	db     database.HealthChecker
	logger *logger.Logger
}

// Compile-time check to ensure HealthHandler implements ServerInterface
var _ generated.ServerInterface = (*HealthHandler)(nil)

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db database.HealthChecker, logger *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// GetHealth handles basic health check endpoint (GET /health).
// This always returns 200 OK to indicate the server is running.
// Implements generated.ServerInterface.GetHealth
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response.Data(c, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetHealthReady handles readiness check endpoint (GET /health/ready).
// This checks if the service is ready to accept requests by verifying
// database connectivity. Returns 200 if ready, 503 if not ready.
// Implements generated.ServerInterface.GetHealthReady
func (h *HealthHandler) GetHealthReady(c *gin.Context) {
	// Create context with timeout for database check
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessCheckTimeout)
	defer cancel()

	// Check database health
	dbHealth, err := h.db.CheckHealth(ctx)
	if err != nil || (dbHealth != nil && !dbHealth.Healthy) {
		h.logger.WithContext(ctx).Warn("readiness check failed: database unhealthy")
		response.ProblemWithCode(
			c,
			http.StatusServiceUnavailable,
			apperrors.CodeServiceUnavailable,
			"Service is not ready to accept traffic",
		)
		return
	}

	// Service is ready
	response.Data(c, http.StatusOK, map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"database": "ok",
		},
	})
}

// GetHealthLive handles liveness check endpoint (GET /health/live).
// This checks if the service is alive and responsive.
// Returns 200 if alive, should only fail if the service is completely unresponsive.
// Implements generated.ServerInterface.GetHealthLive
func (h *HealthHandler) GetHealthLive(c *gin.Context) {
	response.Data(c, http.StatusOK, map[string]interface{}{
		"status": "alive",
	})
}
