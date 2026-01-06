//go:build integration
// +build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/internal/interface/api/handler"
	"github.com/fumkob/ezqrin-server/internal/interface/api/middleware"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ = Describe("Authentication API Integration", func() {
	var (
		router        *gin.Engine
		cfg           *config.Config
		log           *logger.Logger
		db            database.Service
		cacheService  cache.Service
		redisClient   *redis.Client
		authHandler   *handler.AuthHandler
		healthHandler *handler.HealthHandler
		jwtSecret     string
		testUserEmail string
		testUserName  string
		testUserPass  string
		testUserRole  string
	)

	BeforeEach(func() {
		var err error

		// Create test configuration directly (avoid file system dependency)
		jwtSecret = "test-secret-key-minimum-32-characters-long-for-testing"
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "postgres",
				Password:        "postgres",
				Name:            "ezqrin_test",
				SSLMode:         "disable",
				MaxConns:        10,
				MinConns:        2,
				MaxConnLifetime: 30 * time.Minute,
				MaxConnIdleTime: 5 * time.Minute,
			},
			Redis: config.RedisConfig{
				Host:         "localhost",
				Port:         6379,
				Password:     "",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 2,
				MaxRetries:   3,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
			JWT: config.JWTConfig{
				Secret: jwtSecret,
			},
		}

		// Create test logger
		log, err = logger.New(logger.Config{
			Level:       "info",
			Format:      "json",
			Environment: "test",
		})
		Expect(err).NotTo(HaveOccurred())

		// Connect to test database
		ctx := context.Background()
		db, err = database.NewPostgresDB(ctx, &cfg.Database, log)
		Expect(err).NotTo(HaveOccurred())

		// Connect to Redis
		redisConfig := &redis.ClientConfig{
			Host:         cfg.Redis.Host,
			Port:         fmt.Sprintf("%d", cfg.Redis.Port),
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		}
		redisClient, err = redis.NewClient(redisConfig)
		Expect(err).NotTo(HaveOccurred())
		cacheService = redisClient

		// Initialize repositories
		userRepo := database.NewUserRepository(db.GetPool(), log)
		blacklistRepo := redis.NewTokenBlacklistRepository(redisClient)

		// Initialize use cases
		registerUC := auth.NewRegisterUseCase(userRepo, jwtSecret, log)
		loginUC := auth.NewLoginUseCase(userRepo, jwtSecret, log)
		refreshTokenUC := auth.NewRefreshTokenUseCase(userRepo, blacklistRepo, jwtSecret, log)
		logoutUC := auth.NewLogoutUseCase(blacklistRepo, jwtSecret, log)

		// Create handlers
		authHandler = handler.NewAuthHandler(registerUC, loginUC, refreshTokenUC, logoutUC, log)
		healthHandler = handler.NewHealthHandler(db, cacheService, log)

		// Setup router
		gin.SetMode(gin.TestMode)
		router = gin.New()
		router.Use(middleware.RequestID())

		// Register routes using generated handler registration
		combinedHandler := handler.NewHandler(healthHandler, authHandler)
		generated.RegisterHandlers(router, combinedHandler)

		// Clean database before each test
		cleanDatabase(db)

		// Clean Redis before each test
		flushCtx := context.Background()
		_, err = redisClient.GetClient().FlushAll(flushCtx).Result()
		Expect(err).NotTo(HaveOccurred())

		// Setup test user data
		testUserEmail = fmt.Sprintf("test-%s@example.com", uuid.New().String())
		testUserName = "Test User"
		testUserPass = "SecurePassword123!"
		testUserRole = string(entity.RoleOrganizer)
	})

	AfterEach(func() {
		if db != nil {
			cleanDatabase(db)
			db.Close()
		}
		if redisClient != nil {
			redisClient.Close()
		}
	})

	When("registering a new user", func() {
		Context("with valid input", func() {
			It("should create user and return tokens with 201 Created", func() {
				reqBody := generated.RegisterRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: testUserPass,
					Name:     testUserName,
					Role:     generated.UserRole(testUserRole),
				}

				body, err := json.Marshal(reqBody)
				Expect(err).NotTo(HaveOccurred())

				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusCreated))

				var response struct {
					Data generated.AuthResponse `json:"data"`
				}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())

				// Verify response structure
				Expect(response.Data.AccessToken).NotTo(BeEmpty())
				Expect(response.Data.RefreshToken).NotTo(BeEmpty())
				Expect(response.Data.TokenType).To(Equal("Bearer"))
				Expect(response.Data.ExpiresIn).To(Equal(900)) // 15 minutes

				// Verify user info
				Expect(response.Data.User.Email).To(Equal(openapi_types.Email(testUserEmail)))
				Expect(response.Data.User.Name).To(Equal(testUserName))
				Expect(response.Data.User.Role).To(Equal(generated.UserRole(testUserRole)))
				Expect(response.Data.User.Id).NotTo(BeNil())
				Expect(response.Data.User.CreatedAt).NotTo(BeNil())
				Expect(response.Data.User.UpdatedAt).NotTo(BeNil())

				// Verify access token is valid JWT
				claims, err := crypto.ParseToken(response.Data.AccessToken, jwtSecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(claims.UserID).To(Equal(*response.Data.User.Id))
				Expect(claims.Role).To(Equal(testUserRole))
				Expect(claims.TokenType).To(Equal(crypto.TokenTypeAccess))

				// Verify refresh token is valid JWT
				refreshClaims, err := crypto.ParseToken(response.Data.RefreshToken, jwtSecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(refreshClaims.UserID).To(Equal(*response.Data.User.Id))
				Expect(refreshClaims.TokenType).To(Equal(crypto.TokenTypeRefresh))
			})
		})

		Context("with duplicate email", func() {
			It("should return 409 Conflict", func() {
				// First registration
				reqBody := generated.RegisterRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: testUserPass,
					Name:     testUserName,
					Role:     generated.UserRole(testUserRole),
				}

				body, _ := json.Marshal(reqBody)
				req1 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req1.Header.Set("Content-Type", "application/json")
				w1 := httptest.NewRecorder()
				router.ServeHTTP(w1, req1)
				Expect(w1.Code).To(Equal(http.StatusCreated))

				// Second registration with same email
				body, _ = json.Marshal(reqBody)
				req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req2.Header.Set("Content-Type", "application/json")
				w2 := httptest.NewRecorder()
				router.ServeHTTP(w2, req2)

				Expect(w2.Code).To(Equal(http.StatusConflict))
				Expect(w2.Body.String()).To(ContainSubstring("email already exists"))
				Expect(w2.Body.String()).To(ContainSubstring("CONFLICT"))
				Expect(w2.Header().Get("Content-Type")).To(ContainSubstring("application/problem+json"))
			})
		})

		Context("with invalid email format", func() {
			It("should return 400 Bad Request", func() {
				reqBody := map[string]interface{}{
					"email":    "invalid-email",
					"password": testUserPass,
					"name":     testUserName,
					"role":     testUserRole,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("BAD_REQUEST"))
			})
		})

		Context("with weak password", func() {
			It("should return 400 Bad Request", func() {
				reqBody := generated.RegisterRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: "weak",
					Name:     testUserName,
					Role:     generated.UserRole(testUserRole),
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("password"))
				Expect(w.Body.String()).To(ContainSubstring("at least"))
			})
		})

		Context("with missing required fields", func() {
			It("should return 400 Bad Request for missing name", func() {
				reqBody := generated.RegisterRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: testUserPass,
					Name:     "",
					Role:     generated.UserRole(testUserRole),
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("name"))
			})

			It("should return 400 Bad Request for missing password", func() {
				reqBody := generated.RegisterRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: "",
					Name:     testUserName,
					Role:     generated.UserRole(testUserRole),
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("password"))
			})
		})

		Context("with invalid role", func() {
			It("should return 400 Bad Request", func() {
				reqBody := map[string]interface{}{
					"email":    testUserEmail,
					"password": testUserPass,
					"name":     testUserName,
					"role":     "invalid_role",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("role"))
			})
		})
	})

	When("logging in", func() {
		var registeredUserID uuid.UUID

		BeforeEach(func() {
			// Create a test user first
			registeredUserID = createTestUser(router, testUserEmail, testUserPass, testUserName, testUserRole)
		})

		Context("with valid credentials", func() {
			It("should authenticate and return tokens with 200 OK", func() {
				reqBody := generated.LoginRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: testUserPass,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response struct {
					Data generated.AuthResponse `json:"data"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())

				// Verify response structure
				Expect(response.Data.AccessToken).NotTo(BeEmpty())
				Expect(response.Data.RefreshToken).NotTo(BeEmpty())
				Expect(response.Data.TokenType).To(Equal("Bearer"))
				Expect(response.Data.ExpiresIn).To(Equal(900))

				// Verify user info
				Expect(response.Data.User.Email).To(Equal(openapi_types.Email(testUserEmail)))
				Expect(response.Data.User.Name).To(Equal(testUserName))
				Expect(*response.Data.User.Id).To(Equal(openapi_types.UUID(registeredUserID)))

				// Verify tokens are valid
				claims, err := crypto.ParseToken(response.Data.AccessToken, jwtSecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(claims.UserID).To(Equal(registeredUserID))
			})
		})

		Context("with invalid password", func() {
			It("should return 401 Unauthorized", func() {
				reqBody := generated.LoginRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: "WrongPassword123!",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("invalid credentials"))
				Expect(w.Body.String()).To(ContainSubstring("UNAUTHORIZED"))
			})
		})

		Context("with non-existent email", func() {
			It("should return 401 Unauthorized", func() {
				reqBody := generated.LoginRequest{
					Email:    openapi_types.Email("nonexistent@example.com"),
					Password: testUserPass,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("invalid credentials"))
			})
		})

		Context("with missing email", func() {
			It("should return 400 Bad Request", func() {
				reqBody := generated.LoginRequest{
					Email:    "",
					Password: testUserPass,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with missing password", func() {
			It("should return 400 Bad Request", func() {
				reqBody := generated.LoginRequest{
					Email:    openapi_types.Email(testUserEmail),
					Password: "",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("password"))
			})
		})
	})

	When("refreshing token", func() {
		var (
			accessToken  string
			refreshToken string
			userID       uuid.UUID
		)

		BeforeEach(func() {
			// Create user and get tokens
			userID = createTestUser(router, testUserEmail, testUserPass, testUserName, testUserRole)
			tokens := loginTestUser(router, testUserEmail, testUserPass)
			accessToken = tokens.AccessToken
			refreshToken = tokens.RefreshToken
		})

		Context("with valid refresh token", func() {
			It("should refresh and return new tokens with 200 OK", func() {
				reqBody := generated.RefreshTokenRequest{
					RefreshToken: refreshToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response struct {
					Data generated.AuthResponse `json:"data"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())

				// Verify new tokens
				Expect(response.Data.AccessToken).NotTo(BeEmpty())
				Expect(response.Data.RefreshToken).NotTo(BeEmpty())
				Expect(response.Data.AccessToken).NotTo(Equal(accessToken))
				Expect(response.Data.RefreshToken).NotTo(Equal(refreshToken))

				// Verify new tokens are valid
				claims, err := crypto.ParseToken(response.Data.AccessToken, jwtSecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(claims.UserID).To(Equal(userID))
			})

			It("should blacklist old refresh token", func() {
				reqBody := generated.RefreshTokenRequest{
					RefreshToken: refreshToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))

				// Try to use old refresh token again
				req2 := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req2.Header.Set("Content-Type", "application/json")
				w2 := httptest.NewRecorder()

				router.ServeHTTP(w2, req2)

				Expect(w2.Code).To(Equal(http.StatusUnauthorized))
				Expect(w2.Body.String()).To(ContainSubstring("revoked"))
			})
		})

		Context("with invalid refresh token", func() {
			It("should return 401 Unauthorized", func() {
				reqBody := generated.RefreshTokenRequest{
					RefreshToken: "invalid.token.here",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("invalid refresh token"))
			})
		})

		Context("with expired refresh token", func() {
			It("should return 401 Unauthorized", func() {
				// Generate expired token
				expiredToken, err := crypto.GenerateRefreshToken(
					userID.String(),
					testUserRole,
					jwtSecret,
					-1*time.Hour, // Expired 1 hour ago
				)
				Expect(err).NotTo(HaveOccurred())

				reqBody := generated.RefreshTokenRequest{
					RefreshToken: expiredToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("expired"))
			})
		})

		Context("with blacklisted refresh token", func() {
			It("should return 401 Unauthorized", func() {
				// Blacklist the token first
				ctx := context.Background()
				blacklistRepo := redis.NewTokenBlacklistRepository(redisClient)
				err := blacklistRepo.AddToBlacklist(ctx, refreshToken, 1*time.Hour)
				Expect(err).NotTo(HaveOccurred())

				reqBody := generated.RefreshTokenRequest{
					RefreshToken: refreshToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("revoked"))
			})
		})

		Context("with missing refresh token", func() {
			It("should return 400 Bad Request", func() {
				reqBody := generated.RefreshTokenRequest{
					RefreshToken: "",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				Expect(w.Body.String()).To(ContainSubstring("refresh_token"))
			})
		})
	})

	When("logging out", func() {
		var (
			accessToken  string
			refreshToken string
		)

		BeforeEach(func() {
			// Create user and get tokens
			createTestUser(router, testUserEmail, testUserPass, testUserName, testUserRole)
			tokens := loginTestUser(router, testUserEmail, testUserPass)
			accessToken = tokens.AccessToken
			refreshToken = tokens.RefreshToken
		})

		Context("with valid token", func() {
			It("should logout successfully and return 200 OK", func() {
				reqBody := map[string]string{
					"refresh_token": refreshToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response struct {
					Data generated.LogoutResponse `json:"data"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred())
				Expect(response.Data.Message).To(ContainSubstring("Successfully logged out"))
			})

			It("should blacklist both tokens", func() {
				reqBody := map[string]string{
					"refresh_token": refreshToken,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))

				// Try to use access token for authenticated request
				req2 := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req2.Header.Set("Content-Type", "application/json")
				req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
				w2 := httptest.NewRecorder()

				router.ServeHTTP(w2, req2)

				Expect(w2.Code).To(Equal(http.StatusUnauthorized))
				Expect(w2.Body.String()).To(ContainSubstring("revoked"))
			})
		})

		Context("with expired token", func() {
			It("should succeed as best effort", func() {
				// Generate expired access token
				expiredToken, err := crypto.GenerateAccessToken(
					uuid.New().String(),
					testUserRole,
					jwtSecret,
					-1*time.Hour,
				)
				Expect(err).NotTo(HaveOccurred())

				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", expiredToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				// Expired token should fail authentication middleware
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("without authentication", func() {
			It("should require authentication and return 401 Unauthorized", func() {
				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("missing authorization token"))
			})
		})
	})

	When("using authentication middleware", func() {
		var (
			accessToken string
			userID      uuid.UUID
		)

		BeforeEach(func() {
			// Create user and get tokens
			userID = createTestUser(router, testUserEmail, testUserPass, testUserName, testUserRole)
			tokens := loginTestUser(router, testUserEmail, testUserPass)
			accessToken = tokens.AccessToken
		})

		Context("with valid token", func() {
			It("should allow access and extract user context", func() {
				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		Context("with missing Authorization header", func() {
			It("should reject with 401 Unauthorized", func() {
				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("missing authorization token"))
			})
		})

		Context("with invalid token format", func() {
			It("should reject with 401 Unauthorized", func() {
				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "InvalidFormat token")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("with expired token", func() {
			It("should reject with 401 Unauthorized", func() {
				expiredToken, err := crypto.GenerateAccessToken(
					userID.String(),
					testUserRole,
					jwtSecret,
					-1*time.Hour,
				)
				Expect(err).NotTo(HaveOccurred())

				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", expiredToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("expired"))
			})
		})

		Context("with blacklisted token", func() {
			It("should reject with 401 Unauthorized", func() {
				// Blacklist the token
				ctx := context.Background()
				blacklistRepo := redis.NewTokenBlacklistRepository(redisClient)
				err := blacklistRepo.AddToBlacklist(ctx, accessToken, 1*time.Hour)
				Expect(err).NotTo(HaveOccurred())

				reqBody := map[string]string{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("revoked"))
			})
		})
	})
})

// Helper functions

// createTestUser creates a test user via registration endpoint and returns user ID
func createTestUser(router *gin.Engine, email, password, name, role string) uuid.UUID {
	reqBody := generated.RegisterRequest{
		Email:    openapi_types.Email(email),
		Password: password,
		Name:     name,
		Role:     generated.UserRole(role),
	}

	body, err := json.Marshal(reqBody)
	Expect(err).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var response struct {
		Data generated.AuthResponse `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	Expect(err).NotTo(HaveOccurred())

	return uuid.UUID(*response.Data.User.Id)
}

// loginTestUser logs in a test user and returns tokens
func loginTestUser(router *gin.Engine, email, password string) *generated.AuthResponse {
	reqBody := generated.LoginRequest{
		Email:    openapi_types.Email(email),
		Password: password,
	}

	body, err := json.Marshal(reqBody)
	Expect(err).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusOK))

	var response struct {
		Data generated.AuthResponse `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	Expect(err).NotTo(HaveOccurred())

	return &response.Data
}

// cleanDatabase cleans all test data from database
func cleanDatabase(db database.Service) {
	ctx := context.Background()
	_, err := db.GetPool().Exec(ctx, "TRUNCATE TABLE users CASCADE")
	Expect(err).NotTo(HaveOccurred())
}
