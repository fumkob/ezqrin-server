package entity_test

import (
	"encoding/json"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Checkin", func() {
	var (
		validCheckin  *entity.Checkin
		eventID       uuid.UUID
		participantID uuid.UUID
		checkedInBy   uuid.UUID
		now           time.Time
	)

	BeforeEach(func() {
		now = time.Now()
		eventID = uuid.New()
		participantID = uuid.New()
		checkedInBy = uuid.New()

		validCheckin = &entity.Checkin{
			ID:            uuid.New(),
			EventID:       eventID,
			ParticipantID: participantID,
			CheckedInAt:   now,
			CheckedInBy:   &checkedInBy,
			Method:        entity.CheckinMethodQRCode,
			DeviceInfo:    nil,
		}
	})

	When("validating checkin", func() {
		Context("with all required fields", func() {
			It("should succeed", func() {
				Expect(validCheckin.Validate()).To(Succeed())
			})
		})

		Context("with QR code method", func() {
			It("should succeed", func() {
				validCheckin.Method = entity.CheckinMethodQRCode
				Expect(validCheckin.Validate()).To(Succeed())
			})
		})

		Context("with manual method", func() {
			It("should succeed", func() {
				validCheckin.Method = entity.CheckinMethodManual
				Expect(validCheckin.Validate()).To(Succeed())
			})
		})

		Context("with nil event ID", func() {
			It("should fail", func() {
				validCheckin.EventID = uuid.Nil
				Expect(validCheckin.Validate()).To(MatchError(entity.ErrCheckinEventIDRequired))
			})
		})

		Context("with nil participant ID", func() {
			It("should fail", func() {
				validCheckin.ParticipantID = uuid.Nil
				Expect(validCheckin.Validate()).To(MatchError(entity.ErrCheckinParticipantIDRequired))
			})
		})

		Context("with invalid method", func() {
			It("should fail", func() {
				validCheckin.Method = entity.CheckinMethod("invalid")
				Expect(validCheckin.Validate()).To(MatchError(entity.ErrCheckinMethodInvalid))
			})
		})

		Context("with nil checked_in_by (self-service kiosk)", func() {
			It("should succeed", func() {
				validCheckin.CheckedInBy = nil
				Expect(validCheckin.Validate()).To(Succeed())
			})
		})

		Context("with device info", func() {
			It("should succeed", func() {
				deviceInfo := json.RawMessage(`{"os":"iOS","version":"16.0","app":"ezQRin"}`)
				validCheckin.DeviceInfo = &deviceInfo
				Expect(validCheckin.Validate()).To(Succeed())
			})
		})
	})

	When("checking method type", func() {
		Context("with QR code method", func() {
			It("should return true for IsQRCodeMethod", func() {
				validCheckin.Method = entity.CheckinMethodQRCode
				Expect(validCheckin.IsQRCodeMethod()).To(BeTrue())
				Expect(validCheckin.IsManualMethod()).To(BeFalse())
			})
		})

		Context("with manual method", func() {
			It("should return true for IsManualMethod", func() {
				validCheckin.Method = entity.CheckinMethodManual
				Expect(validCheckin.IsManualMethod()).To(BeTrue())
				Expect(validCheckin.IsQRCodeMethod()).To(BeFalse())
			})
		})
	})

	When("checking method validity", func() {
		Context("with valid QR code method", func() {
			It("should return true", func() {
				validCheckin.Method = entity.CheckinMethodQRCode
				Expect(validCheckin.IsValidMethod()).To(BeTrue())
			})
		})

		Context("with valid manual method", func() {
			It("should return true", func() {
				validCheckin.Method = entity.CheckinMethodManual
				Expect(validCheckin.IsValidMethod()).To(BeTrue())
			})
		})

		Context("with invalid method", func() {
			It("should return false", func() {
				validCheckin.Method = entity.CheckinMethod("invalid")
				Expect(validCheckin.IsValidMethod()).To(BeFalse())
			})
		})
	})

	When("converting CheckinMethod to string", func() {
		Context("with QR code method", func() {
			It("should return 'qrcode'", func() {
				method := entity.CheckinMethodQRCode
				Expect(method.String()).To(Equal("qrcode"))
			})
		})

		Context("with manual method", func() {
			It("should return 'manual'", func() {
				method := entity.CheckinMethodManual
				Expect(method.String()).To(Equal("manual"))
			})
		})
	})
})
