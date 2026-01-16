//go:build integration
// +build integration

package database_test

import (
	"context"
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

var _ = Describe("ParticipantRepository", func() {
	var (
		ctx         context.Context
		log         *logger.Logger
		cfg         *config.DatabaseConfig
		db          *database.PostgresDB
		repo        repository.ParticipantRepository
		userRepo    repository.UserRepository
		eventRepo   repository.EventRepository
		eventID     uuid.UUID
		organizerID uuid.UUID
		organizer   *entity.User
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
		Expect(err).To(BeNil())

		repo = database.NewParticipantRepository(db.GetPool())
		userRepo = database.NewUserRepository(db.GetPool(), log)
		eventRepo = database.NewEventRepository(db.GetPool(), log)

		// Create an organizer for the events
		organizerID = uuid.New()
		organizer = &entity.User{
			ID:           organizerID,
			Email:        fmt.Sprintf("organizer_%s@example.com", organizerID.String()[:8]),
			PasswordHash: "hashed_password",
			Name:         "Organizer User",
			Role:         entity.RoleOrganizer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		Expect(userRepo.Create(ctx, organizer)).To(Succeed())

		// Create test event
		eventID = uuid.New()
		testEvent := &entity.Event{
			ID:          eventID,
			OrganizerID: organizerID,
			Name:        "Test Event",
			Description: "Test Description",
			StartDate:   time.Now().Add(24 * time.Hour),
			Location:    "Test Location",
			Status:      entity.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		Expect(eventRepo.Create(ctx, testEvent)).To(Succeed())
	})

	AfterEach(func() {
		if db != nil {
			pool := db.GetPool()
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE events CASCADE")
			_, _ = pool.Exec(ctx, "TRUNCATE TABLE participants CASCADE")
			db.Close()
		}
	})

	Describe("Create", func() {
		Context("with valid participant data", func() {
			It("should create a participant successfully", func() {
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				// Verify participant was created
				retrieved, err := repo.FindByID(ctx, participant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal(participant.Name))
				Expect(retrieved.Email).To(Equal(participant.Email))
			})
		})

		Context("with invalid participant data", func() {
			It("should return validation error for missing email", func() {
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with duplicate email for same event", func() {
			It("should return error for duplicate email", func() {
				email := "duplicate@example.com"

				participant1 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             email,
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_1",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant1)
				Expect(err).NotTo(HaveOccurred())

				participant2 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "Jane Doe",
					Email:             email,
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_2",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err = repo.Create(ctx, participant2)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with duplicate QR code", func() {
			It("should return error for duplicate QR code", func() {
				qrCode := "duplicate_qr_code"

				participant1 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john1@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            qrCode,
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant1)
				Expect(err).NotTo(HaveOccurred())

				participant2 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "Jane Doe",
					Email:             "jane1@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            qrCode,
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err = repo.Create(ctx, participant2)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("BulkCreate", func() {
		Context("with multiple valid participants", func() {
			It("should create all participants successfully", func() {
				participants := []*entity.Participant{
					{
						ID:                uuid.New(),
						EventID:           eventID,
						Name:              "Participant 1",
						Email:             "p1@example.com",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "qr_1",
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					},
					{
						ID:                uuid.New(),
						EventID:           eventID,
						Name:              "Participant 2",
						Email:             "p2@example.com",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "qr_2",
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					},
				}

				err := repo.BulkCreate(ctx, participants)
				Expect(err).NotTo(HaveOccurred())

				// Verify both participants were created
				for _, p := range participants {
					retrieved, err := repo.FindByID(ctx, p.ID)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Name).To(Equal(p.Name))
				}
			})
		})

		Context("with empty participants list", func() {
			It("should not return error", func() {
				err := repo.BulkCreate(ctx, []*entity.Participant{})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("FindByID", func() {
		Context("with existing participant", func() {
			It("should return the participant", func() {
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:                participantID,
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := repo.FindByID(ctx, participantID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(participantID))
				Expect(retrieved.Name).To(Equal("John Doe"))
				Expect(retrieved.Status).To(Equal(entity.ParticipantStatusConfirmed))
			})
		})

		Context("with non-existing participant", func() {
			It("should return error", func() {
				nonExistingID := uuid.New()
				_, err := repo.FindByID(ctx, nonExistingID)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FindByQRCode", func() {
		Context("with existing QR code", func() {
			It("should return the participant", func() {
				qrCode := "unique_qr_code_123"
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            qrCode,
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := repo.FindByQRCode(ctx, qrCode)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.QRCode).To(Equal(qrCode))
				Expect(retrieved.Name).To(Equal("John Doe"))
			})
		})

		Context("with non-existing QR code", func() {
			It("should return error", func() {
				_, err := repo.FindByQRCode(ctx, "non_existing_qr_code")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FindByEventID", func() {
		Context("with participants in event", func() {
			It("should return paginated participants", func() {
				for i := 0; i < 5; i++ {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           eventID,
						Name:              fmt.Sprintf("Participant %d", i),
						Email:             fmt.Sprintf("p%d@example.com", i),
						Status:            entity.ParticipantStatusTentative,
						QRCode:            fmt.Sprintf("qr_%d", i),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}
					err := repo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())
				}

				participants, total, err := repo.FindByEventID(ctx, eventID, 0, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(participants)).To(Equal(3))
				Expect(total).To(Equal(int64(5)))
			})
		})

		Context("with no participants", func() {
			It("should return empty list", func() {
				otherEventID := uuid.New()
				participants, total, err := repo.FindByEventID(ctx, otherEventID, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(participants)).To(Equal(0))
				Expect(total).To(Equal(int64(0)))
			})
		})
	})

	Describe("Update", func() {
		Context("with existing participant", func() {
			It("should update the participant", func() {
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:                participantID,
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				// Update participant
				participant.Name = "Jane Doe"
				participant.Status = entity.ParticipantStatusConfirmed
				participant.PaymentStatus = entity.PaymentPaid
				participant.UpdatedAt = time.Now()

				err = repo.Update(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := repo.FindByID(ctx, participantID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal("Jane Doe"))
				Expect(retrieved.Status).To(Equal(entity.ParticipantStatusConfirmed))
				Expect(retrieved.PaymentStatus).To(Equal(entity.PaymentPaid))
			})
		})

		Context("with non-existing participant", func() {
			It("should return error", func() {
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					UpdatedAt:         time.Now(),
				}

				err := repo.Update(ctx, participant)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Delete", func() {
		Context("with existing participant", func() {
			It("should delete the participant", func() {
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:                participantID,
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				err = repo.Delete(ctx, participantID)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = repo.FindByID(ctx, participantID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with non-existing participant", func() {
			It("should return error", func() {
				err := repo.Delete(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Search", func() {
		Context("with search results", func() {
			It("should find participants by name", func() {
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				results, total, err := repo.Search(ctx, eventID, "John", 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(Equal(1))
				Expect(total).To(Equal(int64(1)))
				Expect(results[0].Name).To(Equal("John Doe"))
			})

			It("should find participants by email", func() {
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             "john@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				results, total, err := repo.Search(ctx, eventID, "john@", 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(Equal(1))
				Expect(total).To(Equal(int64(1)))
			})
		})

		Context("with no search results", func() {
			It("should return empty list", func() {
				results, total, err := repo.Search(ctx, eventID, "NonExisting", 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(Equal(0))
				Expect(total).To(Equal(int64(0)))
			})
		})
	})

	Describe("ExistsByEmail", func() {
		Context("with existing email", func() {
			It("should return true", func() {
				email := "exists@example.com"
				participant := &entity.Participant{
					ID:                uuid.New(),
					EventID:           eventID,
					Name:              "John Doe",
					Email:             email,
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr_code_12345",
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := repo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				exists, err := repo.ExistsByEmail(ctx, eventID, email)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("with non-existing email", func() {
			It("should return false", func() {
				exists, err := repo.ExistsByEmail(ctx, eventID, "nonexists@example.com")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	Describe("GetPaymentStats", func() {
		Context("with mixed payment statuses", func() {
			It("should return correct payment statistics", func() {
				// Create participants with different payment statuses
				for i := 0; i < 3; i++ {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           eventID,
						Name:              fmt.Sprintf("Participant %d", i),
						Email:             fmt.Sprintf("p%d@example.com", i),
						Status:            entity.ParticipantStatusTentative,
						QRCode:            fmt.Sprintf("qr_%d", i),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}
					if i == 0 {
						participant.PaymentStatus = entity.PaymentPaid
						amount := 1500.00
						participant.PaymentAmount = &amount
					}
					err := repo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())
				}

				stats, err := repo.GetPaymentStats(ctx, eventID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats.TotalParticipants).To(Equal(int64(3)))
				Expect(stats.PaidParticipants).To(Equal(int64(1)))
				Expect(stats.UnpaidParticipants).To(Equal(int64(2)))
				Expect(stats.TotalPaymentAmount).To(BeNumerically("~", 1500.00))
			})
		})
	})

	Describe("HealthCheck", func() {
		Context("with valid database connection", func() {
			It("should return nil", func() {
				err := repo.HealthCheck(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
