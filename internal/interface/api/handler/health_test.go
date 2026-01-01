package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

func TestHealthHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HealthHandler Suite")
}

// mockHealthChecker implements database.HealthChecker for testing
type mockHealthChecker struct {
	healthy bool
	err     error
}

func (m *mockHealthChecker) CheckHealth(ctx context.Context) (*database.HealthStatus, error) {
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

var _ = Describe("HealthHandler", func() {
	var (
		router        *gin.Engine
		log           *logger.Logger
		mockDB        *mockHealthChecker
		healthHandler *handler.HealthHandler
	)

	BeforeEach(func() {
		// Set Gin to test mode
		gin.SetMode(gin.TestMode)

		// Create test logger
		var err error
		log, err = logger.New(logger.Config{
			Level:       "info",
			Format:      "json",
			Environment: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		// Create mock database
		mockDB = &mockHealthChecker{healthy: true}

		// Create health handler
		healthHandler = handler.NewHealthHandler(mockDB, log)

		// Setup router
		router = gin.New()
	})

	Describe("Health", func() {
		When("checking basic health endpoint", func() {
			It("should return 200 OK with status", func() {
				router.GET("/health", healthHandler.Health)

				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ok"`))
				Expect(w.Body.String()).To(ContainSubstring(`"success":true`))
			})
		})
	})

	Describe("Ready", func() {
		When("database is healthy", func() {
			It("should return 200 OK with ready status", func() {
				router.GET("/health/ready", healthHandler.Ready)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ready"`))
				Expect(w.Body.String()).To(ContainSubstring(`"database"`))
			})
		})

		When("database is unhealthy", func() {
			It("should return 503 Service Unavailable", func() {
				mockDB.healthy = false
				router.GET("/health/ready", healthHandler.Ready)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"not_ready"`))
			})
		})
	})

	Describe("Live", func() {
		When("checking liveness endpoint", func() {
			It("should return 200 OK with alive status", func() {
				router.GET("/health/live", healthHandler.Live)

				req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"alive"`))
				Expect(w.Body.String()).To(ContainSubstring(`"success":true`))
			})
		})
	})
})
