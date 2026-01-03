package database

import (
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service defines the complete database service interface.
// It combines infrastructure concerns (health checking) with domain repository interfaces.
// This interface should be used at composition boundaries (main.go, router setup).
//
// The Service interface provides:
//   - Health checking for monitoring and readiness probes
//   - Transaction management for atomic operations
//   - Access to the underlying connection pool for repository initialization
//
// Implementation: PostgresDB satisfies this interface.
type Service interface {
	// HealthChecker provides database health status with detailed metrics
	HealthChecker // CheckHealth(ctx) (*HealthStatus, error)

	// Transactor enables transaction management across repository operations
	repository.Transactor // WithTransaction(ctx, fn) error

	// BaseRepository provides basic health checking (simplified version)
	repository.BaseRepository // HealthCheck(ctx) error

	// GetPool returns the underlying connection pool for repository initialization.
	// This is needed during composition to pass the pool to domain repositories.
	GetPool() *pgxpool.Pool

	// Close gracefully shuts down the database connection pool.
	// It waits for all active connections to be released.
	Close()
}
