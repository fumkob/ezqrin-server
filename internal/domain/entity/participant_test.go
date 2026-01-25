package entity_test

import (
	"encoding/json"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Participant Entity", func() {
	var (
		eventID     uuid.UUID
		participant *entity.Participant
	)

	BeforeEach(func() {
		eventID = uuid.New()
		participant = &entity.Participant{
			ID:                uuid.New(),
			EventID:           eventID,
			Name:              "John Doe",
			Email:             "john@example.com",
			Status:            entity.ParticipantStatusTentative,
			QRCode:            "abc123xyz789",
			QRCodeGeneratedAt: time.Now(),
			PaymentStatus:     entity.PaymentUnpaid,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
	})

	Describe("Validate", func() {
		Context("with valid participant data", func() {
			It("should return no error", func() {
				err := participant.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with missing event ID", func() {
			It("should return entity.ErrParticipantEventIDRequired", func() {
				participant.EventID = uuid.Nil
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantEventIDRequired))
			})
		})

		Context("with missing name", func() {
			It("should return entity.ErrParticipantNameRequired", func() {
				participant.Name = ""
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantNameRequired))
			})
		})

		Context("with name exceeding max length", func() {
			It("should return entity.ErrParticipantNameTooLong", func() {
				participant.Name = string(make([]byte, entity.ParticipantNameMaxLength+1))
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantNameTooLong))
			})
		})

		Context("with missing email", func() {
			It("should return entity.ErrParticipantEmailRequired", func() {
				participant.Email = ""
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantEmailRequired))
			})
		})

		Context("with invalid email format", func() {
			It("should return entity.ErrParticipantEmailInvalid", func() {
				participant.Email = "invalid-email"
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantEmailInvalid))
			})
		})

		Context("with missing QR code", func() {
			It("should return entity.ErrParticipantQRCodeRequired", func() {
				participant.QRCode = ""
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantQRCodeRequired))
			})
		})

		Context("with invalid status", func() {
			It("should return entity.ErrParticipantStatusInvalid", func() {
				participant.Status = entity.ParticipantStatus("invalid")
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantStatusInvalid))
			})
		})

		Context("with phone exceeding max length", func() {
			It("should return entity.ErrParticipantPhoneTooLong", func() {
				phone := string(make([]byte, entity.ParticipantPhoneMaxLength+1))
				participant.Phone = &phone
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantPhoneTooLong))
			})
		})

		Context("with employee ID exceeding max length", func() {
			It("should return entity.ErrParticipantEmployeeIDTooLong", func() {
				empID := string(make([]byte, entity.ParticipantEmployeeIDMaxLength+1))
				participant.EmployeeID = &empID
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantEmployeeIDTooLong))
			})
		})

		Context("with invalid payment status", func() {
			It("should return entity.ErrParticipantPaymentStatusInvalid", func() {
				participant.PaymentStatus = entity.PaymentStatus("invalid")
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantPaymentStatusInvalid))
			})
		})

		Context("with metadata exceeding max size", func() {
			It("should return entity.ErrParticipantMetadataTooLarge", func() {
				largeMeta := json.RawMessage(string(make([]byte, entity.MaxMetadataSize+1)))
				participant.Metadata = &largeMeta
				err := participant.Validate()
				Expect(err).To(Equal(entity.ErrParticipantMetadataTooLarge))
			})
		})
	})

	Describe("Status checks", func() {
		Context("when status is tentative", func() {
			BeforeEach(func() {
				participant.Status = entity.ParticipantStatusTentative
			})

			It("should return true for IsTentative", func() {
				Expect(participant.IsTentative()).To(BeTrue())
			})

			It("should return false for IsConfirmed", func() {
				Expect(participant.IsConfirmed()).To(BeFalse())
			})
		})

		Context("when status is confirmed", func() {
			BeforeEach(func() {
				participant.Status = entity.ParticipantStatusConfirmed
			})

			It("should return true for IsConfirmed", func() {
				Expect(participant.IsConfirmed()).To(BeTrue())
			})

			It("should return false for IsTentative", func() {
				Expect(participant.IsTentative()).To(BeFalse())
			})
		})

		Context("when status is cancelled", func() {
			BeforeEach(func() {
				participant.Status = entity.ParticipantStatusCancelled
			})

			It("should return true for IsCancelled", func() {
				Expect(participant.IsCancelled()).To(BeTrue())
			})
		})

		Context("when status is declined", func() {
			BeforeEach(func() {
				participant.Status = entity.ParticipantStatusDeclined
			})

			It("should return true for IsDeclined", func() {
				Expect(participant.IsDeclined()).To(BeTrue())
			})
		})
	})

	Describe("Payment status checks", func() {
		Context("when payment status is paid", func() {
			BeforeEach(func() {
				participant.PaymentStatus = entity.PaymentPaid
			})

			It("should return true for IsPaid", func() {
				Expect(participant.IsPaid()).To(BeTrue())
			})

			It("should return false for IsUnpaid", func() {
				Expect(participant.IsUnpaid()).To(BeFalse())
			})
		})

		Context("when payment status is unpaid", func() {
			BeforeEach(func() {
				participant.PaymentStatus = entity.PaymentUnpaid
			})

			It("should return true for IsUnpaid", func() {
				Expect(participant.IsUnpaid()).To(BeTrue())
			})

			It("should return false for IsPaid", func() {
				Expect(participant.IsPaid()).To(BeFalse())
			})
		})
	})

	Describe("Valid status and payment status checks", func() {
		Context("with valid status values", func() {
			It("should return true for tentative", func() {
				participant.Status = entity.ParticipantStatusTentative
				Expect(participant.IsValidStatus()).To(BeTrue())
			})

			It("should return true for confirmed", func() {
				participant.Status = entity.ParticipantStatusConfirmed
				Expect(participant.IsValidStatus()).To(BeTrue())
			})

			It("should return true for cancelled", func() {
				participant.Status = entity.ParticipantStatusCancelled
				Expect(participant.IsValidStatus()).To(BeTrue())
			})

			It("should return true for declined", func() {
				participant.Status = entity.ParticipantStatusDeclined
				Expect(participant.IsValidStatus()).To(BeTrue())
			})
		})

		Context("with invalid status value", func() {
			It("should return false", func() {
				participant.Status = entity.ParticipantStatus("invalid")
				Expect(participant.IsValidStatus()).To(BeFalse())
			})
		})

		Context("with valid payment status values", func() {
			It("should return true for paid", func() {
				participant.PaymentStatus = entity.PaymentPaid
				Expect(participant.IsValidPaymentStatus()).To(BeTrue())
			})

			It("should return true for unpaid", func() {
				participant.PaymentStatus = entity.PaymentUnpaid
				Expect(participant.IsValidPaymentStatus()).To(BeTrue())
			})
		})

		Context("with invalid payment status value", func() {
			It("should return false", func() {
				participant.PaymentStatus = entity.PaymentStatus("invalid")
				Expect(participant.IsValidPaymentStatus()).To(BeFalse())
			})
		})
	})

	Describe("Optional fields", func() {
		Context("with all optional fields populated", func() {
			BeforeEach(func() {
				phone := "+81901234567"
				empID := "EMP001"
				qrEmail := "qr@example.com"
				amount := 1500.00
				payDate := time.Now()

				participant.Phone = &phone
				participant.EmployeeID = &empID
				participant.QREmail = &qrEmail
				participant.PaymentAmount = &amount
				participant.PaymentDate = &payDate
			})

			It("should validate successfully", func() {
				err := participant.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with all optional fields nil", func() {
			BeforeEach(func() {
				participant.Phone = nil
				participant.EmployeeID = nil
				participant.QREmail = nil
				participant.PaymentAmount = nil
				participant.PaymentDate = nil
			})

			It("should validate successfully", func() {
				err := participant.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
