package participant_test

import (
	"context"
	"errors"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// mockEmailSender はテスト専用のインメモリ EmailSender
type mockEmailSender struct {
	sent      []domainemail.Message
	returnErr error
}

func (m *mockEmailSender) Send(_ context.Context, msg domainemail.Message) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	m.sent = append(m.sent, msg)
	return nil
}

var _ = Describe("SendQRCodes", func() {
	var (
		ctrl            *gomock.Controller
		participantRepo *mocks.MockParticipantRepository
		eventRepo       *mocks.MockEventRepository
		emailSender     *mockEmailSender
		uc              participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		emailSender = &mockEmailSender{}
		uc = participant.NewUsecase(
			participantRepo, eventRepo, qrcode.NewGenerator(),
			"test-hmac-secret-for-testing-only-32chars", "", emailSender,
		)
		ctx = context.Background()
		userID = uuid.New()
		eventID = uuid.New()
	})

	AfterEach(func() { ctrl.Finish() })

	When("sending to specific participant_ids", func() {
		Context("with valid ownership and reachable SMTP", func() {
			It("should send email to participant's primary email", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
				p := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Alice", Email: "alice@example.com",
					QRCode:        "qr-token-alice",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByID(ctx, p.ID).Return(p, nil)

				result, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
					EventID:        eventID,
					ParticipantIDs: []uuid.UUID{p.ID},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.SentCount).To(Equal(1))
				Expect(result.FailedCount).To(Equal(0))
				Expect(result.Total).To(Equal(1))
				Expect(emailSender.sent).To(HaveLen(1))
				Expect(emailSender.sent[0].To).To(Equal("alice@example.com"))
				Expect(emailSender.sent[0].Subject).To(ContainSubstring("Tech Conf"))
			})
		})

		Context("when qr_email is set", func() {
			It("should send to qr_email instead of primary email", func() {
				qrEmail := "alice-work@corp.com"
				event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
				p := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Alice", Email: "alice@example.com",
					QREmail:       &qrEmail,
					QRCode:        "qr-token-alice",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByID(ctx, p.ID).Return(p, nil)

				result, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
					EventID:        eventID,
					ParticipantIDs: []uuid.UUID{p.ID},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(emailSender.sent[0].To).To(Equal("alice-work@corp.com"))
				Expect(result.SentCount).To(Equal(1))
			})
		})

		Context("when email send fails", func() {
			It("should report failure and continue processing remaining participants", func() {
				emailSender.returnErr = errors.New("connection refused")
				event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
				p := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Alice", Email: "alice@example.com",
					QRCode:        "qr-token-alice",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByID(ctx, p.ID).Return(p, nil)

				result, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
					EventID:        eventID,
					ParticipantIDs: []uuid.UUID{p.ID},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.SentCount).To(Equal(0))
				Expect(result.FailedCount).To(Equal(1))
				Expect(result.Failures[0].Email).To(Equal("alice@example.com"))
				Expect(result.Failures[0].Reason).To(ContainSubstring("connection refused"))
			})
		})
	})

	When("called without participant_ids and send_to_all=false", func() {
		It("should return bad request error", func() {
			_, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
				EventID:   eventID,
				SendToAll: false,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("participant_ids"))
		})
	})

	When("user is not the event organizer and not admin", func() {
		It("should return forbidden error", func() {
			otherUserID := uuid.New()
			event := &entity.Event{ID: eventID, OrganizerID: otherUserID, Name: "Tech Conf"}

			eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)

			_, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
				EventID:        eventID,
				ParticipantIDs: []uuid.UUID{uuid.New()},
			})
			Expect(err).To(HaveOccurred())
		})
	})

	When("send_to_all=true", func() {
		It("should load participants via FindAllByEventID and send to all", func() {
			event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
			p1 := &entity.Participant{
				ID: uuid.New(), EventID: eventID,
				Name: "Alice", Email: "alice@example.com",
				QRCode:        "qr-alice",
				Status:        entity.ParticipantStatusConfirmed,
				PaymentStatus: entity.PaymentUnpaid,
			}
			p2 := &entity.Participant{
				ID: uuid.New(), EventID: eventID,
				Name: "Bob", Email: "bob@example.com",
				QRCode:        "qr-bob",
				Status:        entity.ParticipantStatusConfirmed,
				PaymentStatus: entity.PaymentUnpaid,
			}

			eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
			participantRepo.EXPECT().FindAllByEventID(ctx, eventID).Return([]*entity.Participant{p1, p2}, nil)

			result, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
				EventID:   eventID,
				SendToAll: true,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.SentCount).To(Equal(2))
			Expect(result.Total).To(Equal(2))
			Expect(emailSender.sent).To(HaveLen(2))
		})
	})
})
