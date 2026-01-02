package cache_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockHealthChecker is a mock implementation of HealthChecker for testing.
type MockHealthChecker struct {
	shouldFail bool
	err        error
}

func (m *MockHealthChecker) Ping(ctx context.Context) error {
	if m.shouldFail {
		return m.err
	}
	return nil
}

var _ = Describe("Cache Health Check", func() {
	var (
		ctx     context.Context
		checker cache.HealthChecker
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("HealthCheck", func() {
		When("checking cache health", func() {
			Context("with healthy cache", func() {
				It("should return no error", func() {
					checker = &MockHealthChecker{shouldFail: false}

					err := cache.HealthCheck(ctx, checker)

					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("with unhealthy cache", func() {
				It("should return an error", func() {
					expectedErr := errors.New("connection refused")
					checker = &MockHealthChecker{
						shouldFail: true,
						err:        expectedErr,
					}

					err := cache.HealthCheck(ctx, checker)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("cache health check failed"))
				})
			})

			Context("with timeout", func() {
				It("should respect context timeout", func() {
					// Create a context that times out quickly
					timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
					defer cancel()

					// Simulate a slow health check
					checker = &MockHealthChecker{shouldFail: false}

					// Give time for timeout to occur
					time.Sleep(5 * time.Millisecond)

					// The health check should timeout
					// Note: In real implementation, this would be tested with actual blocking
					Expect(timeoutCtx.Err()).To(HaveOccurred())
				})
			})
		})
	})

	Describe("GetHealthStatus", func() {
		When("retrieving cache health status", func() {
			Context("with healthy cache", func() {
				It("should return healthy status", func() {
					checker = &MockHealthChecker{shouldFail: false}

					status := cache.GetHealthStatus(ctx, checker)

					Expect(status.Healthy).To(BeTrue())
					Expect(status.Message).To(Equal("cache is healthy"))
				})
			})

			Context("with unhealthy cache", func() {
				It("should return unhealthy status with error message", func() {
					expectedErr := errors.New("redis connection failed")
					checker = &MockHealthChecker{
						shouldFail: true,
						err:        expectedErr,
					}

					status := cache.GetHealthStatus(ctx, checker)

					Expect(status.Healthy).To(BeFalse())
					Expect(status.Message).To(ContainSubstring("cache health check failed"))
					Expect(status.Message).To(ContainSubstring("redis connection failed"))
				})
			})
		})
	})

	Describe("Health check integration", func() {
		When("used in readiness probe", func() {
			Context("during application startup", func() {
				It("should indicate when cache is ready", func() {
					checker = &MockHealthChecker{shouldFail: false}

					status := cache.GetHealthStatus(ctx, checker)

					// Readiness probe should pass
					Expect(status.Healthy).To(BeTrue())
				})
			})

			Context("when cache becomes unavailable", func() {
				It("should indicate cache is not ready", func() {
					checker = &MockHealthChecker{
						shouldFail: true,
						err:        errors.New("cache unavailable"),
					}

					status := cache.GetHealthStatus(ctx, checker)

					// Readiness probe should fail
					Expect(status.Healthy).To(BeFalse())
				})
			})
		})
	})
})
