// Package api provides HTTP routing and server configuration for the ezQRin API.
package api

import (
	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/container"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	// API_V1_PATH defines the base path for v1 of the API
	API_V1_PATH = "/api/v1"
)

// RouterDependencies holds all dependencies required to setup the router
type RouterDependencies struct {
	Config    *config.Config
	Logger    *logger.Logger
	DB        database.Service     // Interface type for database operations
	Cache     cache.Service        // Interface type for cache operations
	Container *container.Container // Container for all other dependencies
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

	// Initialize authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(
		deps.Container.Repositories.Blacklist,
		deps.Config.JWT.Secret,
		deps.Logger,
	)

	// Initialize all handlers
	combinedHandler := initializeHandlers(deps)

	// Register handlers with authentication middleware that respects OpenAPI security requirements
	// This established the pattern for protecting routes:
	// 1. Define security requirements in OpenAPI spec (e.g., security: [{ bearerAuth: [] }])
	// 2. The generated wrapper will set BearerAuthScopes in the context
	// 3. This middleware will then trigger authentication only for those routes
	options := generated.GinServerOptions{
		Middlewares: []generated.MiddlewareFunc{
			func(c *gin.Context) {
				// Only authenticate if the route has security requirements (BearerAuthScopes is set by the generated wrapper)
				if _, exists := c.Get(generated.BearerAuthScopes); exists {
					authMiddleware.Authenticate()(c)
				}
			},
		},
	}
	generated.RegisterHandlersWithOptions(v1, combinedHandler, options)

	return router
}

// initializeHandlers creates and combines all HTTP handlers
func initializeHandlers(deps *RouterDependencies) *handler.Handler {
	healthHandler := handler.NewHealthHandler(deps.DB, deps.Cache, deps.Logger)

	authUseCases := deps.Container.UseCases.Auth
	authHandler := handler.NewAuthHandler(
		authUseCases.Register,
		authUseCases.Login,
		authUseCases.Refresh,
		authUseCases.Logout,
		deps.Logger,
	)

	eventHandler := handler.NewEventHandler(deps.Container.UseCases.Event, deps.Logger)

	participantHandler := handler.NewParticipantHandler(
		deps.Container.UseCases.Participant,
		deps.Logger,
	)

	checkinHandler := handler.NewCheckinHandler(
		deps.Container.UseCases.Checkin,
		deps.Logger,
	)

	return handler.NewHandler(
		healthHandler,
		authHandler,
		eventHandler,
		participantHandler,
		checkinHandler,
	)
}
