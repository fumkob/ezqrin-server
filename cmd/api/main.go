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
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize dependencies (database, cache, etc.)
	// This will be implemented in later tasks using a DI container
	if err := initializeDependencies(cfg); err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

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
		log.Printf("Starting server on port %d (environment: %s)", cfg.Server.Port, cfg.Server.Environment)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
			if err := srv.Close(); err != nil {
				log.Fatalf("Failed to close server: %v", err)
			}
		}

		log.Println("Server stopped gracefully")
	}
}

// initializeDependencies initializes all application dependencies.
// This includes database connections, cache, repositories, use cases, etc.
// Implementation will be added in later tasks using dependency injection container.
func initializeDependencies(cfg *config.Config) error {
	// TODO: Initialize database connection (Task 1.4)
	// TODO: Initialize Redis cache
	// TODO: Initialize repositories
	// TODO: Initialize use cases
	// TODO: Initialize handlers

	log.Println("Dependencies initialized (placeholder)")
	return nil
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
