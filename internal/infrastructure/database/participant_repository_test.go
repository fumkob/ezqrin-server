//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestParticipantRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Participant Repository Suite")
}

var _ = Describe("ParticipantRepository", func() {
	var (
		db                *pgxpool.Pool
		participantRepo   repository.ParticipantRepository
		eventRepo         repository.EventRepository
		ctx               context.Context
		testEvent         *entity.Event
		testEventID       uuid.UUID
	)

	BeforeEach(func() {
		var err error
		db, err = setupTestDB()
		Expect(err).NotTo(HaveOccurred())

		participantRepo = NewParticipantRepository(db)
		eventRepo = NewEventRepository(db)
		ctx = context.Background()

		// Create test event
		testEventID = uuid.New()
		testEvent = &entity.Event{
			ID:          testEventID,
			OrganizerID: uuid.New(),
			Name:        "Test Event",
			Description: "Test Description",
			StartDate:   time.Now(),
			Status:      entity.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = eventRepo.Create(ctx, testEvent)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
	})

	Describe("Create", func() {
		When("creating a valid participant", func() {
			Context("with all required fields", func() {
				It("should succeed", func() {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "John Doe",
						Email:             "john@example.com",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "unique-qr-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}

					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())

					// Verify the participant was created
					retrieved, err := participantRepo.FindByID(ctx, participant.ID)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Name).To(Equal("John Doe"))
					Expect(retrieved.Email).To(Equal("john@example.com"))
				})
			})

			Context("with optional fields", func() {
				It("should succeed with phone and employee_id", func() {
					phone := "+1234567890"
					empID := "EMP123"
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "Jane Smith",
						Email:             "jane@example.com",
						Phone:             &phone,
						EmployeeID:        &empID,
						Status:            entity.ParticipantStatusConfirmed,
						QRCode:            "unique-qr-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentPaid,
						PaymentAmount:     ptrFloat(100.0),
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					}

					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())

					retrieved, err := participantRepo.FindByID(ctx, participant.ID)
					Expect(err).NotTo(HaveOccurred())
					Expect(*retrieved.Phone).To(Equal(phone))
					Expect(*retrieved.EmployeeID).To(Equal(empID))
					Expect(retrieved.PaymentStatus).To(Equal(entity.PaymentPaid))
				})
			})
		})

		When("creating a participant with invalid data", func() {
			Context("with invalid email", func() {
				It("should fail", func() {
					participant := &entity.Participant{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "Invalid User",
						Email:             "invalidemail",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "unique-qr-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
					}

					err := participantRepo.Create(ctx, participant)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("creating a participant with duplicate email in same event", func() {
			It("should fail", func() {
				participant1 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "User 1",
					Email:             "duplicate@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr-1-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := participantRepo.Create(ctx, participant1)
				Expect(err).NotTo(HaveOccurred())

				participant2 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "User 2",
					Email:             "duplicate@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr-2-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err = participantRepo.Create(ctx, participant2)
				Expect(err).To(HaveOccurred())
			})
		})

		When("creating a participant with duplicate QR code", func() {
			It("should fail", func() {
				qrCode := "unique-qr-" + uuid.New().String()

				participant1 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "User 1",
					Email:             "user1@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            qrCode,
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := participantRepo.Create(ctx, participant1)
				Expect(err).NotTo(HaveOccurred())

				participant2 := &entity.Participant{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "User 2",
					Email:             "user2@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            qrCode,
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err = participantRepo.Create(ctx, participant2)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("CreateBulk", func() {
		When("creating multiple valid participants", func() {
			It("should succeed", func() {
				participants := []*entity.Participant{
					{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "User 1",
						Email:             "bulk1@example.com",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "bulk-qr-1-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					},
					{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "User 2",
						Email:             "bulk2@example.com",
						Status:            entity.ParticipantStatusConfirmed,
						QRCode:            "bulk-qr-2-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentPaid,
						PaymentAmount:     ptrFloat(50.0),
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					},
					{
						ID:                uuid.New(),
						EventID:           testEventID,
						Name:              "User 3",
						Email:             "bulk3@example.com",
						Status:            entity.ParticipantStatusTentative,
						QRCode:            "bulk-qr-3-" + uuid.New().String(),
						QRCodeGeneratedAt: time.Now(),
						PaymentStatus:     entity.PaymentUnpaid,
						CreatedAt:         time.Now(),
						UpdatedAt:         time.Now(),
					},
				}

				count, err := participantRepo.CreateBulk(ctx, participants)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(3)))

				// Verify all were created
				for _, p := range participants {
					retrieved, err := participantRepo.FindByID(ctx, p.ID)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Email).To(Equal(p.Email))
				}
			})
		})

		When("creating empty participant list", func() {
			It("should return 0", func() {
				count, err := participantRepo.CreateBulk(ctx, []*entity.Participant{})
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(0)))
			})
		})
	})

	Describe("FindByID", func() {
		When("finding an existing participant", func() {
			It("should return the participant", func() {
				participant := createTestParticipant(testEventID)
				err := participantRepo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := participantRepo.FindByID(ctx, participant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal(participant.Name))
				Expect(retrieved.Email).To(Equal(participant.Email))
			})
		})

		When("finding non-existent participant", func() {
			It("should fail", func() {
				_, err := participantRepo.FindByID(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FindByQRCode", func() {
		When("finding a participant by QR code", func() {
			It("should return the participant", func() {
				participant := createTestParticipant(testEventID)
				err := participantRepo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := participantRepo.FindByQRCode(ctx, participant.QRCode)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(participant.ID))
			})
		})

		When("finding non-existent QR code", func() {
			It("should fail", func() {
				_, err := participantRepo.FindByQRCode(ctx, "non-existent-qr")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FindByEvent", func() {
		When("finding participants for an event", func() {
			Context("with existing participants", func() {
				It("should return all participants for the event", func() {
					p1 := createTestParticipant(testEventID)
					p2 := createTestParticipant(testEventID)
					p3 := createTestParticipant(testEventID)

					err := participantRepo.Create(ctx, p1)
					Expect(err).NotTo(HaveOccurred())
					err = participantRepo.Create(ctx, p2)
					Expect(err).NotTo(HaveOccurred())
					err = participantRepo.Create(ctx, p3)
					Expect(err).NotTo(HaveOccurred())

					participants, err := participantRepo.FindByEvent(ctx, testEventID)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(participants)).To(Equal(3))
				})
			})

			Context("with no participants", func() {
				It("should return empty slice", func() {
					participants, err := participantRepo.FindByEvent(ctx, uuid.New())
					Expect(err).NotTo(HaveOccurred())
					Expect(len(participants)).To(Equal(0))
				})
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create test participants with different statuses
			participants := []*entity.Participant{
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Alice",
					Email:             "alice@example.com",
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr-alice-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentPaid,
					PaymentAmount:     ptrFloat(100.0),
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Bob",
					Email:             "bob@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr-bob-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Charlie",
					Email:             "charlie@example.com",
					EmployeeID:        ptrString("EMP123"),
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr-charlie-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentPaid,
					PaymentAmount:     ptrFloat(75.0),
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
			}

			for _, p := range participants {
				err := participantRepo.Create(ctx, p)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		When("listing participants with basic filter", func() {
			It("should return all participants and count", func() {
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(3)))
				Expect(len(participants)).To(Equal(3))
			})
		})

		When("listing participants with status filter", func() {
			It("should return only confirmed participants", func() {
				status := entity.ParticipantStatusConfirmed
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
					Status:  &status,
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(2)))
				Expect(len(participants)).To(Equal(2))
			})
		})

		When("listing participants with payment status filter", func() {
			It("should return only paid participants", func() {
				paymentStatus := entity.PaymentPaid
				filter := repository.ParticipantListFilter{
					EventID:       testEventID,
					PaymentStatus: &paymentStatus,
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(2)))
				Expect(len(participants)).To(Equal(2))
			})
		})

		When("listing participants with search filter", func() {
			It("should find participants by name", func() {
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
					Search:  "Alice",
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(1)))
				Expect(len(participants)).To(Equal(1))
				Expect(participants[0].Name).To(Equal("Alice"))
			})

			It("should find participants by email", func() {
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
					Search:  "bob@example.com",
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(1)))
				Expect(len(participants)).To(Equal(1))
				Expect(participants[0].Email).To(Equal("bob@example.com"))
			})

			It("should find participants by employee_id", func() {
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
					Search:  "EMP123",
				}

				participants, total, err := participantRepo.List(ctx, filter, 0, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(1)))
				Expect(len(participants)).To(Equal(1))
				Expect(*participants[0].EmployeeID).To(Equal("EMP123"))
			})
		})

		When("listing with pagination", func() {
			It("should respect limit and offset", func() {
				filter := repository.ParticipantListFilter{
					EventID: testEventID,
				}

				// Get first page
				participants1, total, err := participantRepo.List(ctx, filter, 0, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(3)))
				Expect(len(participants1)).To(Equal(2))

				// Get second page
				participants2, total, err := participantRepo.List(ctx, filter, 2, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(Equal(int64(3)))
				Expect(len(participants2)).To(Equal(1))
			})
		})
	})

	Describe("Update", func() {
		When("updating an existing participant", func() {
			It("should succeed", func() {
				participant := createTestParticipant(testEventID)
				err := participantRepo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				// Update the participant
				participant.Name = "Updated Name"
				participant.Status = entity.ParticipantStatusConfirmed
				participant.UpdatedAt = time.Now()

				err = participantRepo.Update(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				// Verify the update
				retrieved, err := participantRepo.FindByID(ctx, participant.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal("Updated Name"))
				Expect(retrieved.Status).To(Equal(entity.ParticipantStatusConfirmed))
			})
		})

		When("updating non-existent participant", func() {
			It("should fail", func() {
				participant := createTestParticipant(testEventID)
				err := participantRepo.Update(ctx, participant)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Delete", func() {
		When("deleting an existing participant", func() {
			It("should succeed", func() {
				participant := createTestParticipant(testEventID)
				err := participantRepo.Create(ctx, participant)
				Expect(err).NotTo(HaveOccurred())

				err = participantRepo.Delete(ctx, participant.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = participantRepo.FindByID(ctx, participant.ID)
				Expect(err).To(HaveOccurred())
			})
		})

		When("deleting non-existent participant", func() {
			It("should fail", func() {
				err := participantRepo.Delete(ctx, uuid.New())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ExistsByEmail", func() {
		When("checking if email exists for an event", func() {
			Context("with existing email", func() {
				It("should return true", func() {
					participant := createTestParticipant(testEventID)
					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())

					exists, err := participantRepo.ExistsByEmail(ctx, testEventID, participant.Email)
					Expect(err).NotTo(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})

			Context("with non-existent email", func() {
				It("should return false", func() {
					exists, err := participantRepo.ExistsByEmail(ctx, testEventID, "nonexistent@example.com")
					Expect(err).NotTo(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})

			Context("with email from different event", func() {
				It("should return false", func() {
					participant := createTestParticipant(testEventID)
					err := participantRepo.Create(ctx, participant)
					Expect(err).NotTo(HaveOccurred())

					exists, err := participantRepo.ExistsByEmail(ctx, uuid.New(), participant.Email)
					Expect(err).NotTo(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})
		})
	})

	Describe("GetParticipantStats", func() {
		BeforeEach(func() {
			// Create participants with various statuses and payment statuses
			participants := []*entity.Participant{
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Paid Confirmed",
					Email:             "paid1@example.com",
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr-stats-1-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentPaid,
					PaymentAmount:     ptrFloat(100.0),
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Paid Confirmed 2",
					Email:             "paid2@example.com",
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr-stats-2-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentPaid,
					PaymentAmount:     ptrFloat(150.0),
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Unpaid Tentative",
					Email:             "unpaid1@example.com",
					Status:            entity.ParticipantStatusTentative,
					QRCode:            "qr-stats-3-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
				{
					ID:                uuid.New(),
					EventID:           testEventID,
					Name:              "Unpaid Declined",
					Email:             "unpaid2@example.com",
					Status:            entity.ParticipantStatusDeclined,
					QRCode:            "qr-stats-4-" + uuid.New().String(),
					QRCodeGeneratedAt: time.Now(),
					PaymentStatus:     entity.PaymentUnpaid,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
			}

			for _, p := range participants {
				err := participantRepo.Create(ctx, p)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		When("getting participant statistics", func() {
			It("should return correct stats", func() {
				stats, err := participantRepo.GetParticipantStats(ctx, testEventID)
				Expect(err).NotTo(HaveOccurred())

				Expect(stats.TotalCount).To(Equal(int64(4)))
				Expect(stats.ConfirmedCount).To(Equal(int64(2)))
				Expect(stats.TentativeCount).To(Equal(int64(1)))
				Expect(stats.DeclinedCount).To(Equal(int64(1)))
				Expect(stats.PaidCount).To(Equal(int64(2)))
				Expect(stats.UnpaidCount).To(Equal(int64(2)))
				Expect(stats.TotalPaymentAmount).To(Equal(250.0))
			})
		})

		When("getting stats for event with no participants", func() {
			It("should return zero stats", func() {
				stats, err := participantRepo.GetParticipantStats(ctx, uuid.New())
				Expect(err).NotTo(HaveOccurred())

				Expect(stats.TotalCount).To(Equal(int64(0)))
				Expect(stats.ConfirmedCount).To(Equal(int64(0)))
				Expect(stats.PaidCount).To(Equal(int64(0)))
				Expect(stats.TotalPaymentAmount).To(Equal(0.0))
			})
		})
	})

	Describe("HealthCheck", func() {
		When("checking database health", func() {
			It("should succeed", func() {
				err := participantRepo.HealthCheck(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

// Helper functions

func createTestParticipant(eventID uuid.UUID) *entity.Participant {
	return &entity.Participant{
		ID:                uuid.New(),
		EventID:           eventID,
		Name:              "Test User",
		Email:             "test-" + uuid.New().String() + "@example.com",
		Status:            entity.ParticipantStatusTentative,
		QRCode:            "qr-" + uuid.New().String(),
		QRCodeGeneratedAt: time.Now(),
		PaymentStatus:     entity.PaymentUnpaid,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}

func ptrString(v string) *string {
	return &v
}
