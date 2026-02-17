//go:build integration
// +build integration

package database_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckinRepository", func() {
	var (
		repo            repository.CheckinRepository
		eventRepo       repository.EventRepository
		participantRepo repository.ParticipantRepository
		ctx             context.Context
		log             *logger.Logger
		cfg             *config.DatabaseConfig
		db              *database.PostgresDB
		testEvent       *entity.Event
		testParticipant *entity.Participant
		testUser        *entity.User
	)

	BeforeEach(func() {
		ctx = context.Background()
		log, _ = logger.New(logger.Config{
			Level:       "info",
			Format:      "console",
			Environment: "development",
		})
		cfg = &config.DatabaseConfig{
			Host:            "postgres",
			Port:            5432,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            "ezqrin_test",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		}

		var err error
		db, err = database.NewPostgresDB(ctx, cfg, log)
		Expect(err).NotTo(HaveOccurred())

		repo = database.NewCheckinRepository(db.GetPool())
		eventRepo = database.NewEventRepository(db.GetPool(), log)
		participantRepo = database.NewParticipantRepository(db.GetPool())

		// Create test user (organizer)
		testUser = &entity.User{
			ID:           uuid.New(),
			Email:        "organizer@example.com",
			PasswordHash: "hash",
			Name:         "Test Organizer",
			Role:         entity.RoleOrganizer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		userRepo := database.NewUserRepository(db.GetPool(), log)
		err = userRepo.Create(ctx, testUser)
		Expect(err).NotTo(HaveOccurred())

		// Create test event
		testEvent = &entity.Event{
			ID:          uuid.New(),
			OrganizerID: testUser.ID,
			Name:        "Test Event",
			Description: "Test event for checkin tests",
			StartDate:   time.Now().Add(24 * time.Hour),
			Location:    "Test Location",
			Timezone:    "Asia/Tokyo",
			Status:      entity.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = eventRepo.Create(ctx, testEvent)
		Expect(err).NotTo(HaveOccurred())

		// Create test participant
		testParticipant = &entity.Participant{
			ID:                uuid.New(),
			EventID:           testEvent.ID,
			Name:              "Test Participant",
			Email:             "participant@example.com",
			QRCode:            "test-qr-code-" + uuid.New().String(),
			QRCodeGeneratedAt: time.Now(),
			Status:            entity.ParticipantStatusConfirmed,
			PaymentStatus:     entity.PaymentUnpaid,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		err = participantRepo.Create(ctx, testParticipant)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up test data
		if db != nil {
			pool := db.GetPool()
			// Use TRUNCATE for complete cleanup
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE checkins, participants, events, users CASCADE")
			db.Close()
		}
	})

	When("creating a check-in", func() {
		Context("with valid data", func() {
			It("should succeed", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				// Verify the checkin was created
				found, err := repo.FindByID(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.ID).To(Equal(checkin.ID))
				Expect(found.EventID).To(Equal(testEvent.ID))
				Expect(found.ParticipantID).To(Equal(testParticipant.ID))
				Expect(found.Method).To(Equal(entity.CheckinMethodQRCode))
			})
		})

		Context("with manual method", func() {
			It("should succeed", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodManual,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				found, err := repo.FindByID(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.Method).To(Equal(entity.CheckinMethodManual))
			})
		})

		Context("with device info", func() {
			It("should succeed", func() {
				deviceInfo := json.RawMessage(`{"os":"iOS","version":"16.0"}`)
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
					DeviceInfo:    &deviceInfo,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				found, err := repo.FindByID(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.DeviceInfo).NotTo(BeNil())
			})
		})

		Context("with nil checked_in_by (self-service)", func() {
			It("should succeed", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   nil,
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				found, err := repo.FindByID(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.CheckedInBy).To(BeNil())
			})
		})

		Context("with duplicate check-in (same event and participant)", func() {
			It("should fail with ErrCheckinAlreadyExists", func() {
				checkin1 := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin1)
				Expect(err).NotTo(HaveOccurred())

				// Try to create duplicate check-in
				checkin2 := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodManual,
				}

				err = repo.Create(ctx, checkin2)
				Expect(err).To(MatchError(entity.ErrCheckinAlreadyExists))
			})
		})

		Context("with invalid data", func() {
			It("should fail validation", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       uuid.Nil, // Invalid
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("finding check-in by ID", func() {
		Context("with existing check-in", func() {
			It("should return the check-in", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				found, err := repo.FindByID(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.ID).To(Equal(checkin.ID))
			})
		})

		Context("with non-existent check-in", func() {
			It("should return error", func() {
				_, err := repo.FindByID(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("finding check-in by participant", func() {
		Context("with participant who has checked in", func() {
			It("should return the check-in", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}

				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				found, err := repo.FindByParticipant(ctx, testParticipant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(found.ParticipantID).To(Equal(testParticipant.ID))
			})
		})

		Context("with participant who has not checked in", func() {
			It("should return error", func() {
				_, err := repo.FindByParticipant(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("finding check-ins by event", func() {
		Context("with multiple check-ins", func() {
			It("should return paginated results", func() {
				// Create multiple participants and check-ins
				for i := 0; i < 3; i++ {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           testEvent.ID,
						Name:              fmt.Sprintf("Participant %d", i),
						Email:             fmt.Sprintf("participant%d@example.com", i),
						QRCode:            "qr-code-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						Status:            entity.ParticipantStatusConfirmed,
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}
					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())

					checkin := &entity.Checkin{
						ID:            uuid.New(),
						EventID:       testEvent.ID,
						ParticipantID: participant.ID,
						CheckedInAt:   time.Now(),
						CheckedInBy:   &testUser.ID,
						Method:        entity.CheckinMethodQRCode,
					}
					err = repo.Create(ctx, checkin)
					Expect(err).NotTo(HaveOccurred())
				}

				// Find check-ins with pagination
				checkins, total, err := repo.FindByEvent(ctx, testEvent.ID, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(checkins).To(HaveLen(3))
				Expect(total).To(Equal(int64(3)))
			})
		})

		Context("with no check-ins", func() {
			It("should return empty list", func() {
				event := &entity.Event{
					ID:          uuid.New(),
					OrganizerID: testUser.ID,
					Name:        "Empty Event",
					StartDate:   time.Now().Add(24 * time.Hour),
					Location:    "Location",
					Timezone:    "Asia/Tokyo",
					Status:      entity.StatusDraft,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := eventRepo.Create(ctx, event)
				Expect(err).NotTo(HaveOccurred())

				checkins, total, err := repo.FindByEvent(ctx, event.ID, 10, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(checkins).To(BeEmpty())
				Expect(total).To(Equal(int64(0)))
			})
		})
	})

	When("getting event statistics", func() {
		Context("with some participants checked in", func() {
			It("should return correct statistics", func() {
				// Create 2 more participants (total 3)
				for i := 0; i < 2; i++ {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           testEvent.ID,
						Name:              fmt.Sprintf("Participant %d", i),
						Email:             fmt.Sprintf("stats%d@example.com", i),
						QRCode:            "qr-stats-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						Status:            entity.ParticipantStatusConfirmed,
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}
					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())
				}

				// Check in 1 participant (testParticipant)
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}
				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				stats, err := repo.GetEventStats(ctx, testEvent.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats.TotalParticipants).To(Equal(int64(3)))
				Expect(stats.CheckedInCount).To(Equal(int64(1)))
				Expect(stats.CheckinRate).To(BeNumerically("~", 33.33, 0.01))
			})
		})

		Context("with all participants checked in", func() {
			It("should return 100% check-in rate", func() {
				// Check in the test participant
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}
				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				stats, err := repo.GetEventStats(ctx, testEvent.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats.TotalParticipants).To(Equal(int64(1)))
				Expect(stats.CheckedInCount).To(Equal(int64(1)))
				Expect(stats.CheckinRate).To(Equal(100.0))
			})
		})

		Context("with no participants", func() {
			It("should return zero statistics", func() {
				event := &entity.Event{
					ID:          uuid.New(),
					OrganizerID: testUser.ID,
					Name:        "Empty Event",
					StartDate:   time.Now().Add(24 * time.Hour),
					Location:    "Location",
					Timezone:    "Asia/Tokyo",
					Status:      entity.StatusDraft,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := eventRepo.Create(ctx, event)
				Expect(err).NotTo(HaveOccurred())

				stats, err := repo.GetEventStats(ctx, event.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats.TotalParticipants).To(Equal(int64(0)))
				Expect(stats.CheckedInCount).To(Equal(int64(0)))
				Expect(stats.CheckinRate).To(Equal(0.0))
			})
		})
	})

	When("checking if participant has checked in", func() {
		Context("with participant who has checked in", func() {
			It("should return true", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}
				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				exists, err := repo.ExistsByParticipant(ctx, testEvent.ID, testParticipant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("with participant who has not checked in", func() {
			It("should return false", func() {
				exists, err := repo.ExistsByParticipant(ctx, testEvent.ID, testParticipant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	When("deleting a check-in (undo)", func() {
		Context("with existing check-in", func() {
			It("should succeed", func() {
				checkin := &entity.Checkin{
					ID:            uuid.New(),
					EventID:       testEvent.ID,
					ParticipantID: testParticipant.ID,
					CheckedInAt:   time.Now(),
					CheckedInBy:   &testUser.ID,
					Method:        entity.CheckinMethodQRCode,
				}
				err := repo.Create(ctx, checkin)
				Expect(err).NotTo(HaveOccurred())

				err = repo.Delete(ctx, checkin.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify the check-in was deleted
				_, err = repo.FindByID(ctx, checkin.ID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with non-existent check-in", func() {
			It("should return error", func() {
				err := repo.Delete(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("performing health check", func() {
		Context("with healthy database connection", func() {
			It("should succeed", func() {
				err := repo.HealthCheck(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
