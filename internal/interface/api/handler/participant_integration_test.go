//go:build integration
// +build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	redisClient "github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/container"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api"
	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Participant API Integration", func() {
	var (
		router        *gin.Engine
		cfg           *config.Config
		log           *logger.Logger
		db            database.Service
		cacheService  cache.Service
		redis         *redisClient.Client
		organizerAuth *generated.AuthResponse
		adminAuth     *generated.AuthResponse
		testEventID   string
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
			if parsedDB, err := strconv.Atoi(redisDBStr); err == nil {
				redisDB = parsedDB
			}
		}

		jwtSecret := "test-secret-key-minimum-32-characters-long-for-testing"
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
			Level:       "warn",
			Format:      "console",
			Environment: "development",
		})

		ctx := context.Background()
		db, err = database.NewPostgresDB(ctx, &cfg.Database, log)
		Expect(err).NotTo(HaveOccurred())

		redis, err = redisClient.NewClient(&redisClient.ClientConfig{
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
		cacheService = redis

		// Clean database before each test
		cleanDatabaseForEvents(db, redis)

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

		// Create a test event for participant tests
		eventReq := map[string]interface{}{
			"name":        "Test Event for Participants",
			"description": "A test event",
			"start_date":  time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
			"end_date":    time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
			"location":    "Test Location",
			"timezone":    "Asia/Tokyo",
			"status":      "draft",
		}

		eventJSON, _ := json.Marshal(eventReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusCreated))

		var event generated.Event
		err = json.Unmarshal(w.Body.Bytes(), &event)
		Expect(err).NotTo(HaveOccurred())
		testEventID = event.Id.String()
	})

	AfterEach(func() {
		if db != nil && redis != nil {
			cleanDatabaseForEvents(db, redis)
		}
		if db != nil {
			db.Close()
		}
		if redis != nil {
			redis.Close()
		}
	})

	Describe("POST /api/v1/events/:id/participants", func() {
		When("creating a participant with valid data", func() {
			Context("as event organizer", func() {
				It("should create participant with QR code", func() {
					participantReq := map[string]interface{}{
						"name":  "John Doe",
						"email": "john@example.com",
					}

					reqBody, _ := json.Marshal(participantReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/participants",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusCreated))

					var participant generated.Participant
					err := json.Unmarshal(w.Body.Bytes(), &participant)
					Expect(err).NotTo(HaveOccurred())
					Expect(participant.Name).To(Equal("John Doe"))
					Expect(string(participant.Email)).To(Equal("john@example.com"))
					Expect(participant.QrCode).NotTo(BeNil())
					Expect(*participant.QrCode).NotTo(BeEmpty())
					Expect(participant.EventId.String()).To(Equal(testEventID))
				})
			})

			Context("as admin", func() {
				It("should create participant successfully", func() {
					participantReq := map[string]interface{}{
						"name":  "Jane Smith",
						"email": "jane@example.com",
					}

					reqBody, _ := json.Marshal(participantReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/participants",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusCreated))
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				participantReq := map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				}

				reqBody, _ := json.Marshal(participantReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("user does not have permission", func() {
			It("should return 403 Forbidden", func() {
				// Create another user
				createTestUserV1(router, "otheruser@example.com", "Password123!", "Other User", "organizer")
				otherAuth := loginTestUserV1(router, "otheruser@example.com", "Password123!")

				participantReq := map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				}

				reqBody, _ := json.Marshal(participantReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})

		When("duplicate email", func() {
			It("should return 409 Conflict", func() {
				participantReq := map[string]interface{}{
					"name":  "John Doe",
					"email": "duplicate@example.com",
				}

				reqBody, _ := json.Marshal(participantReq)

				// First create
				req1 := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants",
					bytes.NewReader(reqBody),
				)
				req1.Header.Set("Content-Type", "application/json")
				req1.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				w1 := httptest.NewRecorder()
				router.ServeHTTP(w1, req1)
				Expect(w1.Code).To(Equal(http.StatusCreated))

				// Second create (duplicate)
				req2 := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants",
					bytes.NewReader(reqBody),
				)
				req2.Header.Set("Content-Type", "application/json")
				req2.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				w2 := httptest.NewRecorder()
				router.ServeHTTP(w2, req2)
				Expect(w2.Code).To(Equal(http.StatusConflict))
			})
		})
	})

	Describe("GET /api/v1/events/:id/participants", func() {
		BeforeEach(func() {
			// Create test participants
			createTestParticipant(router, testEventID, organizerAuth.AccessToken, "Alice", "alice@example.com")
			createTestParticipant(router, testEventID, organizerAuth.AccessToken, "Bob", "bob@example.com")
		})

		When("listing participants", func() {
			Context("as event organizer", func() {
				It("should return list of participants", func() {
					req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/participants", nil)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.ParticipantListResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(response.Data)).To(BeNumerically(">=", 2))
					Expect(response.Meta.Total).To(BeNumerically(">=", 2))
				})
			})

			Context("with pagination", func() {
				It("should respect per_page parameter", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/events/"+testEventID+"/participants?per_page=1",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.ParticipantListResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(response.Data)).To(Equal(1))
				})
			})

			Context("with search", func() {
				It("should filter by search query", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/events/"+testEventID+"/participants?search=alice",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.ParticipantListResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(response.Data)).To(BeNumerically(">=", 1))
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/participants", nil)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("GET /api/v1/participants/:id", func() {
		var participantID string

		BeforeEach(func() {
			p := createTestParticipant(router, testEventID, organizerAuth.AccessToken, "Test User", "test@example.com")
			participantID = p.Id.String()
		})

		When("getting participant details", func() {
			Context("as event organizer", func() {
				It("should return participant details", func() {
					req := httptest.NewRequest(http.MethodGet, "/api/v1/participants/"+participantID, nil)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var participant generated.Participant
					err := json.Unmarshal(w.Body.Bytes(), &participant)
					Expect(err).NotTo(HaveOccurred())
					Expect(participant.Id.String()).To(Equal(participantID))
					Expect(participant.Name).To(Equal("Test User"))
				})
			})
		})

		When("participant does not exist", func() {
			It("should return 404 Not Found", func() {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/participants/00000000-0000-0000-0000-000000000000",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("PUT /api/v1/participants/:id", func() {
		var participantID string

		BeforeEach(func() {
			p := createTestParticipant(
				router,
				testEventID,
				organizerAuth.AccessToken,
				"Original Name",
				"original@example.com",
			)
			participantID = p.Id.String()
		})

		When("updating participant with valid data", func() {
			Context("as event organizer", func() {
				It("should update participant successfully", func() {
					updateReq := map[string]interface{}{
						"name": "Updated Name",
					}

					reqBody, _ := json.Marshal(updateReq)
					req := httptest.NewRequest(
						http.MethodPut,
						"/api/v1/participants/"+participantID,
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var participant generated.Participant
					err := json.Unmarshal(w.Body.Bytes(), &participant)
					Expect(err).NotTo(HaveOccurred())
					Expect(participant.Name).To(Equal("Updated Name"))
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				updateReq := map[string]interface{}{
					"name": "Updated Name",
				}

				reqBody, _ := json.Marshal(updateReq)
				req := httptest.NewRequest(
					http.MethodPut,
					"/api/v1/participants/"+participantID,
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("DELETE /api/v1/participants/:id", func() {
		var participantID string

		BeforeEach(func() {
			p := createTestParticipant(
				router,
				testEventID,
				organizerAuth.AccessToken,
				"To Delete",
				"delete@example.com",
			)
			participantID = p.Id.String()
		})

		When("deleting participant", func() {
			Context("as event organizer", func() {
				It("should delete participant successfully", func() {
					req := httptest.NewRequest(http.MethodDelete, "/api/v1/participants/"+participantID, nil)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNoContent))

					// Verify deletion
					verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/participants/"+participantID, nil)
					verifyReq.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
					verifyW := httptest.NewRecorder()
					router.ServeHTTP(verifyW, verifyReq)
					Expect(verifyW.Code).To(Equal(http.StatusNotFound))
				})
			})
		})
	})

	Describe("GET /api/v1/participants/:id/qrcode", func() {
		var participantID string

		BeforeEach(func() {
			p := createTestParticipant(router, testEventID, organizerAuth.AccessToken, "QR Test", "qr@example.com")
			participantID = p.Id.String()
		})

		When("downloading QR code in PNG format", func() {
			Context("as event organizer", func() {
				It("should return PNG QR code", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/participants/"+participantID+"/qrcode?format=png",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Header().Get("Content-Type")).To(Equal("image/png"))
					Expect(w.Body.Len()).To(BeNumerically(">", 0))
				})
			})
		})

		When("downloading QR code in SVG format", func() {
			Context("as event organizer", func() {
				It("should return SVG QR code", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/participants/"+participantID+"/qrcode?format=svg",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(w.Header().Get("Content-Type")).To(Equal("image/svg+xml"))
					Expect(w.Body.Len()).To(BeNumerically(">", 0))
				})
			})
		})

		When("invalid format is requested", func() {
			It("should return 400 Bad Request", func() {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/participants/"+participantID+"/qrcode?format=invalid",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("POST /api/v1/events/:id/participants/bulk", func() {
		When("creating multiple participants", func() {
			Context("with all valid data", func() {
				It("should create all participants successfully", func() {
					bulkReq := map[string]interface{}{
						"participants": []map[string]interface{}{
							{"name": "Participant 1", "email": "p1@example.com"},
							{"name": "Participant 2", "email": "p2@example.com"},
							{"name": "Participant 3", "email": "p3@example.com"},
						},
					}

					reqBody, _ := json.Marshal(bulkReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/participants/bulk",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusCreated))

					var response generated.BulkCreateParticipantsResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.CreatedCount).To(Equal(3))
					Expect(response.FailedCount).To(Equal(0))
					Expect(len(response.Participants)).To(Equal(3))
				})
			})

			Context("with partial failures", func() {
				It("should create valid participants and report errors", func() {
					// Create one participant first to cause duplicate
					createTestParticipant(
						router,
						testEventID,
						organizerAuth.AccessToken,
						"Existing",
						"existing@example.com",
					)

					bulkReq := map[string]interface{}{
						"participants": []map[string]interface{}{
							{"name": "Valid 1", "email": "valid1@example.com"},
							{"name": "Duplicate", "email": "existing@example.com"}, // Will fail
							{"name": "Valid 2", "email": "valid2@example.com"},
						},
					}

					reqBody, _ := json.Marshal(bulkReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/participants/bulk",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusCreated))

					var response generated.BulkCreateParticipantsResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.CreatedCount).To(BeNumerically(">=", 2))
					Expect(response.FailedCount).To(BeNumerically(">=", 1))
				})
			})
		})
	})
	Describe("POST /api/v1/events/:id/participants/import", func() {
		When("importing a valid CSV as event organizer", func() {
			It("should import all rows and return imported_count=2", func() {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "participants.csv")
				Expect(err).To(BeNil())
				_, err = strings.NewReader("name,email\nJane Smith,jane-csv@example.com\nJohn Doe,john-csv@example.com").WriteTo(part)
				Expect(err).To(BeNil())
				Expect(writer.Close()).To(Succeed())

				req, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					body,
				)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				var resp map[string]interface{}
				Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
				data := resp["data"].(map[string]interface{})
				Expect(data["imported_count"]).To(BeNumerically("==", 2))
				Expect(data["failed_count"]).To(BeNumerically("==", 0))
			})
		})

		When("importing CSV with duplicate email and skip_duplicates=true", func() {
			It("should skip the duplicate row and return skipped_count=1", func() {
				// First import
				body1 := &bytes.Buffer{}
				w1 := multipart.NewWriter(body1)
				part1, _ := w1.CreateFormFile("file", "p.csv")
				_, _ = strings.NewReader("name,email\nDup User,dup-import@example.com").WriteTo(part1)
				_ = w1.Close()
				req1, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					body1,
				)
				req1.Header.Set("Content-Type", w1.FormDataContentType())
				req1.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				router.ServeHTTP(httptest.NewRecorder(), req1)

				// Second import with skip_duplicates=true
				body2 := &bytes.Buffer{}
				w2 := multipart.NewWriter(body2)
				part2, _ := w2.CreateFormFile("file", "p.csv")
				_, _ = strings.NewReader("name,email\nDup User,dup-import@example.com").WriteTo(part2)
				_ = w2.Close()
				req2, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import?skip_duplicates=true",
					body2,
				)
				req2.Header.Set("Content-Type", w2.FormDataContentType())
				req2.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req2)

				Expect(rec.Code).To(Equal(http.StatusOK))
				var resp map[string]interface{}
				Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
				data := resp["data"].(map[string]interface{})
				Expect(data["imported_count"]).To(BeNumerically("==", 0))
				Expect(data["skipped_count"]).To(BeNumerically("==", 1))
			})
		})

		When("importing CSV with duplicate email and skip_duplicates=false (default)", func() {
			It("should record the duplicate row as an error", func() {
				// First import
				body1 := &bytes.Buffer{}
				w1 := multipart.NewWriter(body1)
				part1, _ := w1.CreateFormFile("file", "p.csv")
				_, _ = strings.NewReader("name,email\nErr User,err-dup@example.com").WriteTo(part1)
				_ = w1.Close()
				req1, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					body1,
				)
				req1.Header.Set("Content-Type", w1.FormDataContentType())
				req1.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				router.ServeHTTP(httptest.NewRecorder(), req1)

				// Second import without skip_duplicates
				body2 := &bytes.Buffer{}
				w2 := multipart.NewWriter(body2)
				part2, _ := w2.CreateFormFile("file", "p.csv")
				_, _ = strings.NewReader("name,email\nErr User,err-dup@example.com").WriteTo(part2)
				_ = w2.Close()
				req2, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					body2,
				)
				req2.Header.Set("Content-Type", w2.FormDataContentType())
				req2.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req2)

				Expect(rec.Code).To(Equal(http.StatusOK))
				var resp map[string]interface{}
				Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
				data := resp["data"].(map[string]interface{})
				Expect(data["failed_count"]).To(BeNumerically("==", 1))
				Expect(data["skipped_count"]).To(BeNumerically("==", 0))
			})
		})

		When("request has no file field", func() {
			It("should return 400", func() {
				req, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					bytes.NewBufferString(""),
				)
				req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("CSV has invalid format (missing required columns)", func() {
			It("should return 400", func() {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "bad.csv")
				_, _ = strings.NewReader("phone,employee_id\n+1-555,EMP001").WriteTo(part)
				_ = writer.Close()

				req, _ := http.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/participants/import",
					body,
				)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})
})

// Helper function to create a test participant
func createTestParticipant(router *gin.Engine, eventID, token, name, email string) *generated.Participant {
	participantReq := map[string]interface{}{
		"name":  name,
		"email": email,
	}

	reqBody, _ := json.Marshal(participantReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+eventID+"/participants", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		panic(fmt.Sprintf("failed to create test participant: %d", w.Code))
	}

	var participant generated.Participant
	json.Unmarshal(w.Body.Bytes(), &participant)
	return &participant
}
