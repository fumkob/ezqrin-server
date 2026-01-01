// Package handler provides HTTP request handlers for the API.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     database.HealthChecker
	logger *logger.Logger
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db database.HealthChecker, logger *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// HealthResponse represents the basic health check response
type HealthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// ReadinessResponse represents the readiness check response with database status
type ReadinessResponse struct {
	Status    string                 `json:"status"`
	Database  *database.HealthStatus `json:"database,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// LivenessResponse represents the liveness check response
type LivenessResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Health handles basic health check endpoint (GET /health).
// This always returns 200 OK to indicate the server is running.
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, "Service is healthy")
}

// Ready handles readiness check endpoint (GET /health/ready).
// This checks if the service is ready to accept requests by verifying
// database connectivity. Returns 200 if ready, 503 if not ready.
func (h *HealthHandler) Ready(c *gin.Context) {
	// Create context with timeout for database check
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database health
	dbHealth, err := h.db.CheckHealth(ctx)
	if err != nil || (dbHealth != nil && !dbHealth.Healthy) {
		h.logger.WithContext(ctx).Warn("readiness check failed: database unhealthy")
		response.Success(c, http.StatusServiceUnavailable, ReadinessResponse{
			Status:    "not_ready",
			Database:  dbHealth,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}, "Service is not ready")
		return
	}

	// Service is ready
	response.Success(c, http.StatusOK, ReadinessResponse{
		Status:    "ready",
		Database:  dbHealth,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, "Service is ready")
}

// Live handles liveness check endpoint (GET /health/live).
// This checks if the service is alive and responsive.
// Returns 200 if alive, should only fail if the service is completely unresponsive.
func (h *HealthHandler) Live(c *gin.Context) {
	response.Success(c, http.StatusOK, LivenessResponse{
		Status:    "alive",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, "Service is alive")
}
