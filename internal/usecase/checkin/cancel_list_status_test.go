package checkin_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/usecase/checkin"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Cancel UseCase", func() {
	var (
		ctrl            *gomock.Controller
		ctx             context.Context
		uc              checkin.Usecase
		mockCheckinRepo *mocks.MockCheckinRepository
		mockParticipant *mocks.MockParticipantRepository
		mockEventRepo   *mocks.MockEventRepository
		testEventID     uuid.UUID
		testUserID      uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		testEventID = uuid.New()
		testUserID = uuid.New()

		mockCheckinRepo = mocks.NewMockCheckinRepository(ctrl)
		mockParticipant = mocks.NewMockParticipantRepository(ctrl)
		mockEventRepo = mocks.NewMockEventRepository(ctrl)

		uc = checkin.NewUsecase(mockCheckinRepo, mockParticipant, mockEventRepo, testQRHMACSecret)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Cancel", func() {
		When("the event associated with the check-in does not exist", func() {
			It("should return the error from the event repository", func() {
				checkinID := uuid.New()
				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: uuid.New(),
					CheckedInAt:   time.Now(),
					Method:        entity.CheckinMethodManual,
				}

				mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).Return(checkinRecord, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).
					Return(nil, apperrors.NotFound("event not found"))

				err := uc.Cancel(ctx, testUserID, false, checkinID)

				Expect(err).To(HaveOccurred())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeNotFound))
			})
		})

		When("the repository fails to delete the check-in", func() {
			It("should return a wrapped error", func() {
				checkinID := uuid.New()
				checkinRecord := &entity.Checkin{
					ID:            checkinID,
					EventID:       testEventID,
					ParticipantID: uuid.New(),
					CheckedInAt:   time.Now(),
					Method:        entity.CheckinMethodManual,
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockCheckinRepo.EXPECT().FindByID(gomock.Any(), checkinID).Return(checkinRecord, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().Delete(gomock.Any(), checkinID).
					Return(errors.New("database connection failed"))

				err := uc.Cancel(ctx, testUserID, false, checkinID)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to cancel check-in"))
				Expect(err.Error()).To(ContainSubstring("database connection failed"))
			})
		})
	})
})

var _ = Describe("List UseCase", func() {
	var (
		ctrl            *gomock.Controller
		ctx             context.Context
		uc              checkin.Usecase
		mockCheckinRepo *mocks.MockCheckinRepository
		mockParticipant *mocks.MockParticipantRepository
		mockEventRepo   *mocks.MockEventRepository
		testEventID     uuid.UUID
		testUserID      uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		testEventID = uuid.New()
		testUserID = uuid.New()

		mockCheckinRepo = mocks.NewMockCheckinRepository(ctrl)
		mockParticipant = mocks.NewMockParticipantRepository(ctrl)
		mockEventRepo = mocks.NewMockEventRepository(ctrl)

		uc = checkin.NewUsecase(mockCheckinRepo, mockParticipant, mockEventRepo, testQRHMACSecret)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("List", func() {
		When("the check-in repository returns an error", func() {
			It("should return a wrapped error", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().FindByEvent(gomock.Any(), testEventID, 10, 0).
					Return(nil, int64(0), errors.New("database query failed"))

				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := uc.List(ctx, testUserID, false, input)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to list check-ins"))
				Expect(err.Error()).To(ContainSubstring("database query failed"))
			})
		})

		When("a participant associated with a check-in cannot be found", func() {
			It("should skip that check-in and return only the ones with found participants", func() {
				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				participantWithData := &entity.Participant{
					ID:      uuid.New(),
					EventID: testEventID,
					Name:    "Found User",
					Email:   "found@example.com",
				}
				missingParticipantID := uuid.New()

				checkins := []*entity.Checkin{
					{
						ID:            uuid.New(),
						EventID:       testEventID,
						ParticipantID: participantWithData.ID,
						CheckedInAt:   time.Now(),
						Method:        entity.CheckinMethodQRCode,
					},
					{
						ID:            uuid.New(),
						EventID:       testEventID,
						ParticipantID: missingParticipantID,
						CheckedInAt:   time.Now(),
						Method:        entity.CheckinMethodManual,
					},
				}

				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().FindByEvent(gomock.Any(), testEventID, 10, 0).
					Return(checkins, int64(2), nil)
				mockParticipant.EXPECT().FindByID(gomock.Any(), participantWithData.ID).
					Return(participantWithData, nil)
				mockParticipant.EXPECT().FindByID(gomock.Any(), missingParticipantID).
					Return(nil, apperrors.NotFound("participant not found"))

				input := checkin.ListCheckInsInput{
					EventID: testEventID,
					Page:    1,
					PerPage: 10,
				}

				result, err := uc.List(ctx, testUserID, false, input)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.CheckIns).To(HaveLen(1))
				Expect(result.CheckIns[0].ParticipantName).To(Equal("Found User"))
				Expect(result.TotalCount).To(Equal(int64(2)))
			})
		})
	})
})

var _ = Describe("GetStatus UseCase", func() {
	var (
		ctrl            *gomock.Controller
		ctx             context.Context
		uc              checkin.Usecase
		mockCheckinRepo *mocks.MockCheckinRepository
		mockParticipant *mocks.MockParticipantRepository
		mockEventRepo   *mocks.MockEventRepository
		testEventID     uuid.UUID
		testUserID      uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		testEventID = uuid.New()
		testUserID = uuid.New()

		mockCheckinRepo = mocks.NewMockCheckinRepository(ctrl)
		mockParticipant = mocks.NewMockParticipantRepository(ctrl)
		mockEventRepo = mocks.NewMockEventRepository(ctrl)

		uc = checkin.NewUsecase(mockCheckinRepo, mockParticipant, mockEventRepo, testQRHMACSecret)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("GetStatus", func() {
		When("the event associated with the participant does not exist", func() {
			It("should return the error from the event repository", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Test User",
					Email:   "test@example.com",
				}

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).
					Return(nil, apperrors.NotFound("event not found"))

				result, err := uc.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				var appErr *apperrors.AppError
				Expect(errors.As(err, &appErr)).To(BeTrue())
				Expect(appErr.Code).To(Equal(apperrors.CodeNotFound))
			})
		})

		When("the check-in repository returns a non-NotFound error", func() {
			It("should return a wrapped error", func() {
				participantID := uuid.New()

				participant := &entity.Participant{
					ID:      participantID,
					EventID: testEventID,
					Name:    "Test User",
					Email:   "test@example.com",
				}

				event := &entity.Event{
					ID:          testEventID,
					OrganizerID: testUserID,
					Name:        "Test Event",
				}

				mockParticipant.EXPECT().FindByID(gomock.Any(), participantID).Return(participant, nil)
				mockEventRepo.EXPECT().FindByID(gomock.Any(), testEventID).Return(event, nil)
				mockCheckinRepo.EXPECT().FindByParticipant(gomock.Any(), participantID).
					Return(nil, errors.New("unexpected database error"))

				result, err := uc.GetStatus(ctx, testUserID, false, participantID)

				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to get check-in status"))
				Expect(err.Error()).To(ContainSubstring("unexpected database error"))
			})
		})
	})
})
