package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	// healthCheckTimeout is the timeout for health check operations
	healthCheckTimeout = 2 * time.Second
)

// HealthChecker defines the interface for cache health checks.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// HealthCheck performs a health check on the cache service.
// Returns an error if the cache is unreachable or unhealthy.
func HealthCheck(ctx context.Context, checker HealthChecker) error {
	// Create a timeout context for the health check
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	if err := checker.Ping(ctx); err != nil {
		return fmt.Errorf("cache health check failed: %w", err)
	}

	return nil
}

// HealthStatus represents the health status of the cache.
type HealthStatus struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// GetHealthStatus returns the health status of the cache.
func GetHealthStatus(ctx context.Context, checker HealthChecker) HealthStatus {
	err := HealthCheck(ctx, checker)
	if err != nil {
		return HealthStatus{
			Healthy: false,
			Message: err.Error(),
		}
	}

	return HealthStatus{
		Healthy: true,
		Message: "cache is healthy",
	}
}
