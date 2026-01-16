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
	"os"
	"strconv"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/container"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event API Integration", func() {
	var (
		router        *gin.Engine
		cfg           *config.Config
		log           *logger.Logger
		db            database.Service
		cacheService  cache.Service
		redisClient   *redis.Client
		jwtSecret     string
		organizerAuth *generated.AuthResponse
		adminAuth     *generated.AuthResponse
	)

	BeforeEach(func() {
		var err error

		// Set up environment for tests
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		redisDBStr := os.Getenv("TEST_REDIS_DB")
		redisDB := 1
		if redisDBStr != "" {
			if db, err := strconv.Atoi(redisDBStr); err == nil {
				redisDB = db
			}
		}

		jwtSecret = "test-secret-key-minimum-32-characters-long-for-testing"
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:            dbHost,
				Port:            5432,
				User:            "ezqrin",
				Password:        "ezqrin_dev",
				Name:            "ezqrin_test",
				SSLMode:         "disable",
				MaxConns:        10,
				MinConns:        2,
				MaxConnLifetime: 30 * time.Minute,
				MaxConnIdleTime: 5 * time.Minute,
			},
			Redis: config.RedisConfig{
				Host:         redisHost,
				Port:         6379,
				Password:     "",
				DB:           redisDB,
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
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"*"},
			},
		}

		log, _ = logger.New(logger.Config{
			Level:       "warn", // Suppress logs in tests
			Format:      "console",
			Environment: "development",
		})

		ctx := context.Background()
		db, err = database.NewPostgresDB(ctx, &cfg.Database, log)
		Expect(err).NotTo(HaveOccurred())

		redisClient, err = redis.NewClient(&redis.ClientConfig{
			Host:         cfg.Redis.Host,
			Port:         strconv.Itoa(cfg.Redis.Port),
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			MaxRetries:   cfg.Redis.MaxRetries,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		})
		Expect(err).NotTo(HaveOccurred())
		cacheService = redisClient

		// Clean database before each test
		cleanDatabaseForEvents(db, redisClient)

		// Set up container
		appContainer := container.NewContainer(cfg, log, db, cacheService)

		// Set up router
		router = api.SetupRouter(&api.RouterDependencies{
			Config:    cfg,
			Logger:    log,
			DB:        db,
			Cache:     cacheService,
			Container: appContainer,
		})

		// Create organizer and admin for tests
		createTestUserV1(router, "organizer@example.com", "Password123!", "Organizer User", "organizer")
		organizerAuth = loginTestUserV1(router, "organizer@example.com", "Password123!")

		createTestUserV1(router, "admin@example.com", "Password123!", "Admin User", "admin")
		adminAuth = loginTestUserV1(router, "admin@example.com", "Password123!")
	})

	AfterEach(func() {
		if db != nil && redisClient != nil {
			cleanDatabaseForEvents(db, redisClient)
		}
		if db != nil {
			db.Close()
		}
		if redisClient != nil {
			_ = redisClient.Close()
		}
	})

	Describe("POST /events", func() {
		It("should create a new event when authorized as organizer", func() {
			reqBody := generated.CreateEventRequest{
				Name:        "Test Event",
				Description: stringPtr("Test Description"),
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     timePtr(time.Now().Add(48 * time.Hour)),
				Location:    stringPtr("Test Location"),
				Status:      generated.EventStatusDraft,
			}

			body, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusCreated))

			var response generated.Event
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Name).To(Equal(reqBody.Name))
			Expect(response.OrganizerId.String()).To(Equal(organizerAuth.User.Id.String()))
		})

		It("should return 401 when unauthorized", func() {
			reqBody := generated.CreateEventRequest{
				Name:      "Test Event",
				StartDate: time.Now().Add(24 * time.Hour),
				Status:    generated.EventStatusDraft,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /events", func() {
		BeforeEach(func() {
			// Create some events for testing
			for i := 1; i <= 3; i++ {
				createEvent(router, organizerAuth.AccessToken, fmt.Sprintf("Organizer Event %d", i))
			}
			createEvent(router, adminAuth.AccessToken, "Admin Event")
		})

		It("should list only own events for organizer", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response generated.EventListResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Data).To(HaveLen(3))
			for _, e := range response.Data {
				Expect(e.OrganizerId.String()).To(Equal(organizerAuth.User.Id.String()))
			}
		})

		It("should list all events for admin", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
			req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response generated.EventListResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Data).To(HaveLen(4))
		})
	})

	Describe("DELETE /events/{id}", func() {
		var event *generated.Event

		BeforeEach(func() {
			// Create an event for deletion testing
			event = createEvent(router, organizerAuth.AccessToken, "Test Event for Delete")
		})

		It("should delete event as owner", func() {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/events/%s", event.Id), nil)
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
		})

		It("should return 403 when trying to delete someone else's event", func() {
			// Create an event as organizer
			otherEvent := createEvent(router, organizerAuth.AccessToken, "Another Event")

			// Create another organizer
			createTestUserV1(router, "organizer2@example.com", "Password123!", "Organizer User 2", "organizer")
			organizer2Auth := loginTestUserV1(router, "organizer2@example.com", "Password123!")

			// Try to delete as different organizer
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/events/%s", otherEvent.Id), nil)
			req.Header.Set("Authorization", "Bearer "+organizer2Auth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("should allow admin to delete any event", func() {
			// Create an event as organizer
			adminDeleteEvent := createEvent(router, organizerAuth.AccessToken, "Admin Can Delete This")

			// Delete as admin
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/events/%s", adminDeleteEvent.Id), nil)
			req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
		})

		It("should return 404 for non-existent event", func() {
			fakeID := uuid.New().String()
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/events/%s", fakeID), nil)
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})

// Helpers

func createTestUserV1(router *gin.Engine, email, password, name, role string) uuid.UUID {
	reqBody := generated.RegisterRequest{
		Email:    openapi_types.Email(email),
		Password: password,
		Name:     name,
		Role:     generated.UserRole(role),
	}

	body, err := json.Marshal(reqBody)
	Expect(err).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var response generated.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	Expect(err).NotTo(HaveOccurred())

	return uuid.UUID(*response.User.Id)
}

func loginTestUserV1(router *gin.Engine, email, password string) *generated.AuthResponse {
	reqBody := generated.LoginRequest{
		Email:    openapi_types.Email(email),
		Password: password,
	}

	body, err := json.Marshal(reqBody)
	Expect(err).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusOK))

	var response generated.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	Expect(err).NotTo(HaveOccurred())

	return &response
}

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func createEvent(router *gin.Engine, token string, name string) *generated.Event {
	reqBody := generated.CreateEventRequest{
		Name:      name,
		StartDate: time.Now().Add(24 * time.Hour),
		Status:    generated.EventStatusPublished,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var response generated.Event
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	return &response
}

func cleanDatabaseForEvents(db database.Service, redisClient *redis.Client) {
	ctx := context.Background()
	pool := db.GetPool()
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE checkins, participants, events, users CASCADE")
	_ = redisClient.GetClient().FlushDB(ctx).Err()
}
