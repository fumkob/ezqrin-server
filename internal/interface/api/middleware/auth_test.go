package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const testJWTSecret = "test-jwt-secret-for-middleware-tests"

// problemResponse mirrors the detail field from the RFC 9457 response body.
type problemResponse struct {
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

func decodeProblem(body []byte) problemResponse {
	var p problemResponse
	_ = json.Unmarshal(body, &p)
	return p
}

var _ = Describe("AuthMiddleware", func() {
	var (
		ctrl           *gomock.Controller
		mockBlacklist  *mocks.MockTokenBlacklistRepository
		authMiddleware *middleware.AuthMiddleware
		router         *gin.Engine
		nopLogger      *logger.Logger
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)

		ctrl = gomock.NewController(GinkgoT())
		mockBlacklist = mocks.NewMockTokenBlacklistRepository(ctrl)
		nopLogger = &logger.Logger{Logger: zap.NewNop()}
		authMiddleware = middleware.NewAuthMiddleware(mockBlacklist, testJWTSecret, nopLogger)
		router = gin.New()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// ---------------------------------------------------------------------------
	// Helper: build a valid access token for the given userID and role.
	// ---------------------------------------------------------------------------
	newAccessToken := func(userID uuid.UUID, role string, expiry time.Duration) string {
		token, err := crypto.GenerateAccessToken(userID.String(), role, testJWTSecret, expiry)
		Expect(err).NotTo(HaveOccurred())
		return token
	}

	newRefreshToken := func(userID uuid.UUID, role string, expiry time.Duration) string {
		token, err := crypto.GenerateRefreshToken(userID.String(), role, testJWTSecret, "web", expiry)
		Expect(err).NotTo(HaveOccurred())
		return token
	}

	// ---------------------------------------------------------------------------
	// Authenticate()
	// ---------------------------------------------------------------------------
	Describe("Authenticate", func() {
		BeforeEach(func() {
			router.Use(authMiddleware.Authenticate())
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})
		})

		When("no Authorization header is provided", func() {
			It("should return 401 with missing authorization token", func() {
				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("missing authorization token"))
			})
		})

		When("the Authorization header is malformed", func() {
			Context("when the Bearer prefix is missing", func() {
				It("should return 401 with missing authorization token", func() {
					req := httptest.NewRequest(http.MethodGet, "/protected", nil)
					req.Header.Set("Authorization", "Token some-random-value")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusUnauthorized))
					p := decodeProblem(w.Body.Bytes())
					Expect(p.Detail).To(Equal("missing authorization token"))
				})
			})

			Context("when only the word Bearer is provided without a token", func() {
				It("should return 401 with missing authorization token", func() {
					req := httptest.NewRequest(http.MethodGet, "/protected", nil)
					// SplitN with n=2 on "Bearer" yields one part, so extractBearerToken returns "".
					req.Header.Set("Authorization", "Bearer")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusUnauthorized))
					p := decodeProblem(w.Body.Bytes())
					Expect(p.Detail).To(Equal("missing authorization token"))
				})
			})

			Context("when the value is an empty string", func() {
				It("should return 401 with missing authorization token", func() {
					req := httptest.NewRequest(http.MethodGet, "/protected", nil)
					req.Header.Set("Authorization", "")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusUnauthorized))
					p := decodeProblem(w.Body.Bytes())
					Expect(p.Detail).To(Equal("missing authorization token"))
				})
			})
		})

		When("the token is expired", func() {
			It("should return 401 with token has expired", func() {
				expiredToken := newAccessToken(uuid.New(), "attendee", -1*time.Second)

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+expiredToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("token has expired"))
			})
		})

		When("the token is invalid (bad signature)", func() {
			It("should return 401 with invalid token", func() {
				// Sign with a different secret so verification fails.
				wrongSecretToken, err := crypto.GenerateAccessToken(
					uuid.New().String(),
					"attendee",
					"wrong-secret",
					time.Hour,
				)
				Expect(err).NotTo(HaveOccurred())

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("invalid token"))
			})
		})

		When("the token is a refresh token rather than an access token", func() {
			It("should return 401 with invalid token type", func() {
				// The token-type check runs before the blacklist check, so no
				// IsBlacklisted call is expected here.
				refreshToken := newRefreshToken(uuid.New(), "attendee", time.Hour)

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+refreshToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("invalid token type"))
			})
		})

		When("the token is blacklisted", func() {
			It("should return 401 with token has been revoked", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(true, nil)

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("token has been revoked"))
			})
		})

		When("the blacklist check returns an error", func() {
			It("should return 500 with failed to validate token", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, errors.New("redis connection lost"))

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("failed to validate token"))
			})
		})

		When("the token is valid and not blacklisted", func() {
			It("should call the next handler and return 200", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})

			It("should set the user ID in the gin context so GetUserID returns the correct value", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				var capturedID uuid.UUID
				var capturedOK bool

				router2 := gin.New()
				router2.Use(authMiddleware.Authenticate())
				router2.GET("/check", func(c *gin.Context) {
					capturedID, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedOK).To(BeTrue())
				Expect(capturedID).To(Equal(userID))
			})

			It("should set the user role in the gin context so GetUserRole returns the correct value", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				var capturedRole string

				router2 := gin.New()
				router2.Use(authMiddleware.Authenticate())
				router2.GET("/check", func(c *gin.Context) {
					capturedRole = middleware.GetUserRole(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedRole).To(Equal("organizer"))
			})
		})
	})

	// ---------------------------------------------------------------------------
	// OptionalAuth()
	// ---------------------------------------------------------------------------
	Describe("OptionalAuth", func() {
		BeforeEach(func() {
			router.Use(authMiddleware.OptionalAuth())
			router.GET("/optional", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})
		})

		When("no Authorization header is provided", func() {
			It("should continue without aborting and return 200", func() {
				req := httptest.NewRequest(http.MethodGet, "/optional", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})

			It("should not set user ID in context so GetUserID returns uuid.Nil and false", func() {
				var capturedID uuid.UUID
				var capturedOK bool

				router2 := gin.New()
				router2.Use(authMiddleware.OptionalAuth())
				router2.GET("/check", func(c *gin.Context) {
					capturedID, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedOK).To(BeFalse())
				Expect(capturedID).To(Equal(uuid.Nil))
			})
		})

		When("a valid access token is provided", func() {
			It("should set user ID and role in context and return 200", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "attendee", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				var capturedID uuid.UUID
				var capturedRole string

				router2 := gin.New()
				router2.Use(authMiddleware.OptionalAuth())
				router2.GET("/check", func(c *gin.Context) {
					capturedID, _ = middleware.GetUserID(c)
					capturedRole = middleware.GetUserRole(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedID).To(Equal(userID))
				Expect(capturedRole).To(Equal("attendee"))
			})
		})

		When("an invalid token is provided", func() {
			It("should continue without aborting and not set user context", func() {
				var capturedID uuid.UUID
				var capturedOK bool

				router2 := gin.New()
				router2.Use(authMiddleware.OptionalAuth())
				router2.GET("/check", func(c *gin.Context) {
					capturedID, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer not.a.valid.token")
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedOK).To(BeFalse())
				Expect(capturedID).To(Equal(uuid.Nil))
			})
		})

		When("a blacklisted token is provided", func() {
			It("should continue without setting user context and return 200", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "attendee", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(true, nil)

				var capturedOK bool

				router2 := gin.New()
				router2.Use(authMiddleware.OptionalAuth())
				router2.GET("/check", func(c *gin.Context) {
					_, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedOK).To(BeFalse())
			})
		})

		When("the blacklist check fails", func() {
			It("should continue without setting user context and return 200", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "attendee", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, errors.New("redis unavailable"))

				var capturedOK bool

				router2 := gin.New()
				router2.Use(authMiddleware.OptionalAuth())
				router2.GET("/check", func(c *gin.Context) {
					_, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				router2.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(capturedOK).To(BeFalse())
			})
		})
	})

	// ---------------------------------------------------------------------------
	// RequireRole()
	// ---------------------------------------------------------------------------
	Describe("RequireRole", func() {
		// Helper: build a router that already has user ID / role pre-set in context,
		// simulating that Authenticate ran successfully before RequireRole.
		buildRouterWithRole := func(role string, allowedRoles ...string) *gin.Engine {
			r := gin.New()
			// Simulate Authenticate having already set the role.
			r.Use(func(c *gin.Context) {
				if role != "" {
					c.Set(middleware.ContextKeyUserRole, role)
				}
				c.Next()
			})
			r.Use(authMiddleware.RequireRole(allowedRoles...))
			r.GET("/admin", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})
			return r
		}

		When("the user has one of the allowed roles", func() {
			It("should continue to the handler and return 200", func() {
				r := buildRouterWithRole("organizer", "organizer", "admin")

				req := httptest.NewRequest(http.MethodGet, "/admin", nil)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		When("the user does not have an allowed role", func() {
			It("should return 403 with insufficient permissions", func() {
				r := buildRouterWithRole("attendee", "organizer", "admin")

				req := httptest.NewRequest(http.MethodGet, "/admin", nil)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("insufficient permissions"))
			})
		})

		When("no role is present in context (unauthenticated request)", func() {
			It("should return 401 with authentication required", func() {
				r := gin.New()
				// Do NOT set any role in context.
				r.Use(authMiddleware.RequireRole("organizer"))
				r.GET("/admin", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/admin", nil)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				p := decodeProblem(w.Body.Bytes())
				Expect(p.Detail).To(Equal("authentication required"))
			})
		})

		When("multiple allowed roles are specified", func() {
			It("should accept any role that appears in the allowed list", func() {
				roles := []string{"organizer", "admin", "superuser"}
				for _, role := range roles {
					r := buildRouterWithRole(role, "organizer", "admin", "superuser")

					req := httptest.NewRequest(http.MethodGet, "/admin", nil)
					w := httptest.NewRecorder()

					r.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK),
						"expected 200 for role %q", role)
				}
			})
		})
	})

	// ---------------------------------------------------------------------------
	// GetUserID()
	// ---------------------------------------------------------------------------
	Describe("GetUserID", func() {
		When("the context contains a valid user ID", func() {
			It("should return the correct UUID and true", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "organizer", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				var capturedID uuid.UUID
				var capturedOK bool

				r := gin.New()
				r.Use(authMiddleware.Authenticate())
				r.GET("/check", func(c *gin.Context) {
					capturedID, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(capturedOK).To(BeTrue())
				Expect(capturedID).To(Equal(userID))
			})
		})

		When("the context is empty (no authentication)", func() {
			It("should return uuid.Nil and false", func() {
				var capturedID uuid.UUID
				var capturedOK bool

				r := gin.New()
				r.GET("/check", func(c *gin.Context) {
					capturedID, capturedOK = middleware.GetUserID(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(capturedOK).To(BeFalse())
				Expect(capturedID).To(Equal(uuid.Nil))
			})
		})
	})

	// ---------------------------------------------------------------------------
	// GetUserRole()
	// ---------------------------------------------------------------------------
	Describe("GetUserRole", func() {
		When("the context contains a valid user role", func() {
			It("should return the correct role string", func() {
				userID := uuid.New()
				validToken := newAccessToken(userID, "attendee", time.Hour)

				mockBlacklist.EXPECT().
					IsBlacklisted(gomock.Any(), validToken).
					Return(false, nil)

				var capturedRole string

				r := gin.New()
				r.Use(authMiddleware.Authenticate())
				r.GET("/check", func(c *gin.Context) {
					capturedRole = middleware.GetUserRole(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(capturedRole).To(Equal("attendee"))
			})
		})

		When("the context is empty (no authentication)", func() {
			It("should return an empty string", func() {
				var capturedRole string

				r := gin.New()
				r.GET("/check", func(c *gin.Context) {
					capturedRole = middleware.GetUserRole(c)
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest(http.MethodGet, "/check", nil)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				Expect(capturedRole).To(Equal(""))
			})
		})
	})
})
