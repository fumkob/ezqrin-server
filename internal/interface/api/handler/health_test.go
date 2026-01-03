package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHealthHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HealthHandler Suite")
}

var _ = Describe("HealthHandler", func() {
	var (
		router        *gin.Engine
		log           *logger.Logger
		mockDB        *mockDBHealthChecker
		mockRedis     *mockRedisHealthChecker
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

		// Create mocks
		mockDB = &mockDBHealthChecker{healthy: true}
		mockRedis = &mockRedisHealthChecker{shouldFail: false}

		// Create health handler with both DB and Redis health checkers
		healthHandler = handler.NewHealthHandler(mockDB, mockRedis, log)

		// Setup router with RequestID middleware for header testing
		router = gin.New()
		router.Use(middleware.RequestID())
	})

	Describe("GetHealth", func() {
		When("checking basic health endpoint", func() {
			It("should return 200 OK with OpenAPI-compliant response", func() {
				router.GET("/health", healthHandler.GetHealth)

				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"healthy"`))

				// Verify no success wrapper
				Expect(w.Body.String()).ToNot(ContainSubstring(`"success"`))

				// Verify request_id is NOT in JSON body (OpenAPI compliance)
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))

				// Verify request ID is in header instead
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})
	})

	Describe("GetHealthReady", func() {
		When("database and Redis are healthy", func() {
			It("should return 200 OK with OpenAPI-compliant response", func() {
				router.GET("/health/ready", healthHandler.GetHealthReady)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ready"`))

				// Response structure now has "checks" object
				Expect(w.Body.String()).To(ContainSubstring(`"checks"`))
				Expect(w.Body.String()).To(ContainSubstring(`"database":"ok"`))
				Expect(w.Body.String()).To(ContainSubstring(`"redis":"ok"`))

				// Verify request_id is NOT in JSON body
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))

				// Verify request ID in header
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})

		When("database is unhealthy", func() {
			It("should return 503 Service Unavailable with RFC 9457 Problem Details", func() {
				mockDB.healthy = false
				router.GET("/health/ready", healthHandler.GetHealthReady)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))

				// Verify RFC 9457 structure
				Expect(w.Body.String()).To(ContainSubstring(`"type"`))
				Expect(w.Body.String()).To(ContainSubstring(`problems/service-unavailable`))
				Expect(w.Body.String()).To(ContainSubstring(`"title"`))
				Expect(w.Body.String()).To(ContainSubstring(`"status":503`))
				Expect(w.Body.String()).To(ContainSubstring(`"detail"`))
				Expect(w.Body.String()).To(ContainSubstring(`"instance"`))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"SERVICE_UNAVAILABLE"`))

				// Verify no success wrapper
				Expect(w.Body.String()).ToNot(ContainSubstring(`"success"`))

				// Verify request_id is NOT in JSON body
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))

				// Verify request ID in header
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})

		When("Redis is unhealthy", func() {
			It("should return 503 Service Unavailable with RFC 9457 Problem Details", func() {
				mockRedis.shouldFail = true
				mockRedis.err = errors.New("redis connection failed")
				router.GET("/health/ready", healthHandler.GetHealthReady)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))

				// Verify RFC 9457 structure
				Expect(w.Body.String()).To(ContainSubstring(`"type"`))
				Expect(w.Body.String()).To(ContainSubstring(`problems/service-unavailable`))
				Expect(w.Body.String()).To(ContainSubstring(`"status":503`))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"SERVICE_UNAVAILABLE"`))

				// Verify request ID in header
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})

		When("both database and Redis are unhealthy", func() {
			It("should return 503 with RFC 9457 Problem Details", func() {
				mockDB.healthy = false
				mockRedis.shouldFail = true
				mockRedis.err = errors.New("redis connection failed")
				router.GET("/health/ready", healthHandler.GetHealthReady)

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))

				// Verify RFC 9457 structure
				Expect(w.Body.String()).To(ContainSubstring(`"type"`))
				Expect(w.Body.String()).To(ContainSubstring(`problems/service-unavailable`))
				Expect(w.Body.String()).To(ContainSubstring(`"status":503`))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"SERVICE_UNAVAILABLE"`))

				// Verify request ID in header
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})
	})

	Describe("GetHealthLive", func() {
		When("checking liveness endpoint", func() {
			It("should return 200 OK with OpenAPI-compliant response", func() {
				router.GET("/health/live", healthHandler.GetHealthLive)

				req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"alive"`))

				// Verify no success wrapper
				Expect(w.Body.String()).ToNot(ContainSubstring(`"success"`))

				// Verify request_id is NOT in JSON body
				Expect(w.Body.String()).ToNot(ContainSubstring(`"request_id"`))

				// Verify request ID in header
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})
	})
})
