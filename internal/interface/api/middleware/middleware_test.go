package middleware_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Middleware", func() {
	var (
		router *gin.Engine
		log    *logger.Logger
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)

		var err error
		log, err = logger.New(logger.Config{
			Level:       "info",
			Format:      "json",
			Environment: "test",
		})
		Expect(err).ToNot(HaveOccurred())

		router = gin.New()
	})

	Describe("RequestID", func() {
		When("client provides X-Request-ID header", func() {
			It("should use the provided request ID", func() {
				router.Use(middleware.RequestID())
				router.GET("/test", func(c *gin.Context) {
					requestID, exists := c.Get("request_id")
					Expect(exists).To(BeTrue())
					Expect(requestID).To(Equal("test-request-id"))
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Request-ID", "test-request-id")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Header().Get("X-Request-ID")).To(Equal("test-request-id"))
			})
		})

		When("client does not provide X-Request-ID header", func() {
			It("should generate a new UUID request ID", func() {
				router.Use(middleware.RequestID())
				router.GET("/test", func(c *gin.Context) {
					requestID, exists := c.Get("request_id")
					Expect(exists).To(BeTrue())
					Expect(requestID).ToNot(BeEmpty())
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Header().Get("X-Request-ID")).ToNot(BeEmpty())
			})
		})
	})

	Describe("CORS", func() {
		When("request from allowed origin", func() {
			It("should set CORS headers", func() {
				corsConfig := &config.CORSConfig{
					AllowedOrigins:   []string{"http://localhost:3000"},
					AllowedMethods:   []string{"GET", "POST"},
					AllowedHeaders:   []string{"Content-Type"},
					AllowCredentials: true,
				}

				router.Use(middleware.CORS(corsConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Origin", "http://localhost:3000")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Header().Get("Access-Control-Allow-Origin")).To(Equal("http://localhost:3000"))
				Expect(w.Header().Get("Access-Control-Allow-Credentials")).To(Equal("true"))
			})
		})

		When("preflight OPTIONS request", func() {
			It("should return 204 No Content", func() {
				corsConfig := &config.CORSConfig{
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{"GET", "POST"},
					AllowedHeaders: []string{"Content-Type"},
				}

				router.Use(middleware.CORS(corsConfig))
				router.OPTIONS("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodOptions, "/test", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNoContent))
			})
		})
	})

	Describe("Recovery", func() {
		When("handler panics", func() {
			It("should recover and return 500 error", func() {
				router.Use(middleware.Recovery(log))
				router.GET("/panic", func(c *gin.Context) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/panic", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(ContainSubstring("Internal server error"))
			})
		})
	})

	Describe("Logging", func() {
		When("request is processed", func() {
			It("should log request details", func() {
				router.Use(middleware.RequestID())
				router.Use(middleware.Logging(log))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})
	})
})
