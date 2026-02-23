//go:build integration
// +build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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

var _ = Describe("Check-in API Integration", func() {
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
		participant1  *generated.Participant
		participant2  *generated.Participant
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
		cleanDatabaseForCheckins(db, redis)

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

		// Create users for tests
		createTestUserV1(router, "organizer@example.com", "Password123!", "Organizer User", "organizer")
		organizerAuth = loginTestUserV1(router, "organizer@example.com", "Password123!")

		createTestUserV1(router, "admin@example.com", "Password123!", "Admin User", "admin")
		adminAuth = loginTestUserV1(router, "admin@example.com", "Password123!")

		// Create a test event
		eventReq := map[string]interface{}{
			"name":        "Test Event for Check-in",
			"description": "A test event for check-in functionality",
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

		// Create test participants
		participant1 = createTestParticipant(
			router,
			testEventID,
			organizerAuth.AccessToken,
			"Participant 1",
			"participant1@example.com",
		)
		participant2 = createTestParticipant(
			router,
			testEventID,
			organizerAuth.AccessToken,
			"Participant 2",
			"participant2@example.com",
		)
	})

	AfterEach(func() {
		if db != nil && redis != nil {
			cleanDatabaseForCheckins(db, redis)
		}
		if db != nil {
			db.Close()
		}
		if redis != nil {
			redis.Close()
		}
	})

	Describe("POST /api/v1/events/:id/checkin", func() {
		When("checking in with QR code", func() {
			Context("with valid QR code", func() {
				It("should successfully check in participant", func() {
					checkinReq := map[string]interface{}{
						"method":  "qrcode",
						"qr_code": participant1.QrCode,
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.ParticipantId.String()).To(Equal(participant1.Id.String()))
					Expect(response.Participant.Name).To(Equal("Participant 1"))
					Expect(string(response.CheckinMethod)).To(Equal("qrcode"))
				})
			})

			Context("with invalid QR code", func() {
				It("should return 404 Not Found", func() {
					checkinReq := map[string]interface{}{
						"method":  "qrcode",
						"qr_code": "invalid-qr-code-12345",
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})

			Context("with missing QR code", func() {
				It("should return 400 Bad Request", func() {
					checkinReq := map[string]interface{}{
						"method": "qrcode",
						// Missing qr_code
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		When("checking in manually", func() {
			Context("as event organizer", func() {
				It("should successfully check in participant", func() {
					checkinReq := map[string]interface{}{
						"method":         "manual",
						"participant_id": participant1.Id.String(),
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.ParticipantId.String()).To(Equal(participant1.Id.String()))
					Expect(string(response.CheckinMethod)).To(Equal("manual"))
					Expect(response.CheckedInBy).NotTo(BeNil())
				})
			})

			Context("as admin", func() {
				It("should successfully check in participant", func() {
					checkinReq := map[string]interface{}{
						"method":         "manual",
						"participant_id": participant2.Id.String(),
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
				})
			})

			Context("as non-organizer user", func() {
				It("should return 403 Forbidden", func() {
					// Create another user who is not the organizer
					createTestUserV1(router, "other@example.com", "Password123!", "Other User", "organizer")
					otherAuth := loginTestUserV1(router, "other@example.com", "Password123!")

					checkinReq := map[string]interface{}{
						"method":         "manual",
						"participant_id": participant1.Id.String(),
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusForbidden))
				})
			})

			Context("with invalid participant ID", func() {
				It("should return 404 Not Found", func() {
					checkinReq := map[string]interface{}{
						"method":         "manual",
						"participant_id": "00000000-0000-0000-0000-000000000000",
					}

					reqBody, _ := json.Marshal(checkinReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/checkin",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})
		})

		When("participant already checked in", func() {
			It("should return 409 Conflict", func() {
				checkinReq := map[string]interface{}{
					"method":  "qrcode",
					"qr_code": participant1.QrCode,
				}
				reqBody, _ := json.Marshal(checkinReq)

				// First check-in
				req1 := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/checkin",
					bytes.NewReader(reqBody),
				)
				req1.Header.Set("Content-Type", "application/json")
				req1.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				w1 := httptest.NewRecorder()
				router.ServeHTTP(w1, req1)
				Expect(w1.Code).To(Equal(http.StatusOK))

				// Second check-in (duplicate)
				req2 := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/checkin",
					bytes.NewReader(reqBody),
				)
				req2.Header.Set("Content-Type", "application/json")
				req2.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
				w2 := httptest.NewRecorder()
				router.ServeHTTP(w2, req2)
				Expect(w2.Code).To(Equal(http.StatusConflict))
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				checkinReq := map[string]interface{}{
					"method":  "qrcode",
					"qr_code": participant1.QrCode,
				}

				reqBody, _ := json.Marshal(checkinReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/checkin",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				// No Authorization header

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("request body is invalid", func() {
			It("should return 400 Bad Request", func() {
				invalidJSON := []byte("{invalid json}")
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/checkin",
					bytes.NewReader(invalidJSON),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("event does not exist", func() {
			It("should return 404 Not Found", func() {
				checkinReq := map[string]interface{}{
					"method":  "qrcode",
					"qr_code": participant1.QrCode,
				}

				reqBody, _ := json.Marshal(checkinReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/00000000-0000-0000-0000-000000000000/checkin",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("GET /api/v1/events/:id/checkins", func() {
		BeforeEach(func() {
			// Check in some participants
			checkin1 := map[string]interface{}{
				"method":  "qrcode",
				"qr_code": participant1.QrCode,
			}
			reqBody1, _ := json.Marshal(checkin1)
			req1 := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/events/"+testEventID+"/checkin",
				bytes.NewReader(reqBody1),
			)
			req1.Header.Set("Content-Type", "application/json")
			req1.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w1 := httptest.NewRecorder()
			router.ServeHTTP(w1, req1)

			checkin2 := map[string]interface{}{
				"method":         "manual",
				"participant_id": participant2.Id.String(),
			}
			reqBody2, _ := json.Marshal(checkin2)
			req2 := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/events/"+testEventID+"/checkin",
				bytes.NewReader(reqBody2),
			)
			req2.Header.Set("Content-Type", "application/json")
			req2.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w2 := httptest.NewRecorder()
			router.ServeHTTP(w2, req2)
		})

		When("listing check-ins", func() {
			Context("as event organizer", func() {
				It("should return list of check-ins", func() {
					req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/checkins", nil)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInListResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(response.Checkins)).To(BeNumerically(">=", 2))
					Expect(response.Pagination.Total).To(BeNumerically(">=", 2))
				})
			})

			Context("as admin", func() {
				It("should return list of check-ins", func() {
					req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/checkins", nil)
					req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
				})
			})

			Context("with pagination", func() {
				It("should respect per_page parameter", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/events/"+testEventID+"/checkins?per_page=1",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInListResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(response.Checkins)).To(Equal(1))
				})

				It("should respect page parameter", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/events/"+testEventID+"/checkins?page=1&per_page=10",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/checkins", nil)
				// No Authorization header

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("user does not have permission", func() {
			It("should return 403 Forbidden", func() {
				// Create another user who is not the organizer
				createTestUserV1(router, "other2@example.com", "Password123!", "Other User 2", "organizer")
				otherAuth := loginTestUserV1(router, "other2@example.com", "Password123!")

				req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+testEventID+"/checkins", nil)
				req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})

		When("event does not exist", func() {
			It("should return 404 Not Found", func() {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/events/00000000-0000-0000-0000-000000000000/checkins",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("GET /api/v1/participants/:id/checkin-status", func() {
		var checkedInParticipant *generated.Participant
		var notCheckedInParticipant *generated.Participant

		BeforeEach(func() {
			checkedInParticipant = createTestParticipant(
				router,
				testEventID,
				organizerAuth.AccessToken,
				"Checked In",
				"checkedin@example.com",
			)
			notCheckedInParticipant = createTestParticipant(
				router,
				testEventID,
				organizerAuth.AccessToken,
				"Not Checked In",
				"notcheckedin@example.com",
			)

			// Check in one participant
			checkinReq := map[string]interface{}{
				"method":  "qrcode",
				"qr_code": checkedInParticipant.QrCode,
			}
			reqBody, _ := json.Marshal(checkinReq)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/events/"+testEventID+"/checkin",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})

		When("checking status of checked-in participant", func() {
			Context("as event organizer", func() {
				It("should return checked-in status with details", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/participants/"+checkedInParticipant.Id.String()+"/checkin-status",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInStatusResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.CheckedIn).To(BeTrue())
					Expect(response.Checkin).NotTo(BeNil())
				})
			})
		})

		When("checking status of not checked-in participant", func() {
			Context("as event organizer", func() {
				It("should return not checked-in status", func() {
					req := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/participants/"+notCheckedInParticipant.Id.String()+"/checkin-status",
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.CheckInStatusResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.CheckedIn).To(BeFalse())
					Expect(response.Checkin).To(BeNil())
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/participants/"+checkedInParticipant.Id.String()+"/checkin-status",
					nil,
				)
				// No Authorization header

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("participant does not exist", func() {
			It("should return 404 Not Found", func() {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/participants/00000000-0000-0000-0000-000000000000/checkin-status",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})

		When("user does not have permission", func() {
			It("should return 403 Forbidden", func() {
				// Create another user who is not the organizer
				createTestUserV1(router, "other3@example.com", "Password123!", "Other User 3", "organizer")
				otherAuth := loginTestUserV1(router, "other3@example.com", "Password123!")

				req := httptest.NewRequest(
					http.MethodGet,
					"/api/v1/participants/"+checkedInParticipant.Id.String()+"/checkin-status",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})
	})

	Describe("DELETE /api/v1/events/:id/checkins/:cid", func() {
		var checkinID string

		BeforeEach(func() {
			// Check in a participant
			checkinReq := map[string]interface{}{
				"method":  "qrcode",
				"qr_code": participant1.QrCode,
			}
			reqBody, _ := json.Marshal(checkinReq)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/events/"+testEventID+"/checkin",
				bytes.NewReader(reqBody),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var response generated.CheckInResponse
			json.Unmarshal(w.Body.Bytes(), &response)
			checkinID = response.Id.String()
		})

		When("canceling check-in", func() {
			Context("as event organizer", func() {
				It("should successfully cancel check-in", func() {
					req := httptest.NewRequest(
						http.MethodDelete,
						"/api/v1/events/"+testEventID+"/checkins/"+checkinID,
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNoContent))

					// Verify check-in was canceled
					statusReq := httptest.NewRequest(
						http.MethodGet,
						"/api/v1/participants/"+participant1.Id.String()+"/checkin-status",
						nil,
					)
					statusReq.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)
					statusW := httptest.NewRecorder()
					router.ServeHTTP(statusW, statusReq)

					var statusResponse generated.CheckInStatusResponse
					json.Unmarshal(statusW.Body.Bytes(), &statusResponse)
					Expect(statusResponse.CheckedIn).To(BeFalse())
				})
			})

			Context("as admin", func() {
				It("should successfully cancel check-in", func() {
					req := httptest.NewRequest(
						http.MethodDelete,
						"/api/v1/events/"+testEventID+"/checkins/"+checkinID,
						nil,
					)
					req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNoContent))
				})
			})
		})

		When("authentication is missing", func() {
			It("should return 401 Unauthorized", func() {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/events/"+testEventID+"/checkins/"+checkinID, nil)
				// No Authorization header

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("user does not have permission", func() {
			It("should return 403 Forbidden", func() {
				// Create another user who is not the organizer
				createTestUserV1(router, "other4@example.com", "Password123!", "Other User 4", "organizer")
				otherAuth := loginTestUserV1(router, "other4@example.com", "Password123!")

				req := httptest.NewRequest(http.MethodDelete, "/api/v1/events/"+testEventID+"/checkins/"+checkinID, nil)
				req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})

		When("check-in does not exist", func() {
			It("should return 404 Not Found", func() {
				req := httptest.NewRequest(
					http.MethodDelete,
					"/api/v1/events/"+testEventID+"/checkins/00000000-0000-0000-0000-000000000000",
					nil,
				)
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})
	})
})

// cleanDatabaseForCheckins cleans all test data from database
func cleanDatabaseForCheckins(db database.Service, redisClient *redisClient.Client) {
	ctx := context.Background()
	_, err := db.GetPool().Exec(ctx, "TRUNCATE TABLE checkins, participants, events, users CASCADE")
	Expect(err).NotTo(HaveOccurred())
	if redisClient != nil {
		_ = redisClient.GetClient().FlushDB(ctx).Err()
	}
}
