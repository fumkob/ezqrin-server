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

	"go.uber.org/zap"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// Application dependencies
var (
	appDB     *database.PostgresDB
	appLogger *logger.Logger
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

	// Initialize dependencies (database, cache, etc.)
	ctx := context.Background()
	if err := initializeDependencies(ctx, cfg); err != nil {
		appLogger.Fatal("failed to initialize dependencies", zap.Error(err))
	}
	defer cleanup()

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      setupRouter(cfg),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

// initializeDependencies initializes all application dependencies.
// This includes database connections, cache, repositories, use cases, etc.
// Note: Logger must be initialized before calling this function.
func initializeDependencies(ctx context.Context, cfg *config.Config) error {
	appLogger.Info("initializing application dependencies")

	// Initialize database
	if err := initializeDatabase(ctx, cfg); err != nil {
		return err
	}

	// TODO: Initialize Redis cache (Task 1.6)
	// TODO: Initialize repositories (future tasks)
	// TODO: Initialize use cases (future tasks)
	// TODO: Initialize handlers (future tasks)

	appLogger.Info("dependencies initialized successfully")
	return nil
}

// initializeDatabase establishes database connection and waits for it to become healthy.
func initializeDatabase(ctx context.Context, cfg *config.Config) error {
	var err error
	appDB, err = database.NewPostgresDB(ctx, &cfg.Database, appLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Wait for database to become healthy (30 second timeout, 5 second retry interval)
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := appDB.WaitForHealthy(healthCtx, 5*time.Second); err != nil {
		appDB.Close()
		return fmt.Errorf("database not healthy: %w", err)
	}

	appLogger.Info("database connection established and healthy")
	return nil
}

// cleanup gracefully closes all application dependencies.
func cleanup() {
	if appLogger != nil {
		appLogger.Info("shutting down application dependencies")
	}

	if appDB != nil {
		appDB.Close()
	}

	if appLogger != nil {
		appLogger.Info("cleanup completed")
	}
}

// setupRouter configures and returns the HTTP router.
// This will be implemented in Task 1.5 with Gin framework.
func setupRouter(cfg *config.Config) http.Handler {
	// TODO: Implement router with Gin (Task 1.5)
	// For now, return a simple handler for testing
	mux := http.NewServeMux()

	// Basic health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","environment":"%s"}`, cfg.Server.Environment)
	})

	return mux
}
