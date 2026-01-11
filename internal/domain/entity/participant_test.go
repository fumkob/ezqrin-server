package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestParticipant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Participant Entity Suite")
}

var _ = Describe("Participant Entity", func() {
	var (
		eventID   uuid.UUID
		validName string
		validEmail string
	)

	BeforeEach(func() {
		eventID = uuid.New()
		validName = "John Doe"
		validEmail = "john@example.com"
	})

	Describe("Validate", func() {
		When("validating a valid participant", func() {
			Context("with all required fields", func() {
				It("should succeed", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             validEmail,
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code-12345",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(BeNil())
				})
			})

			Context("with optional phone field", func() {
				It("should succeed", func() {
					phone := "+1234567890"
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             validEmail,
						Phone:             &phone,
						Status:            ParticipantStatusConfirmed,
						QRCode:            "test-qr-code-12345",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(BeNil())
				})
			})

			Context("with optional employee_id", func() {
				It("should succeed", func() {
					empID := "EMP123"
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             validEmail,
						EmployeeID:        &empID,
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code-12345",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(BeNil())
				})
			})

			Context("with all optional fields", func() {
				It("should succeed", func() {
					phone := "+1234567890"
					empID := "EMP123"
					qrEmail := "qr@example.com"
					amount := 100.0
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             validEmail,
						Phone:             &phone,
						EmployeeID:        &empID,
						QREmail:           &qrEmail,
						Status:            ParticipantStatusConfirmed,
						QRCode:            "test-qr-code-12345",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentPaid,
						PaymentAmount:     &amount,
						Metadata: map[string]interface{}{
							"notes": "VIP attendee",
						},
					}
					Expect(p.Validate()).To(BeNil())
				})
			})
		})

		When("validating a participant with missing event_id", func() {
			It("should fail", func() {
				p := &Participant{
					Name:              validName,
					Email:             validEmail,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantEventIDRequired))
			})
		})

		When("validating a participant with missing name", func() {
			It("should fail", func() {
				p := &Participant{
					EventID:           eventID,
					Email:             validEmail,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantNameRequired))
			})
		})

		When("validating a participant with name exceeding max length", func() {
			It("should fail", func() {
				longName := ""
				for i := 0; i < 256; i++ {
					longName += "a"
				}
				p := &Participant{
					EventID:           eventID,
					Name:              longName,
					Email:             validEmail,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantNameTooLong))
			})
		})

		When("validating a participant with missing email", func() {
			It("should fail", func() {
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantEmailRequired))
			})
		})

		When("validating a participant with invalid email", func() {
			Context("email without @ symbol", func() {
				It("should fail", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             "invalidemail.com",
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(Equal(ErrParticipantEmailInvalid))
				})
			})

			Context("email with multiple @ symbols", func() {
				It("should fail", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             "user@@example.com",
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(Equal(ErrParticipantEmailInvalid))
				})
			})

			Context("email without domain", func() {
				It("should fail", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             "user@",
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(Equal(ErrParticipantEmailInvalid))
				})
			})

			Context("email without local part", func() {
				It("should fail", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             "@example.com",
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(Equal(ErrParticipantEmailInvalid))
				})
			})

			Context("email without dot in domain", func() {
				It("should fail", func() {
					p := &Participant{
						EventID:           eventID,
						Name:              validName,
						Email:             "user@domain",
						Status:            ParticipantStatusTentative,
						QRCode:            "test-qr-code",
						QRCodeGeneratedAt: Now(),
						PaymentStatus:     PaymentUnpaid,
					}
					Expect(p.Validate()).To(Equal(ErrParticipantEmailInvalid))
				})
			})
		})

		When("validating a participant with email exceeding max length", func() {
			It("should fail", func() {
				longEmail := ""
				for i := 0; i < 256; i++ {
					longEmail += "a"
				}
				longEmail = "a@" + longEmail[2:]
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             longEmail,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantEmailTooLong))
			})
		})

		When("validating a participant with phone exceeding max length", func() {
			It("should fail", func() {
				longPhone := ""
				for i := 0; i < 51; i++ {
					longPhone += "1"
				}
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             validEmail,
					Phone:             &longPhone,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantPhoneTooLong))
			})
		})

		When("validating a participant with employee_id exceeding max length", func() {
			It("should fail", func() {
				longEmpID := ""
				for i := 0; i < 256; i++ {
					longEmpID += "E"
				}
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             validEmail,
					EmployeeID:        &longEmpID,
					Status:            ParticipantStatusTentative,
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantEmployeeIDLong))
			})
		})

		When("validating a participant with missing qr_code", func() {
			It("should fail", func() {
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             validEmail,
					Status:            ParticipantStatusTentative,
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantQRCodeRequired))
			})
		})

		When("validating a participant with qr_code exceeding max length", func() {
			It("should fail", func() {
				longQR := ""
				for i := 0; i < 256; i++ {
					longQR += "q"
				}
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             validEmail,
					Status:            ParticipantStatusTentative,
					QRCode:            longQR,
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantQRCodeTooLong))
			})
		})

		When("validating a participant with invalid status", func() {
			It("should fail", func() {
				p := &Participant{
					EventID:           eventID,
					Name:              validName,
					Email:             validEmail,
					Status:            ParticipantStatus("invalid_status"),
					QRCode:            "test-qr-code",
					QRCodeGeneratedAt: Now(),
					PaymentStatus:     PaymentUnpaid,
				}
				Expect(p.Validate()).To(Equal(ErrParticipantStatusInvalid))
			})
		})
	})

	Describe("IsValidStatus", func() {
		When("checking valid statuses", func() {
			Context("with tentative status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusTentative}
					Expect(p.IsValidStatus()).To(BeTrue())
				})
			})

			Context("with confirmed status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusConfirmed}
					Expect(p.IsValidStatus()).To(BeTrue())
				})
			})

			Context("with cancelled status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusCancelled}
					Expect(p.IsValidStatus()).To(BeTrue())
				})
			})

			Context("with declined status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusDeclined}
					Expect(p.IsValidStatus()).To(BeTrue())
				})
			})
		})

		When("checking invalid status", func() {
			It("should return false", func() {
				p := &Participant{Status: ParticipantStatus("unknown")}
				Expect(p.IsValidStatus()).To(BeFalse())
			})
		})
	})

	Describe("Status helper methods", func() {
		When("checking IsPaid", func() {
			Context("with paid payment status", func() {
				It("should return true", func() {
					p := &Participant{PaymentStatus: PaymentPaid}
					Expect(p.IsPaid()).To(BeTrue())
				})
			})

			Context("with unpaid payment status", func() {
				It("should return false", func() {
					p := &Participant{PaymentStatus: PaymentUnpaid}
					Expect(p.IsPaid()).To(BeFalse())
				})
			})
		})

		When("checking IsConfirmed", func() {
			Context("with confirmed status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusConfirmed}
					Expect(p.IsConfirmed()).To(BeTrue())
				})
			})

			Context("with tentative status", func() {
				It("should return false", func() {
					p := &Participant{Status: ParticipantStatusTentative}
					Expect(p.IsConfirmed()).To(BeFalse())
				})
			})
		})

		When("checking IsCancelled", func() {
			Context("with cancelled status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusCancelled}
					Expect(p.IsCancelled()).To(BeTrue())
				})
			})

			Context("with tentative status", func() {
				It("should return false", func() {
					p := &Participant{Status: ParticipantStatusTentative}
					Expect(p.IsCancelled()).To(BeFalse())
				})
			})
		})

		When("checking IsDeclined", func() {
			Context("with declined status", func() {
				It("should return true", func() {
					p := &Participant{Status: ParticipantStatusDeclined}
					Expect(p.IsDeclined()).To(BeTrue())
				})
			})

			Context("with tentative status", func() {
				It("should return false", func() {
					p := &Participant{Status: ParticipantStatusTentative}
					Expect(p.IsDeclined()).To(BeFalse())
				})
			})
		})
	})
})

// Helper function for test
func Now() time.Time {
	return time.Now()
}
