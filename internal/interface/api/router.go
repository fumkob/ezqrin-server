// Package api provides HTTP routing and server configuration for the ezQRin API.
package api

import (
	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	// API_V1_PATH defines the base path for v1 of the API
	API_V1_PATH = "/api/v1"
)

// RouterDependencies holds all dependencies required to setup the router
type RouterDependencies struct {
	Config     *config.Config
	Logger     *logger.Logger
	DB         database.Service // Interface type for database operations
	Cache      cache.Service    // Interface type for cache operations
	RegisterUC *auth.RegisterUseCase
	LoginUC    *auth.LoginUseCase
	RefreshUC  *auth.RefreshTokenUseCase
	LogoutUC   *auth.LogoutUseCase
}

// SetupRouter creates and configures the Gin HTTP router with all middleware and routes.
// It applies middleware in the correct order: RequestID → Logging → Recovery → CORS.
// Routes are registered using OpenAPI-generated code for type safety and spec compliance.
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

	// Register OpenAPI-generated routes under the versioned base path
	// This automatically registers all routes defined in the OpenAPI specification
	v1 := router.Group(API_V1_PATH)
	healthHandler := handler.NewHealthHandler(deps.DB, deps.Cache, deps.Logger)
	authHandler := handler.NewAuthHandler(
		deps.RegisterUC,
		deps.LoginUC,
		deps.RefreshUC,
		deps.LogoutUC,
		deps.Logger,
	)
	combinedHandler := handler.NewHandler(healthHandler, authHandler)
	generated.RegisterHandlers(v1, combinedHandler)

	return router
}
