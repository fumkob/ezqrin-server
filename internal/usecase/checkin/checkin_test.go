package checkin_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/usecase/checkin"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// testQRHMACSecret is the HMAC secret used for tests.
// It is 32+ characters long to satisfy the minimum length requirement.
const testQRHMACSecret = "test-hmac-secret-for-testing-only-32chars"

var _ = Describe("CheckIn UseCase", func() {
	var (
		ctrl            *gomock.Controller
		ctx             context.Context
		usecase         checkin.Usecase
		mockCheckinRepo *mocks.MockCheckinRepository
		mockParticipant *mocks.MockParticipantRepository
		mockEventRepo   *mocks.MockEventRepository
		testEventID     uuid.UUID
		testUserID      uuid.UUID
		testOrganizerID uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		testEventID = uuid.New()
		testUserID = uuid.New()
		testOrganizerID = uuid.New()

		mockCheckinRepo = mocks.NewMockCheckinRepository(ctrl)
		mockParticipant = mocks.NewMockParticipantRepository(ctrl)
		mockEventRepo = mocks.NewMockEventRepository(ctrl)

		usecase = checkin.NewUsecase(mockCheckinRepo, mockParticipant, mockEventRepo, testQRHMACSecret)
	})

	Describe("CheckIn", func() {
		When("checking in with QR code", func() {
			Context("with valid QR code", func() {
				It("should successfully check in participant", func() {
					qrCode, err := crypto.GenerateHMACSignedToken(testQRHMACSecret)
					Expect(err).NotTo(HaveOccurred())

					participant := &entity.Participant{
						ID:      uuid.New(),
						EventID: testEventID,
						Name:    "John Doe",
						Email:   "john@example.com",
						QRCode:  qrCode,
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().FindByQRCode(gomock.Any(), qrCode).Return(participant, nil)
					mockCheckinRepo.EXPECT().
						ExistsByParticipant(gomock.Any(), testEventID, participant.ID).
						Return(false, nil)
					mockCheckinRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &qrCode,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ParticipantID).To(Equal(participant.ID))
					Expect(result.ParticipantName).To(Equal("John Doe"))
					Expect(result.Method).To(Equal(entity.CheckinMethodQRCode))
				})
			})

			Context("with invalid QR code", func() {
				It("should return not found error", func() {
					// A non-HMAC-signed string will fail HMAC verification
					qrCode := "invalid-qr-code-not-hmac-signed"
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &qrCode,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeNotFound))
				})
			})

			Context("with missing QR code parameter", func() {
				It("should return bad request error", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
					}
					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      nil, // Missing QR code
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
				})
			})

			Context("when participant status is cancelled", func() {
				It("should return bad request error", func() {
					cancelledQR, err := crypto.GenerateHMACSignedToken(testQRHMACSecret)
					Expect(err).NotTo(HaveOccurred())

					cancelledParticipant := &entity.Participant{
						ID:      uuid.New(),
						EventID: testEventID,
						Status:  entity.ParticipantStatusCancelled,
						QRCode:  cancelledQR,
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().FindByQRCode(gomock.Any(), cancelledQR).Return(cancelledParticipant, nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &cancelledQR,
						CheckedInBy: testUserID,
					}
					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(result).To(BeNil())
					Expect(err).NotTo(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
					Expect(err.Error()).To(ContainSubstring("cannot check in"))
				})
			})

			Context("when participant status is declined", func() {
				It("should return bad request error", func() {
					declinedQR, err := crypto.GenerateHMACSignedToken(testQRHMACSecret)
					Expect(err).NotTo(HaveOccurred())

					declinedParticipant := &entity.Participant{
						ID:      uuid.New(),
						EventID: testEventID,
						Status:  entity.ParticipantStatusDeclined,
						QRCode:  declinedQR,
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().FindByQRCode(gomock.Any(), declinedQR).Return(declinedParticipant, nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodQRCode,
						QRCode:      &declinedQR,
						CheckedInBy: testUserID,
					}
					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(result).To(BeNil())
					Expect(err).NotTo(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
					Expect(err.Error()).To(ContainSubstring("cannot check in"))
				})
			})
		})

		When("checking in manually", func() {
			Context("as event organizer", func() {
				It("should successfully check in participant", func() {
					participantID := uuid.New()
					participant := &entity.Participant{
						ID:      participantID,
						EventID: testEventID,
						Name:    "Jane Smith",
						Email:   "jane@example.com",
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID, // User is the organizer
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
					mockCheckinRepo.EXPECT().
						ExistsByParticipant(gomock.Any(), testEventID, participantID).
						Return(false, nil)
					mockCheckinRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ParticipantID).To(Equal(participantID))
					Expect(result.Method).To(Equal(entity.CheckinMethodManual))
					Expect(result.CheckedInBy).NotTo(BeNil())
					Expect(*result.CheckedInBy).To(Equal(testUserID))
				})
			})

			Context("as admin", func() {
				It("should successfully check in participant", func() {
					participantID := uuid.New()
					participant := &entity.Participant{
						ID:      participantID,
						EventID: testEventID,
						Name:    "Admin Checkin",
						Email:   "admin@example.com",
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from current user
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
					mockCheckinRepo.EXPECT().
						ExistsByParticipant(gomock.Any(), testEventID, participantID).
						Return(false, nil)
					mockCheckinRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, true, input) // isAdmin = true

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
				})
			})

			Context("without permission", func() {
				It("should return forbidden error", func() {
					participantID := uuid.New()
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from current user
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: &participantID,
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input) // Not admin, not organizer

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
				})
			})

			Context("with missing participant ID", func() {
				It("should return bad request error", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
					}
					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

					input := checkin.CheckInInput{
						EventID:       testEventID,
						Method:        entity.CheckinMethodManual,
						ParticipantID: nil, // Missing participant ID
						CheckedInBy:   testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
				})
			})

			Context("with employee ID", func() {
				It("should successfully check in participant", func() {
					employeeID := "EMP001"
					participant := &entity.Participant{
						ID:         uuid.New(),
						EventID:    testEventID,
						Name:       "Employee User",
						Email:      "emp@example.com",
						EmployeeID: &employeeID,
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().
						FindByEmployeeID(gomock.Any(), testEventID, employeeID).
						Return(participant, nil)
					mockCheckinRepo.EXPECT().
						ExistsByParticipant(gomock.Any(), testEventID, participant.ID).
						Return(false, nil)
					mockCheckinRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodManual,
						EmployeeID:  &employeeID,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ParticipantID).To(Equal(participant.ID))
					Expect(result.Method).To(Equal(entity.CheckinMethodManual))
				})
			})

			Context("with invalid employee ID", func() {
				It("should return not found error", func() {
					employeeID := "INVALID999"
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}
					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockParticipant.EXPECT().
						FindByEmployeeID(gomock.Any(), testEventID, employeeID).
						Return(nil, apperrors.NotFound("participant not found"))

					input := checkin.CheckInInput{
						EventID:     testEventID,
						Method:      entity.CheckinMethodManual,
						EmployeeID:  &employeeID,
						CheckedInBy: testUserID,
					}

					result, err := usecase.CheckIn(ctx, testUserID, false, input)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeNotFound))
				})
			})
		})

		When("participant already checked in", func() {
			It("should return conflict error", func() {
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Already Checked In",
					Email:   "already@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				// Already checked in
				mockCheckinRepo.EXPECT().
					ExistsByParticipant(gomock.Any(), testEventID, participantID).
					Return(true, nil)

				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeConflict))
			})
		})

		When("participant belongs to different event", func() {
			It("should return bad request error", func() {
				differentEventID := uuid.New()
				participantID := uuid.New()
				participant := &entity.Participant{
					ID:      participantID,
					EventID: differentEventID, // Different event
					Name:    "Wrong Event",
					Email:   "wrong@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)

				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				participantID := uuid.New()

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).
					Return(nil, apperrors.NotFound("event not found"))

				input := checkin.CheckInInput{
					EventID:       testEventID,
					Method:        entity.CheckinMethodManual,
					ParticipantID: &participantID,
					CheckedInBy:   testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		When("invalid check-in method", func() {
			It("should return bad request error", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
				}
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

				input := checkin.CheckInInput{
					EventID:     testEventID,
					Method:      entity.CheckinMethod("invalid"),
					CheckedInBy: testUserID,
				}

				result, err := usecase.CheckIn(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeBadRequest))
			})
		})
	})

	Describe("GetStatus", func() {
		When("participant has checked in", func() {
			It("should return checked-in status with details", func() {
				participantID := uuid.New()
				checkinID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Checked In User",
					Email:   "checkedin@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: participantID,
					CheckedInAt:   time.Now(),
					Method:        entity.CheckinMethodQRCode,
				}

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().FindByParticipant(gomock.Any(), participantID).Return(checkinRecord, nil)

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.IsCheckedIn).To(BeTrue())
				Expect(result.CheckIn).NotTo(BeNil())
				Expect(result.CheckIn.ID).To(Equal(checkinID))
			})
		})

		When("participant has not checked in", func() {
			It("should return not checked-in status", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Not Checked In",
					Email:   "notcheckedin@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().FindByParticipant(gomock.Any(), participantID).
					Return(nil, apperrors.NotFound("check-in not found"))

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.IsCheckedIn).To(BeFalse())
				Expect(result.CheckIn).To(BeNil())
			})
		})

		When("participant does not exist", func() {
			It("should return not found error", func() {
				participantID := uuid.New()

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).
					Return(nil, apperrors.NotFound("participant not found"))

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Test User",
					Email:   "test@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

				result, err := usecase.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})
	})

	Describe("List", func() {
		When("listing check-ins for an event", func() {
			Context("with multiple check-ins", func() {
				It("should return paginated list", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					participants := []*entity.Participant{
						{ID: uuid.New(), EventID: testEventID, Name: "User 1", Email: "user1@example.com"},
						{ID: uuid.New(), EventID: testEventID, Name: "User 2", Email: "user2@example.com"},
					}

					checkins := []*entity.Checkin{
						{
							ID:            uuid.New(),
							EventID:       testEventID,
							ParticipantID: participants[0].ID,
							CheckedInAt:   time.Now(),
							Method:        entity.CheckinMethodQRCode,
						},
						{
							ID:            uuid.New(),
							EventID:       testEventID,
							ParticipantID: participants[1].ID,
							CheckedInAt:   time.Now(),
							Method:        entity.CheckinMethodManual,
						},
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockCheckinRepo.EXPECT().FindByEvent(gomock.Any(), testEventID, 10, 0).
						Return(checkins, int64(2), nil)
					mockParticipant.EXPECT().FindByID(gomock.Any(), participants[0].ID).Return(participants[0], nil)
					mockParticipant.EXPECT().FindByID(gomock.Any(), participants[1].ID).Return(participants[1], nil)

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.CheckIns).To(HaveLen(2))
					Expect(result.TotalCount).To(Equal(int64(2)))
				})
			})

			Context("with empty list", func() {
				It("should return empty list", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockCheckinRepo.EXPECT().FindByEvent(gomock.Any(), testEventID, 10, 0).
						Return([]*entity.Checkin{}, int64(0), nil)

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    1,
						PerPage: 10,
					}

					result, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.CheckIns).To(BeEmpty())
					Expect(result.TotalCount).To(Equal(int64(0)))
				})
			})

			Context("with pagination", func() {
				It("should respect page and per_page parameters", func() {
					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					// page=2, perPage=5 -> offset=(2-1)*5=5, limit=5
					mockCheckinRepo.EXPECT().FindByEvent(gomock.Any(), testEventID, 5, 5).
						Return([]*entity.Checkin{}, int64(0), nil)

					input := checkin.ListCheckInsInput{
						EventID: testEventID,
						Page:    2,
						PerPage: 5,
					}

					_, err := usecase.List(ctx, testUserID, false, input)

					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := usecase.List(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})

		When("event does not exist", func() {
			It("should return not found error", func() {
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).
					Return(nil, apperrors.NotFound("event not found"))

				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := usecase.List(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Cancel", func() {
		When("canceling a check-in", func() {
			Context("as event organizer", func() {
				It("should successfully cancel check-in", func() {
					checkinID := uuid.New()
					checkinRecord := &entity.Checkin{
						ID:            checkinID,
						EventID:       testEventID,
						ParticipantID: uuid.New(),
						CheckedInAt:   time.Now(),
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testUserID,
						Name:        "Test Event",
					}

					mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).Return(checkinRecord, nil)
					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockCheckinRepo.EXPECT().Delete(gomock.Any(), checkinID).Return(nil)

					err := usecase.Cancel(ctx, testUserID, false, checkinID)

					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("as admin", func() {
				It("should successfully cancel check-in", func() {
					checkinID := uuid.New()
					checkinRecord := &entity.Checkin{
						ID:            checkinID,
						EventID:       testEventID,
						ParticipantID: uuid.New(),
						CheckedInAt:   time.Now(),
					}

					event := &entity.Event{
						ID:          testEventID,
						OrganizerID: testOrganizerID, // Different from testUserID
						Name:        "Test Event",
					}

					mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).Return(checkinRecord, nil)
					mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
					mockCheckinRepo.EXPECT().Delete(gomock.Any(), checkinID).Return(nil)

					err := usecase.Cancel(ctx, testUserID, true, checkinID) // isAdmin = true

					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		When("check-in does not exist", func() {
			It("should return not found error", func() {
				checkinID := uuid.New()

				mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).
					Return(nil, apperrors.NotFound("check-in not found"))

				err := usecase.Cancel(ctx, testUserID, false, checkinID)

				Expect(err).To(HaveOccurred())
			})
		})

		When("user does not have permission", func() {
			It("should return forbidden error", func() {
				checkinID := uuid.New()
				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: uuid.New(),
					CheckedInAt:   time.Now(),
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testOrganizerID, // Different from testUserID
					Name:        "Test Event",
				}

				mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).Return(checkinRecord, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)

				err := usecase.Cancel(ctx, testUserID, false, checkinID) // Not admin, not organizer

				Expect(err).To(HaveOccurred())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeForbidden))
			})
		})
	})
})
