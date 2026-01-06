// Package main is the entry point for the ezQRin API server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	redisClient "github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/container"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"go.uber.org/zap"
)

const (
	shutdownTimeout      = 15 * time.Second
	dbHealthCheckTimeout = 30 * time.Second
	dbRetryInterval      = 5 * time.Second
)

// Application dependencies
var (
	appDB     database.Service
	appLogger *logger.Logger
	appCache  cache.Service
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger first (before other dependencies)
	appLogger, err = logger.New(logger.Config{
		Level:       cfg.Logging.Level,
		Format:      cfg.Logging.Format,
		Environment: cfg.Server.Environment,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize infrastructure (database, cache)
	ctx := context.Background()
	if err := initializeInfrastructure(ctx, cfg); err != nil {
		appLogger.Fatal("failed to initialize infrastructure", zap.Error(err))
	}
	defer cleanup()

	// Initialize container with repositories and use cases
	appContainer := container.NewContainer(cfg, appLogger, appDB, appCache)

	// Setup router with dependencies
	router := api.SetupRouter(&api.RouterDependencies{
		Config:    cfg,
		Logger:    appLogger,
		DB:        appDB,
		Cache:     appCache,
		Container: appContainer,
	})

	// Create and run HTTP server
	srv := createServer(cfg, router)
	runServerWithGracefulShutdown(srv, cfg)
}

// createServer creates and configures the HTTP server.
func createServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// runServerWithGracefulShutdown starts the server and handles graceful shutdown.
func runServerWithGracefulShutdown(srv *http.Server, cfg *config.Config) {
	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Info("starting HTTP server",
			zap.Int("port", cfg.Server.Port),
			zap.String("environment", cfg.Server.Environment),
		)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		appLogger.Fatal("server error", zap.Error(err))

	case sig := <-shutdown:
		appLogger.Info("received shutdown signal, starting graceful shutdown",
			zap.String("signal", sig.String()),
		)

		// Create context with timeout for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			appLogger.Error("error during shutdown", zap.Error(err))
			if err := srv.Close(); err != nil {
				appLogger.Fatal("failed to close server", zap.Error(err))
			}
		}

		appLogger.Info("server stopped gracefully")
	}
}

// initializeInfrastructure initializes basic infrastructure dependencies.
func initializeInfrastructure(ctx context.Context, cfg *config.Config) error {
	appLogger.Info("initializing application infrastructure")

	// Initialize database
	if err := initializeDatabase(ctx, cfg); err != nil {
		return err
	}

	// Initialize Redis cache
	if err := initializeRedis(ctx, cfg); err != nil {
		return err
	}

	return nil
}

// initializeDatabase establishes database connection and waits for it to become healthy.
func initializeDatabase(ctx context.Context, cfg *config.Config) error {
	var err error
	db, err := database.NewPostgresDB(ctx, &cfg.Database, appLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Wait for database to become healthy
	healthCtx, cancel := context.WithTimeout(ctx, dbHealthCheckTimeout)
	defer cancel()

	if err := db.WaitForHealthy(healthCtx, dbRetryInterval); err != nil {
		db.Close()
		return fmt.Errorf("database not healthy: %w", err)
	}

	appDB = db // Assign to interface after health check
	appLogger.Info("database connection established and healthy")
	return nil
}

// initializeRedis establishes Redis connection and verifies health.
func initializeRedis(ctx context.Context, cfg *config.Config) error {
	var err error

	redisConfig := &redisClient.ClientConfig{
		Host:         cfg.Redis.Host,
		Port:         fmt.Sprintf("%d", cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	}

	client, err := redisClient.NewClient(redisConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize redis: %w", err)
	}
	appCache = client // Implicit interface conversion to cache.Service

	// Verify connection
	if err := appCache.Ping(ctx); err != nil {
		appCache.Close()
		return fmt.Errorf("redis not healthy: %w", err)
	}

	appLogger.Info("redis connection established and healthy")
	return nil
}

// cleanup gracefully closes all application dependencies.
func cleanup() {
	if appLogger != nil {
		appLogger.Info("shutting down application infrastructure")
	}

	if appDB != nil {
		appDB.Close()
	}

	if appCache != nil {
		appCache.Close()
	}

	if appLogger != nil {
		// Sync logger before exit to flush any buffered logs
		_ = appLogger.Sync()
		appLogger.Info("cleanup completed")
	}
}
