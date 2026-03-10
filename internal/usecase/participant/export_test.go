package participant_test

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ExportCSV", func() {
	var (
		ctrl            *gomock.Controller
		mockParticipant *mocks.MockParticipantRepository
		mockEvent       *mocks.MockEventRepository
		uc              participant.Usecase
		ctx             context.Context
		organizerID     uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockParticipant = mocks.NewMockParticipantRepository(ctrl)
		mockEvent = mocks.NewMockEventRepository(ctrl)
		ctx = context.Background()
		organizerID = uuid.New()
		eventID = uuid.New()

		uc = participant.NewUsecase(
			mockParticipant,
			mockEvent,
			qrcode.NewGenerator(),
			"test-hmac-secret-for-testing-only-32chars",
			"",
			nil,
		)
	})

	AfterEach(func() { ctrl.Finish() })

	When("exporting participants", func() {
		Context("as event organizer", func() {
			It("should return all participants", func() {
				event := &entity.Event{ID: eventID, OrganizerID: organizerID}
				participants := []*entity.Participant{
					{ID: uuid.New(), EventID: eventID, Name: "Alice", Email: "alice@example.com"},
				}

				mockEvent.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				mockParticipant.EXPECT().FindAllByEventID(ctx, eventID).Return(participants, nil)

				result, err := uc.ExportCSV(ctx, organizerID, false, eventID)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].Name).To(Equal("Alice"))
			})
		})

		Context("as non-owner", func() {
			It("should return Forbidden error", func() {
				otherUserID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: organizerID}

				mockEvent.EXPECT().FindByID(ctx, eventID).Return(event, nil)

				_, err := uc.ExportCSV(ctx, otherUserID, false, eventID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("permission"))
			})
		})

		Context("as admin", func() {
			It("should bypass organizer check and return all participants", func() {
				adminID := uuid.New()
				event := &entity.Event{ID: eventID, OrganizerID: organizerID}

				mockEvent.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				mockParticipant.EXPECT().FindAllByEventID(ctx, eventID).Return([]*entity.Participant{}, nil)

				result, err := uc.ExportCSV(ctx, adminID, true, eventID)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("when event does not exist", func() {
			It("should return error", func() {
				mockEvent.EXPECT().FindByID(ctx, eventID).Return(nil, apperrors.NotFound("event not found"))

				_, err := uc.ExportCSV(ctx, organizerID, false, eventID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when FindAllByEventID fails", func() {
			It("should return the repository error", func() {
				repoErr := apperrors.Internal("database error")
				event := &entity.Event{ID: eventID, OrganizerID: organizerID}

				mockEvent.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				mockParticipant.EXPECT().FindAllByEventID(ctx, eventID).Return(nil, repoErr)

				_, err := uc.ExportCSV(ctx, organizerID, false, eventID)
				Expect(err).To(MatchError(repoErr))
			})
		})
	})
})
