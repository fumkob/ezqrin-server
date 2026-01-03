package handler_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health Handler OpenAPI Integration", func() {
	var (
		router        *gin.Engine
		log           *logger.Logger
		mockDB        *mockDBHealthChecker
		mockRedis     *mockRedisHealthChecker
		healthHandler *handler.HealthHandler
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)

		// Create test logger
		var err error
		log, err = logger.New(logger.Config{
			Level:       "info",
			Format:      "json",
			Environment: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		// Create mocks
		mockDB = &mockDBHealthChecker{healthy: true}
		mockRedis = &mockRedisHealthChecker{shouldFail: false}

		// Create health handler with both DB and Redis health checkers
		healthHandler = handler.NewHealthHandler(mockDB, mockRedis, log)

		// Setup router with middleware and use generated RegisterHandlers
		router = gin.New()
		router.Use(middleware.RequestID())

		// Use generated route registration (production code path)
		generated.RegisterHandlers(router, healthHandler)
	})

	When("using generated.RegisterHandlers", func() {
		Context("for GET /health endpoint", func() {
			It("should be registered and return 200 OK with correct response", func() {
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"healthy"`))

				// Verify no success wrapper
				Expect(w.Body.String()).ToNot(ContainSubstring(`"success"`))

				// Verify OpenAPI compliance
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})

		Context("for GET /health/ready endpoint", func() {
			It("should be registered and return 200 OK when database and Redis are healthy", func() {
				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ready"`))
				Expect(w.Body.String()).To(ContainSubstring(`"checks"`))
				Expect(w.Body.String()).To(ContainSubstring(`"database":"ok"`))
				Expect(w.Body.String()).To(ContainSubstring(`"redis":"ok"`))

				// Verify OpenAPI compliance
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})

			It("should return 503 with RFC 9457 Problem Details when database is unhealthy", func() {
				mockDB.healthy = false

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))

				// Verify RFC 9457 structure
				Expect(w.Body.String()).To(ContainSubstring(`"type"`))
				Expect(w.Body.String()).To(ContainSubstring(`problems/service-unavailable`))
				Expect(w.Body.String()).To(ContainSubstring(`"status":503`))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"SERVICE_UNAVAILABLE"`))
			})
		})

		Context("for GET /health/live endpoint", func() {
			It("should be registered and return 200 OK", func() {
				req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"alive"`))

				// Verify no success wrapper
				Expect(w.Body.String()).ToNot(ContainSubstring(`"success"`))

				// Verify OpenAPI compliance
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})
	})

	When("verifying route paths", func() {
		It("should have all three health endpoints registered correctly", func() {
			// Test that all routes return valid responses
			endpoints := []struct {
				path       string
				statusCode int
			}{
				{"/health", http.StatusOK},
				{"/health/ready", http.StatusOK},
				{"/health/live", http.StatusOK},
			}

			for _, endpoint := range endpoints {
				req := httptest.NewRequest(http.MethodGet, endpoint.path, nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(endpoint.statusCode),
					"Expected %s to return %d", endpoint.path, endpoint.statusCode)
			}
		})
	})
})
