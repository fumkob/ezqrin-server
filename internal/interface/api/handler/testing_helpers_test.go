package handler_test

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"go.uber.org/zap"
)

// mockDBHealthChecker implements database.HealthChecker for testing.
type mockDBHealthChecker struct {
	healthy bool
	err     error
}

func (m *mockDBHealthChecker) CheckHealth(ctx context.Context) (*database.HealthStatus, error) {
	if m.err != nil {
		return &database.HealthStatus{
			Healthy: false,
			Error:   m.err.Error(),
		}, m.err
	}
	return &database.HealthStatus{
		Healthy:      m.healthy,
		ResponseTime: 10,
		TotalConns:   5,
		IdleConns:    3,
		MaxConns:     25,
	}, nil
}

// mockRedisHealthChecker implements cache.HealthChecker for testing.
type mockRedisHealthChecker struct {
	shouldFail bool
	err        error
}

func (m *mockRedisHealthChecker) Ping(ctx context.Context) error {
	if m.shouldFail {
		return m.err
	}
	return nil
}

// newTestLogger returns a test logger with no-op logging.
func newTestLogger() *logger.Logger {
	return &logger.Logger{Logger: zap.NewNop()}
}
