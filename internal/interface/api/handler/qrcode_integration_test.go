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

var _ = Describe("QRCode API Integration", func() {
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
				Secret:                   jwtSecret,
				AccessTokenExpiry:        time.Hour,
				RefreshTokenExpiryWeb:    7 * 24 * time.Hour,
				RefreshTokenExpiryMobile: 90 * 24 * time.Hour,
			},
			QRCode: config.QRCodeConfig{
				HMACSecret: "test-hmac-secret-minimum-32-characters-long-for-testing",
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

		cleanDatabaseForQRCodes(db, redis)

		appContainer, err := container.NewContainer(cfg, log, db, cacheService)
		Expect(err).NotTo(HaveOccurred())

		router = api.SetupRouter(&api.RouterDependencies{
			Config:    cfg,
			Logger:    log,
			DB:        db,
			Cache:     cacheService,
			Container: appContainer,
		})

		createTestUserV1(router, "qr-organizer@example.com", "Password123!", "QR Organizer", "organizer")
		organizerAuth = loginTestUserV1(router, "qr-organizer@example.com", "Password123!")

		createTestUserV1(router, "qr-admin@example.com", "Password123!", "QR Admin", "admin")
		adminAuth = loginTestUserV1(router, "qr-admin@example.com", "Password123!")

		eventReq := map[string]interface{}{
			"name":        "Test Event for QR Codes",
			"description": "A test event for QR code email sending",
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

		participant1 = createTestParticipant(
			router,
			testEventID,
			organizerAuth.AccessToken,
			"QR Participant 1",
			"qr-participant1@example.com",
		)
		participant2 = createTestParticipant(
			router,
			testEventID,
			organizerAuth.AccessToken,
			"QR Participant 2",
			"qr-participant2@example.com",
		)
	})

	AfterEach(func() {
		if db != nil && redis != nil {
			cleanDatabaseForQRCodes(db, redis)
		}
		if db != nil {
			db.Close()
		}
		if redis != nil {
			redis.Close()
		}
	})

	Describe("POST /api/v1/events/:eventId/qrcodes/send", func() {
		When("sending QR codes to specific participants", func() {
			Context("with valid organizer token and existing participants", func() {
				It("should accept the request and return a valid SendQRCodesResponse structure", func() {
					sendReq := map[string]interface{}{
						"participant_ids": []string{
							participant1.Id.String(),
							participant2.Id.String(),
						},
					}

					reqBody, _ := json.Marshal(sendReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					// 200 (all failed due to no SMTP) or 207 (partial) are both acceptable;
					// in a test environment without SMTP, expect 200 with all failures.
					Expect(w.Code).To(BeElementOf(http.StatusOK, http.StatusMultiStatus))

					var response generated.SendQRCodesResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.Total).To(Equal(2))
					Expect(response.SentCount + response.FailedCount).To(Equal(response.Total))
				})

				It("should report FailedCount equal to Total when SMTP is unavailable", func() {
					sendReq := map[string]interface{}{
						"participant_ids": []string{
							participant1.Id.String(),
						},
					}

					reqBody, _ := json.Marshal(sendReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var response generated.SendQRCodesResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.Total).To(Equal(1))
					// In a test environment without SMTP, all emails fail
					Expect(response.FailedCount).To(Equal(response.Total))
					Expect(response.SentCount).To(Equal(0))
					Expect(response.Failures).To(HaveLen(1))
				})
			})
		})

		When("sending QR codes with send_to_all", func() {
			Context("with valid organizer token", func() {
				It("should accept the request even though email delivery fails", func() {
					sendToAll := true
					sendReq := map[string]interface{}{
						"send_to_all": sendToAll,
					}

					reqBody, _ := json.Marshal(sendReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(BeElementOf(http.StatusOK, http.StatusMultiStatus))

					var response generated.SendQRCodesResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					// Both participants should be attempted
					Expect(response.Total).To(Equal(2))
					Expect(response.SentCount + response.FailedCount).To(Equal(response.Total))
				})
			})
		})

		When("unauthenticated", func() {
			It("should return 401 Unauthorized", func() {
				sendReq := map[string]interface{}{
					"participant_ids": []string{participant1.Id.String()},
				}

				reqBody, _ := json.Marshal(sendReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/qrcodes/send",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				// No Authorization header

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		When("event does not exist", func() {
			It("should return 404 Not Found", func() {
				sendReq := map[string]interface{}{
					"participant_ids": []string{participant1.Id.String()},
				}

				reqBody, _ := json.Marshal(sendReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/00000000-0000-0000-0000-000000000000/qrcodes/send",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})

		When("user is not the event owner", func() {
			It("should return 403 Forbidden", func() {
				createTestUserV1(router, "qr-other@example.com", "Password123!", "Other User", "organizer")
				otherAuth := loginTestUserV1(router, "qr-other@example.com", "Password123!")

				sendReq := map[string]interface{}{
					"participant_ids": []string{participant1.Id.String()},
				}

				reqBody, _ := json.Marshal(sendReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/qrcodes/send",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+otherAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})

		When("admin sends QR codes for another user's event", func() {
			It("should be authorized and not return 403", func() {
				sendReq := map[string]interface{}{
					"participant_ids": []string{participant1.Id.String()},
				}

				reqBody, _ := json.Marshal(sendReq)
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/events/"+testEventID+"/qrcodes/send",
					bytes.NewReader(reqBody),
				)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+adminAuth.AccessToken)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(BeElementOf(http.StatusOK, http.StatusMultiStatus))
			})
		})

		When("request body is invalid", func() {
			Context("with missing body (no participant_ids and no send_to_all)", func() {
				It("should return 400 Bad Request", func() {
					emptyReq := map[string]interface{}{}

					reqBody, _ := json.Marshal(emptyReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})

			Context("with invalid JSON", func() {
				It("should return 400 Bad Request", func() {
					invalidJSON := []byte("{invalid json}")
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(invalidJSON),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		When("email_template is specified", func() {
			Context("with the 'detailed' template", func() {
				It("should accept the request with the specified template", func() {
					sendReq := map[string]interface{}{
						"participant_ids": []string{participant1.Id.String()},
						"email_template":  "detailed",
					}

					reqBody, _ := json.Marshal(sendReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(BeElementOf(http.StatusOK, http.StatusMultiStatus))

					var response generated.SendQRCodesResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.Total).To(Equal(1))
				})
			})

			Context("with the 'minimal' template", func() {
				It("should accept the request with the specified template", func() {
					sendReq := map[string]interface{}{
						"participant_ids": []string{participant2.Id.String()},
						"email_template":  "minimal",
					}

					reqBody, _ := json.Marshal(sendReq)
					req := httptest.NewRequest(
						http.MethodPost,
						"/api/v1/events/"+testEventID+"/qrcodes/send",
						bytes.NewReader(reqBody),
					)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+organizerAuth.AccessToken)

					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					Expect(w.Code).To(BeElementOf(http.StatusOK, http.StatusMultiStatus))

					var response generated.SendQRCodesResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.Total).To(Equal(1))
				})
			})
		})
	})
})

// cleanDatabaseForQRCodes cleans all test data from the database.
func cleanDatabaseForQRCodes(db database.Service, redisClient *redisClient.Client) {
	ctx := context.Background()
	_, err := db.GetPool().Exec(ctx, "TRUNCATE TABLE checkins, participants, events, users CASCADE")
	Expect(err).NotTo(HaveOccurred())
	if redisClient != nil {
		_ = redisClient.GetClient().FlushDB(ctx).Err()
	}
}
