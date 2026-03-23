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
	_ "time/tzdata"

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

// app holds the application's core dependencies.
type app struct {
	db     database.Service
	logger *logger.Logger
	cache  cache.Service
}

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger first (before other dependencies)
	appLogger, err := logger.New(logger.Config{
		Level:       cfg.Logging.Level,
		Format:      cfg.Logging.Format,
		Environment: cfg.Server.Environment,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	a := &app{logger: appLogger}

	// Initialize infrastructure (database, cache)
	ctx := context.Background()
	if err := a.initializeInfrastructure(ctx, cfg); err != nil {
		a.logger.Fatal("failed to initialize infrastructure", zap.Error(err))
	}
	defer a.cleanup()

	// Initialize container with repositories and use cases
	appContainer, err := container.NewContainer(cfg, a.logger, a.db, a.cache)
	if err != nil {
		a.logger.Fatal("failed to initialize container", zap.Error(err))
	}

	// Setup router with dependencies
	router := api.SetupRouter(&api.RouterDependencies{
		Config:    cfg,
		Logger:    a.logger,
		DB:        a.db,
		Cache:     a.cache,
		Container: appContainer,
	})

	// Create and run HTTP server
	srv := createServer(cfg, router)
	a.runServerWithGracefulShutdown(srv, cfg)
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
func (a *app) runServerWithGracefulShutdown(srv *http.Server, cfg *config.Config) {
	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Info("starting HTTP server",
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
		a.logger.Fatal("server error", zap.Error(err))

	case sig := <-shutdown:
		a.logger.Info("received shutdown signal, starting graceful shutdown",
			zap.String("signal", sig.String()),
		)

		// Create context with timeout for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("error during shutdown", zap.Error(err))
			if err := srv.Close(); err != nil {
				a.logger.Fatal("failed to close server", zap.Error(err))
			}
		}

		a.logger.Info("server stopped gracefully")
	}
}

// initializeInfrastructure initializes basic infrastructure dependencies.
func (a *app) initializeInfrastructure(ctx context.Context, cfg *config.Config) error {
	a.logger.Info("initializing application infrastructure")

	// Initialize database
	if err := a.initializeDatabase(ctx, cfg); err != nil {
		return err
	}

	// Initialize Redis cache
	if err := a.initializeRedis(ctx, cfg); err != nil {
		return err
	}

	return nil
}

// initializeDatabase establishes database connection and waits for it to become healthy.
func (a *app) initializeDatabase(ctx context.Context, cfg *config.Config) error {
	db, err := database.NewPostgresDB(ctx, &cfg.Database, a.logger)
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

	a.db = db // Assign to interface after health check
	a.logger.Info("database connection established and healthy")
	return nil
}

// initializeRedis establishes Redis connection and verifies health.
func (a *app) initializeRedis(ctx context.Context, cfg *config.Config) error {
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
	a.cache = client // Implicit interface conversion to cache.Service

	// Verify connection
	if err := a.cache.Ping(ctx); err != nil {
		a.cache.Close()
		return fmt.Errorf("redis not healthy: %w", err)
	}

	a.logger.Info("redis connection established and healthy")
	return nil
}

// cleanup gracefully closes all application dependencies.
func (a *app) cleanup() {
	if a.logger != nil {
		a.logger.Info("shutting down application infrastructure")
	}

	if a.db != nil {
		a.db.Close()
	}

	if a.cache != nil {
		a.cache.Close()
	}

	if a.logger != nil {
		// Sync logger before exit to flush any buffered logs
		_ = a.logger.Sync()
		a.logger.Info("cleanup completed")
	}
}
