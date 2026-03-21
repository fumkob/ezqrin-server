package participant_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newTestUsecase builds a participant.Usecase wired to the given mocks.
// Uses a fixed HMAC secret that satisfies the minimum length requirement and a
// hosting base URL so distribution URLs are populated in test assertions.
func newTestUsecase(
	participantRepo *mocks.MockParticipantRepository,
	eventRepo *mocks.MockEventRepository,
) participant.Usecase {
	nopLogger := &logger.Logger{Logger: zap.NewNop()}
	return participant.NewUsecase(
		participantRepo,
		eventRepo,
		qrcode.NewGenerator(),
		"test-hmac-secret-for-testing-only-32chars",
		"https://qr.example.com",
		"",
		nil,
		false,
		nopLogger,
	)
}

// validCreateInput returns a minimal valid CreateParticipantInput for the given eventID.
func validCreateInput(eventID uuid.UUID) participant.CreateParticipantInput {
	return participant.CreateParticipantInput{
		EventID:       eventID,
		Name:          "Alice Smith",
		Email:         "alice@example.com",
		Status:        entity.ParticipantStatusConfirmed,
		PaymentStatus: entity.PaymentUnpaid,
	}
}

// makeParticipant builds a minimal *entity.Participant that passes Validate().
func makeParticipant(id, eventID uuid.UUID) *entity.Participant {
	return &entity.Participant{
		ID:                id,
		EventID:           eventID,
		Name:              "Alice Smith",
		Email:             "alice@example.com",
		QRCode:            "dummy-qr-token",
		QRCodeGeneratedAt: time.Now(),
		Status:            entity.ParticipantStatusConfirmed,
		PaymentStatus:     entity.PaymentUnpaid,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

var _ = Describe("Create", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("creating a participant", func() {
		Context("with valid input as event organizer", func() {
			It("should create the participant and return it with QR code and distribution URL", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := validCreateInput(eventID)

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Name).To(Equal("Alice Smith"))
				Expect(result.Email).To(Equal("alice@example.com"))
				Expect(result.EventID).To(Equal(eventID))
				Expect(result.ID).NotTo(Equal(uuid.Nil))
				Expect(result.QRCode).NotTo(BeEmpty())
				Expect(result.QRDistributionURL).To(HavePrefix("https://qr.example.com/qr/"))
			})
		})

		Context("with valid input as admin (non-owner)", func() {
			It("should bypass the organizer check and create the participant", func() {
				adminID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID} // userID is the owner, not adminID
				input := validCreateInput(eventID)

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

				result, err := uc.Create(ctx, adminID, true, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
			})
		})

		Context("when the caller is neither admin nor the event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := validCreateInput(eventID)

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.Create(ctx, otherUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the event is not found", func() {
			It("should return the repository error", func() {
				notFoundErr := apperrors.NotFound("event not found")
				input := validCreateInput(eventID)

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the participant repository returns an error", func() {
			It("should return the repository error", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := validCreateInput(eventID)
				dbErr := errors.New("database connection failed")

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(dbErr)

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(dbErr))
				Expect(result).To(BeNil())
			})
		})

		Context("with invalid input (empty name)", func() {
			It("should return a validation error without calling the repository", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := validCreateInput(eventID)
				input.Name = "" // make invalid

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				// participantRepo.Create must NOT be called

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsValidation(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("with invalid input (invalid email format)", func() {
			It("should return a validation error", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := validCreateInput(eventID)
				input.Email = "not-an-email"

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsValidation(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("with optional fields populated", func() {
			It("should store employee ID, phone, and metadata on the participant", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				empID := "EMP-001"
				phone := "+819012345678"
				meta := `{"department":"engineering"}`
				amount := 5000.0
				payDate := time.Now()

				input := participant.CreateParticipantInput{
					EventID:       eventID,
					Name:          "Bob Jones",
					Email:         "bob@example.com",
					Status:        entity.ParticipantStatusTentative,
					PaymentStatus: entity.PaymentPaid,
					EmployeeID:    &empID,
					Phone:         &phone,
					Metadata:      &meta,
					PaymentAmount: &amount,
					PaymentDate:   &payDate,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

				result, err := uc.Create(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.EmployeeID).To(Equal(&empID))
				Expect(result.Phone).To(Equal(&phone))
				Expect(result.Metadata).NotTo(BeNil())
				Expect(result.PaymentAmount).To(Equal(&amount))
			})
		})
	})
})

var _ = Describe("BulkCreate", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("bulk creating participants", func() {
		Context("with two valid entries as the event organizer", func() {
			It("should create both participants and return CreatedCount=2", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := participant.BulkCreateInput{
					EventID: eventID,
					Participants: []participant.CreateParticipantInput{
						{
							EventID:       eventID,
							Name:          "Alice",
							Email:         "alice@example.com",
							Status:        entity.ParticipantStatusConfirmed,
							PaymentStatus: entity.PaymentUnpaid,
						},
						{
							EventID:       eventID,
							Name:          "Bob",
							Email:         "bob@example.com",
							Status:        entity.ParticipantStatusConfirmed,
							PaymentStatus: entity.PaymentUnpaid,
						},
					},
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil).Times(2)

				output, err := uc.BulkCreate(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.CreatedCount).To(Equal(2))
				Expect(output.FailedCount).To(Equal(0))
				Expect(output.Participants).To(HaveLen(2))
				Expect(output.Errors).To(BeEmpty())
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error before processing participants", func() {
				otherUserID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := participant.BulkCreateInput{
					EventID: eventID,
					Participants: []participant.CreateParticipantInput{
						validCreateInput(eventID),
					},
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.BulkCreate(ctx, otherUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(output).To(Equal(participant.BulkCreateOutput{}))
			})
		})

		Context("when admin creates participants for another organizer's event", func() {
			It("should bypass the organizer check", func() {
				adminID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := participant.BulkCreateInput{
					EventID:      eventID,
					Participants: []participant.CreateParticipantInput{validCreateInput(eventID)},
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

				output, err := uc.BulkCreate(ctx, adminID, true, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.CreatedCount).To(Equal(1))
			})
		})

		Context("when the event is not found", func() {
			It("should return the repository error", func() {
				notFoundErr := apperrors.NotFound("event not found")
				input := participant.BulkCreateInput{
					EventID:      eventID,
					Participants: []participant.CreateParticipantInput{validCreateInput(eventID)},
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				output, err := uc.BulkCreate(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(output).To(Equal(participant.BulkCreateOutput{}))
			})
		})

		Context("when one entry is invalid (empty name) and one is valid", func() {
			It("should record a failure for the invalid entry and succeed for the valid one", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				invalidInput := participant.CreateParticipantInput{
					EventID:       eventID,
					Name:          "", // invalid
					Email:         "invalid@example.com",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}
				validInput := participant.CreateParticipantInput{
					EventID:       eventID,
					Name:          "Valid Person",
					Email:         "valid@example.com",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}

				input := participant.BulkCreateInput{
					EventID:      eventID,
					Participants: []participant.CreateParticipantInput{invalidInput, validInput},
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				// Only the valid entry reaches the repository
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil).Times(1)

				output, err := uc.BulkCreate(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.CreatedCount).To(Equal(1))
				Expect(output.FailedCount).To(Equal(1))
				Expect(output.Errors).To(HaveLen(1))
				Expect(output.Errors[0].Index).To(Equal(0))
				Expect(output.Errors[0].Email).To(Equal("invalid@example.com"))
			})
		})

		Context("when SkipDuplicates is true and a conflict error occurs", func() {
			It("should skip the duplicate and not count it as a failure", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				duplicateInput := validCreateInput(eventID)
				conflictErr := apperrors.Conflict("email already exists")

				input := participant.BulkCreateInput{
					EventID:        eventID,
					Participants:   []participant.CreateParticipantInput{duplicateInput},
					SkipDuplicates: true,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(conflictErr)

				output, err := uc.BulkCreate(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.CreatedCount).To(Equal(0))
				Expect(output.FailedCount).To(Equal(0))
				Expect(output.SkippedCount).To(Equal(1))
				Expect(output.SkippedRows).To(HaveLen(1))
			})
		})

		Context("when SkipDuplicates is false and a conflict error occurs", func() {
			It("should record it as a failure", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				duplicateInput := validCreateInput(eventID)
				conflictErr := apperrors.Conflict("email already exists")

				input := participant.BulkCreateInput{
					EventID:        eventID,
					Participants:   []participant.CreateParticipantInput{duplicateInput},
					SkipDuplicates: false,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Create(ctx, gomock.Any()).Return(conflictErr)

				output, err := uc.BulkCreate(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.FailedCount).To(Equal(1))
				Expect(output.SkippedCount).To(Equal(0))
			})
		})
	})
})

var _ = Describe("Delete", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
		participantID   uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
		participantID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("deleting a participant", func() {
		Context("as the event organizer", func() {
			It("should delete the participant successfully", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Delete(ctx, participantID).Return(nil)

				err := uc.Delete(ctx, userID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("as admin (non-owner)", func() {
			It("should bypass the organizer check and delete successfully", func() {
				adminID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Delete(ctx, participantID).Return(nil)

				err := uc.Delete(ctx, adminID, true, participantID)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				err := uc.Delete(ctx, otherUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
			})
		})

		Context("when the participant is not found", func() {
			It("should return the not-found error", func() {
				notFoundErr := apperrors.NotFound("participant not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(nil, notFoundErr)

				err := uc.Delete(ctx, userID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("when the event is not found", func() {
			It("should return the not-found error", func() {
				p := makeParticipant(participantID, eventID)
				notFoundErr := apperrors.NotFound("event not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				err := uc.Delete(ctx, userID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("when the repository delete call fails", func() {
			It("should return the repository error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				dbErr := errors.New("constraint violation")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Delete(ctx, participantID).Return(dbErr)

				err := uc.Delete(ctx, userID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(dbErr))
			})
		})
	})
})

var _ = Describe("Update", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
		participantID   uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
		participantID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("updating a participant", func() {
		Context("with a valid name change as the event organizer", func() {
			It("should return the updated participant with the new name", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				newName := "Alice Updated"
				input := participant.UpdateParticipantInput{Name: &newName}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Name).To(Equal("Alice Updated"))
			})
		})

		Context("with only payment fields changed", func() {
			It("should update payment status and amount without touching other fields", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				newStatus := entity.PaymentPaid
				newAmount := 9800.0
				input := participant.UpdateParticipantInput{
					PaymentStatus: &newStatus,
					PaymentAmount: &newAmount,
				}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.PaymentStatus).To(Equal(entity.PaymentPaid))
				Expect(result.PaymentAmount).To(Equal(&newAmount))
				// unchanged field
				Expect(result.Name).To(Equal("Alice Smith"))
			})
		})

		Context("with metadata update", func() {
			It("should set the metadata JSON on the returned participant", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				meta := `{"role":"speaker"}`
				input := participant.UpdateParticipantInput{Metadata: &meta}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Metadata).NotTo(BeNil())
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				newName := "Hacker"
				input := participant.UpdateParticipantInput{Name: &newName}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.Update(ctx, otherUserID, false, participantID, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the participant is not found", func() {
			It("should return the not-found error", func() {
				notFoundErr := apperrors.NotFound("participant not found")
				input := participant.UpdateParticipantInput{}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(nil, notFoundErr)

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the update would produce an invalid participant (empty name)", func() {
			It("should return a validation error without calling the repository update", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				emptyName := ""
				input := participant.UpdateParticipantInput{Name: &emptyName}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				// participantRepo.Update must NOT be called

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsValidation(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the repository update call fails", func() {
			It("should return the repository error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				newName := "Alice Updated"
				input := participant.UpdateParticipantInput{Name: &newName}
				dbErr := errors.New("deadlock detected")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Update(ctx, gomock.Any()).Return(dbErr)

				result, err := uc.Update(ctx, userID, false, participantID, input)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(dbErr))
				Expect(result).To(BeNil())
			})
		})

		Context("as admin updating another organizer's participant", func() {
			It("should bypass the organizer check and return the updated participant", func() {
				adminID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				newName := "Admin Updated Name"
				input := participant.UpdateParticipantInput{Name: &newName}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

				result, err := uc.Update(ctx, adminID, true, participantID, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Name).To(Equal("Admin Updated Name"))
			})
		})
	})
})

var _ = Describe("GetByID", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
		participantID   uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
		participantID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("retrieving a participant by ID", func() {
		Context("as the event organizer", func() {
			It("should return the participant with the distribution URL populated", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.GetByID(ctx, userID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.ID).To(Equal(participantID))
				Expect(result.QRDistributionURL).To(HavePrefix("https://qr.example.com/qr/"))
			})
		})

		Context("as admin (non-owner)", func() {
			It("should bypass the organizer check and return the participant", func() {
				adminID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.GetByID(ctx, adminID, true, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				result, err := uc.GetByID(ctx, otherUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the participant is not found", func() {
			It("should return the not-found error", func() {
				notFoundErr := apperrors.NotFound("participant not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(nil, notFoundErr)

				result, err := uc.GetByID(ctx, userID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})

		Context("when the event is not found", func() {
			It("should return the not-found error", func() {
				p := makeParticipant(participantID, eventID)
				notFoundErr := apperrors.NotFound("event not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				result, err := uc.GetByID(ctx, userID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(result).To(BeNil())
			})
		})
	})
})

var _ = Describe("List", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("listing participants", func() {
		Context("as the event organizer with no search query", func() {
			It("should return a paginated list with distribution URLs populated", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				participants := []*entity.Participant{
					makeParticipant(uuid.New(), eventID),
					makeParticipant(uuid.New(), eventID),
				}
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 10,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				// Page=1, PerPage=10 → offset=0, limit=10
				participantRepo.EXPECT().FindByEventID(ctx, eventID, 0, 10).Return(participants, int64(2), nil)

				output, err := uc.List(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.TotalCount).To(Equal(int64(2)))
				Expect(output.Participants).To(HaveLen(2))
				for _, p := range output.Participants {
					Expect(p.QRDistributionURL).To(HavePrefix("https://qr.example.com/qr/"))
				}
			})
		})

		Context("with a non-empty search query", func() {
			It("should delegate to the Search repository method", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				participants := []*entity.Participant{makeParticipant(uuid.New(), eventID)}
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 5,
					Search:  "Alice",
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().Search(ctx, eventID, "Alice", 0, 5).Return(participants, int64(1), nil)

				output, err := uc.List(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.TotalCount).To(Equal(int64(1)))
				Expect(output.Participants).To(HaveLen(1))
			})
		})

		Context("with a status filter", func() {
			It("should return only participants matching the status", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				confirmed := makeParticipant(uuid.New(), eventID)
				confirmed.Status = entity.ParticipantStatusConfirmed
				tentative := makeParticipant(uuid.New(), eventID)
				tentative.Status = entity.ParticipantStatusTentative
				statusFilter := entity.ParticipantStatusConfirmed
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 10,
					Status:  &statusFilter,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByEventID(ctx, eventID, 0, 10).
					Return([]*entity.Participant{confirmed, tentative}, int64(2), nil)

				output, err := uc.List(ctx, userID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.Participants).To(HaveLen(1))
				Expect(output.Participants[0].Status).To(Equal(entity.ParticipantStatusConfirmed))
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 10,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.List(ctx, otherUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(output).To(Equal(participant.ListParticipantsOutput{}))
			})
		})

		Context("when the event is not found", func() {
			It("should return the not-found error", func() {
				notFoundErr := apperrors.NotFound("event not found")
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 10,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				output, err := uc.List(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(output).To(Equal(participant.ListParticipantsOutput{}))
			})
		})

		Context("when the FindByEventID repository call fails", func() {
			It("should return the repository error", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				dbErr := errors.New("connection reset by peer")
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    2,
					PerPage: 20,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				// Page=2, PerPage=20 → offset=20, limit=20
				participantRepo.EXPECT().FindByEventID(ctx, eventID, 20, 20).Return(nil, int64(0), dbErr)

				output, err := uc.List(ctx, userID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(dbErr))
				Expect(output).To(Equal(participant.ListParticipantsOutput{}))
			})
		})

		Context("as admin listing participants for another organizer's event", func() {
			It("should bypass the organizer check and return the list", func() {
				adminID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: userID}
				participants := []*entity.Participant{makeParticipant(uuid.New(), eventID)}
				input := participant.ListParticipantsInput{
					EventID: eventID,
					Page:    1,
					PerPage: 10,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByEventID(ctx, eventID, 0, 10).Return(participants, int64(1), nil)

				output, err := uc.List(ctx, adminID, true, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.Participants).To(HaveLen(1))
			})
		})
	})
})

var _ = Describe("GetQRCode", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
		participantID   uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		uc = newTestUsecase(participantRepo, eventRepo)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
		participantID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("requesting a QR code", func() {
		Context("in PNG format as the event organizer", func() {
			It("should return PNG data with correct content type and filename", func() {
				p := makeParticipant(participantID, eventID)
				p.Name = "Alice Smith"
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 300)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.ContentType).To(Equal("image/png"))
				Expect(output.Filename).To(Equal("participant-Alice Smith-qr.png"))
				Expect(output.Data).NotTo(BeEmpty())
			})
		})

		Context("in SVG format as the event organizer", func() {
			It("should return SVG data with correct content type and filename", func() {
				p := makeParticipant(participantID, eventID)
				p.Name = "Bob Jones"
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "svg", 256)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.ContentType).To(Equal("image/svg+xml"))
				Expect(output.Filename).To(Equal("participant-Bob Jones-qr.svg"))
				Expect(output.Data).NotTo(BeEmpty())
			})
		})

		Context("as admin (non-owner)", func() {
			It("should bypass the organizer check and return the QR code", func() {
				adminID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, adminID, true, participantID, "png", 200)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.ContentType).To(Equal("image/png"))
			})
		})

		Context("when the caller is neither admin nor event organizer", func() {
			It("should return a Forbidden error", func() {
				otherUserID := uuid.New()
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, otherUserID, false, participantID, "png", 300)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsForbidden(err)).To(BeTrue())
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})

		Context("with an unsupported format", func() {
			It("should return a bad request error after authorization passes", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "pdf", 300)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid format"))
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})

		Context("with PNG format and size below the minimum (99)", func() {
			It("should return a bad request error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 99)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid size"))
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})

		Context("with PNG format and size above the maximum (2001)", func() {
			It("should return a bad request error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 2001)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid size"))
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})

		Context("with PNG format and the boundary size values", func() {
			It("should accept size=100 (minimum) without error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 100)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.ContentType).To(Equal("image/png"))
			})

			It("should accept size=2000 (maximum) without error", func() {
				p := makeParticipant(participantID, eventID)
				event := &entity.Event{ID: eventID, OrganizerID: userID}

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 2000)

				Expect(err).NotTo(HaveOccurred())
				Expect(output.ContentType).To(Equal("image/png"))
			})
		})

		Context("when the participant is not found", func() {
			It("should return the not-found error", func() {
				notFoundErr := apperrors.NotFound("participant not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(nil, notFoundErr)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 300)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})

		Context("when the event is not found", func() {
			It("should return the not-found error", func() {
				p := makeParticipant(participantID, eventID)
				notFoundErr := apperrors.NotFound("event not found")

				participantRepo.EXPECT().FindByID(ctx, participantID).Return(p, nil)
				eventRepo.EXPECT().FindByID(ctx, eventID).Return(nil, notFoundErr)

				output, err := uc.GetQRCode(ctx, userID, false, participantID, "png", 300)

				Expect(err).To(HaveOccurred())
				Expect(apperrors.IsNotFound(err)).To(BeTrue())
				Expect(output).To(Equal(participant.QRCodeOutput{}))
			})
		})
	})
})
