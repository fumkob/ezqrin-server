package participant_test

import (
	"context"
	"errors"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	"github.com/fumkob/ezqrin-server/internal/usecase/participant"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// mockEmailSender is an in-memory EmailSender for tests.
// Set errorsFor[toAddress] to make Send return an error for that recipient;
// any address not present in the map succeeds.
type mockEmailSender struct {
	sent      []domainemail.Message
	errorsFor map[string]error // key: To address
}

func (m *mockEmailSender) Send(_ context.Context, msg domainemail.Message) error {
	if err, ok := m.errorsFor[msg.To]; ok {
		return err
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
		ucNoURL         participant.Usecase
		ctx             context.Context
		userID          uuid.UUID
		eventID         uuid.UUID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		participantRepo = mocks.NewMockParticipantRepository(ctrl)
		eventRepo = mocks.NewMockEventRepository(ctrl)
		emailSender = &mockEmailSender{errorsFor: map[string]error{}}
		nopLogger := &logger.Logger{Logger: zap.NewNop()}
		uc = participant.NewUsecase(
			participantRepo, eventRepo, qrcode.NewGenerator(),
			"test-hmac-secret-for-testing-only-32chars", "https://qr.example.com", emailSender, false, nopLogger,
		)
		ucNoURL = participant.NewUsecase(
			participantRepo, eventRepo, qrcode.NewGenerator(),
			"test-hmac-secret-for-testing-only-32chars", "", emailSender, false, nopLogger,
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
				participantRepo.EXPECT().FindByIDs(ctx, []uuid.UUID{p.ID}).Return([]*entity.Participant{p}, nil)

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
				Expect(emailSender.sent[0].Body).To(ContainSubstring("https://qr.example.com/"))
				Expect(emailSender.sent[0].Body).To(ContainSubstring("View QR Code"))
				Expect(emailSender.sent[0].Attachments).To(BeEmpty())
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
				participantRepo.EXPECT().FindByIDs(ctx, []uuid.UUID{p.ID}).Return([]*entity.Participant{p}, nil)

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
				event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
				p := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Alice", Email: "alice@example.com",
					QRCode:        "qr-token-alice",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}
				emailSender.errorsFor["alice@example.com"] = errors.New("connection refused")

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByIDs(ctx, []uuid.UUID{p.ID}).Return([]*entity.Participant{p}, nil)

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

		Context("when one of multiple participants fails (partial success)", func() {
			It("should record the failure and succeed for the rest", func() {
				event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
				alice := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Alice", Email: "alice@example.com",
					QRCode:        "qr-alice",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}
				bob := &entity.Participant{
					ID: uuid.New(), EventID: eventID,
					Name: "Bob", Email: "bob@example.com",
					QRCode:        "qr-bob",
					Status:        entity.ParticipantStatusConfirmed,
					PaymentStatus: entity.PaymentUnpaid,
				}
				emailSender.errorsFor["bob@example.com"] = errors.New("mailbox full")

				eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
				participantRepo.EXPECT().FindByIDs(ctx, []uuid.UUID{alice.ID, bob.ID}).
					Return([]*entity.Participant{alice, bob}, nil)

				result, err := uc.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
					EventID:        eventID,
					ParticipantIDs: []uuid.UUID{alice.ID, bob.ID},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.SentCount).To(Equal(1))
				Expect(result.FailedCount).To(Equal(1))
				Expect(result.Total).To(Equal(2))
				Expect(emailSender.sent).To(HaveLen(1))
				Expect(emailSender.sent[0].To).To(Equal("alice@example.com"))
				Expect(result.Failures[0].Email).To(Equal("bob@example.com"))
				Expect(result.Failures[0].Reason).To(ContainSubstring("mailbox full"))
			})
		})
	})

	When("QRDistributionURL is empty (no hosting base URL configured)", func() {
		It("should report failure for that participant", func() {
			event := &entity.Event{ID: eventID, OrganizerID: userID, Name: "Tech Conf"}
			p := &entity.Participant{
				ID: uuid.New(), EventID: eventID,
				Name: "Alice", Email: "alice@example.com",
				QRCode:            "qr-token-alice",
				QRDistributionURL: "", // empty — should fail
				Status:            entity.ParticipantStatusConfirmed,
				PaymentStatus:     entity.PaymentUnpaid,
			}

			eventRepo.EXPECT().FindByID(ctx, eventID).Return(event, nil)
			participantRepo.EXPECT().FindByIDs(ctx, []uuid.UUID{p.ID}).Return([]*entity.Participant{p}, nil)

			result, err := ucNoURL.SendQRCodes(ctx, userID, false, participant.SendQRCodesInput{
				EventID:        eventID,
				ParticipantIDs: []uuid.UUID{p.ID},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.SentCount).To(Equal(0))
			Expect(result.FailedCount).To(Equal(1))
			Expect(result.Failures[0].Reason).To(ContainSubstring("QRDistributionURL"))
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
