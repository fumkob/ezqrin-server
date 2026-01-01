package database

import (
	"context"
	"time"

	"go.uber.org/zap"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
)

// HealthChecker defines the interface for database health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus contains database health and connection pool statistics
type HealthStatus struct {
	Healthy          bool      `json:"healthy"`
	ResponseTime     int64     `json:"response_time_ms"`
	TotalConns       int32     `json:"total_connections"`
	IdleConns        int32     `json:"idle_connections"`
	MaxConns         int32     `json:"max_connections"`
	AcquiredConns    int32     `json:"acquired_connections"`
	ConstructingConn int32     `json:"constructing_connections"`
	Timestamp        time.Time `json:"timestamp"`
	Error            string    `json:"error,omitempty"`
}

// CheckHealth performs a health check on the database connection.
// It pings the database and returns connection pool statistics.
// This method is safe to call frequently for monitoring purposes.
func (db *PostgresDB) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	startTime := time.Now()

	status := &HealthStatus{
		Timestamp: startTime,
		Healthy:   false,
	}

	// Ping database to check connectivity
	if err := db.pool.Ping(ctx); err != nil {
		status.Error = err.Error()
		status.ResponseTime = time.Since(startTime).Milliseconds()
		db.logger.Error("database health check failed",
			zap.Error(err),
			zap.Int64("response_time_ms", status.ResponseTime),
		)
		return status, apperrors.Wrapf(err, "database health check failed")
	}

	// Get connection pool statistics
	stats := db.pool.Stat()
	status.Healthy = true
	status.ResponseTime = time.Since(startTime).Milliseconds()
	status.TotalConns = stats.TotalConns()
	status.IdleConns = stats.IdleConns()
	status.MaxConns = stats.MaxConns()
	status.AcquiredConns = stats.AcquiredConns()
	status.ConstructingConn = stats.ConstructingConns()

	db.logger.Debug("database health check succeeded",
		zap.Int64("response_time_ms", status.ResponseTime),
		zap.Int32("total_conns", status.TotalConns),
		zap.Int32("idle_conns", status.IdleConns),
		zap.Int32("acquired_conns", status.AcquiredConns),
	)

	return status, nil
}

// IsHealthy is a convenience method that returns true if the database is healthy.
// It performs a health check and returns a boolean result, ignoring detailed statistics.
func (db *PostgresDB) IsHealthy(ctx context.Context) bool {
	status, err := db.CheckHealth(ctx)
	if err != nil {
		return false
	}
	return status.Healthy
}

// GetPoolStats returns current connection pool statistics without performing a ping.
// This is a lightweight operation suitable for frequent monitoring.
func (db *PostgresDB) GetPoolStats() *PoolStats {
	stats := db.pool.Stat()
	return &PoolStats{
		TotalConns:       stats.TotalConns(),
		IdleConns:        stats.IdleConns(),
		MaxConns:         stats.MaxConns(),
		AcquiredConns:    stats.AcquiredConns(),
		ConstructingConn: stats.ConstructingConns(),
		Timestamp:        time.Now(),
	}
}

// PoolStats contains lightweight connection pool statistics
type PoolStats struct {
	TotalConns       int32     `json:"total_connections"`
	IdleConns        int32     `json:"idle_connections"`
	MaxConns         int32     `json:"max_connections"`
	AcquiredConns    int32     `json:"acquired_connections"`
	ConstructingConn int32     `json:"constructing_connections"`
	Timestamp        time.Time `json:"timestamp"`
}

// WaitForHealthy blocks until the database becomes healthy or the context is cancelled.
// This is useful during application startup to ensure database connectivity before proceeding.
//
// Example usage:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := db.WaitForHealthy(ctx, 5*time.Second); err != nil {
//		log.Fatal("database not healthy after 30 seconds")
//	}
func (db *PostgresDB) WaitForHealthy(ctx context.Context, retryInterval time.Duration) error {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return apperrors.Wrapf(ctx.Err(), "context cancelled while waiting for database to become healthy")
		case <-ticker.C:
			if db.IsHealthy(ctx) {
				db.logger.Info("database is healthy")
				return nil
			}
			db.logger.Warn("database not healthy, retrying...",
				zap.Duration("retry_interval", retryInterval),
			)
		}
	}
}

// Compile-time check to ensure PostgresDB implements HealthChecker
var _ HealthChecker = (*PostgresDB)(nil)
