// Package api provides HTTP routing and server configuration for the ezQRin API.
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// RouterDependencies holds all dependencies required to setup the router
type RouterDependencies struct {
	Config *config.Config
	Logger *logger.Logger
	DB     *database.PostgresDB
}

// SetupRouter creates and configures the Gin HTTP router with all middleware and routes.
// It applies middleware in the correct order: RequestID → Logging → Recovery → CORS.
func SetupRouter(deps *RouterDependencies) *gin.Engine {
	// Set Gin mode based on environment
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create router
	router := gin.New()

	// Configure trusted proxies (important for production)
	// In production, you should specify actual proxy IPs
	if err := router.SetTrustedProxies(nil); err != nil {
		deps.Logger.Warn("failed to set trusted proxies")
	}

	// Apply global middleware in order
	router.Use(middleware.RequestID())             // Generate request ID first
	router.Use(middleware.Logging(deps.Logger))    // Log requests with request ID
	router.Use(middleware.Recovery(deps.Logger))   // Recover from panics
	router.Use(middleware.CORS(&deps.Config.CORS)) // Handle CORS

	// Register routes
	registerHealthRoutes(router, deps)
	// TODO: Register API routes (Task 2.3, 3.2, 4.3, 5.2)

	return router
}

// registerHealthRoutes registers health check endpoints
func registerHealthRoutes(router *gin.Engine, deps *RouterDependencies) {
	healthHandler := handler.NewHealthHandler(deps.DB, deps.Logger)

	// Health check routes (no authentication required)
	health := router.Group("/health")
	{
		health.GET("", healthHandler.Health)      // Basic health check
		health.GET("/ready", healthHandler.Ready) // Readiness probe
		health.GET("/live", healthHandler.Live)   // Liveness probe
	}
}
