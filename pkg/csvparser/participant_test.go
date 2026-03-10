package csvparser_test

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/pkg/csvparser"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseParticipantCSV", func() {
	When("parsing a valid CSV", func() {
		Context("with all fields present", func() {
			It("should return parsed inputs for all rows", func() {
				csv := `name,email,employee_id,phone,status,payment_status,payment_amount,payment_date,metadata
Jane Smith,jane@example.com,EMP001,+1-555-0123,confirmed,paid,150.00,2025-11-08T12:30:00Z,"{""company"":""Tech Corp""}"
John Doe,john@example.com,,,tentative,unpaid,,,`

				inputs, rowErrors, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))

				Expect(err).To(BeNil())
				Expect(rowErrors).To(BeEmpty())
				Expect(inputs).To(HaveLen(2))
				Expect(inputs[0].Row).To(Equal(1))
				Expect(inputs[0].Input.Name).To(Equal("Jane Smith"))
				Expect(inputs[0].Input.Email).To(Equal("jane@example.com"))
				Expect(inputs[0].Input.EmployeeID).To(HaveValue(Equal("EMP001")))
				Expect(inputs[0].Input.Phone).To(HaveValue(Equal("+1-555-0123")))
				Expect(inputs[0].Input.PaymentAmount).To(HaveValue(BeNumerically("==", 150.0)))
				Expect(inputs[0].Input.PaymentDate).NotTo(BeNil())
				Expect(inputs[1].Row).To(Equal(2))
				Expect(inputs[1].Input.Name).To(Equal("John Doe"))
				Expect(inputs[1].Input.EmployeeID).To(BeNil())
			})
		})

		Context("with columns in different order", func() {
			It("should parse correctly regardless of column order", func() {
				csv := `email,name,status
jane@example.com,Jane Smith,confirmed`

				inputs, rowErrors, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))

				Expect(err).To(BeNil())
				Expect(rowErrors).To(BeEmpty())
				Expect(inputs).To(HaveLen(1))
				Expect(inputs[0].Input.Name).To(Equal("Jane Smith"))
				Expect(inputs[0].Input.Email).To(Equal("jane@example.com"))
			})
		})

		Context("with optional columns missing", func() {
			It("should parse with nil optional fields", func() {
				csv := `name,email
Jane Smith,jane@example.com`

				inputs, rowErrors, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))

				Expect(err).To(BeNil())
				Expect(rowErrors).To(BeEmpty())
				Expect(inputs).To(HaveLen(1))
				Expect(inputs[0].Input.EmployeeID).To(BeNil())
				Expect(inputs[0].Input.Phone).To(BeNil())
				Expect(inputs[0].Input.PaymentAmount).To(BeNil())
				Expect(inputs[0].Input.PaymentDate).To(BeNil())
			})
		})
	})

	When("parsing CSV with row-level errors", func() {
		Context("with invalid payment_date format", func() {
			It("should collect the row error and skip the row", func() {
				csv := `name,email,payment_date
Jane Smith,jane@example.com,not-a-date
John Doe,john@example.com,2025-11-08T12:30:00Z`

				inputs, rowErrors, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))

				Expect(err).To(BeNil())
				Expect(rowErrors).To(HaveLen(1))
				Expect(rowErrors[0].Row).To(Equal(1))
				Expect(rowErrors[0].Email).To(Equal("jane@example.com"))
				Expect(rowErrors[0].Message).To(ContainSubstring("payment_date"))
				Expect(inputs).To(HaveLen(1))
				Expect(inputs[0].Input.Email).To(Equal("john@example.com"))
			})
		})
	})

	When("parsing an invalid CSV", func() {
		Context("with empty file", func() {
			It("should return a file-level error", func() {
				_, _, err := csvparser.ParseParticipantCSV(strings.NewReader(""))
				Expect(err).To(MatchError(ContainSubstring("empty")))
			})
		})

		Context("with header only (no data rows)", func() {
			It("should return a file-level error", func() {
				csv := "name,email"
				_, _, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))
				Expect(err).To(MatchError(ContainSubstring("no data rows")))
			})
		})

		Context("with missing required column 'name'", func() {
			It("should return a file-level error", func() {
				csv := `email,phone
jane@example.com,+1-555-0123`
				_, _, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))
				Expect(err).To(MatchError(ContainSubstring("'name'")))
			})
		})

		Context("with missing required column 'email'", func() {
			It("should return a file-level error", func() {
				csv := `name,phone
Jane Smith,+1-555-0123`
				_, _, err := csvparser.ParseParticipantCSV(strings.NewReader(csv))
				Expect(err).To(MatchError(ContainSubstring("'email'")))
			})
		})
	})
})

var _ = Describe("ExportParticipantCSV", func() {
	When("exporting a list of participants", func() {
		Context("with full data and default (English) format", func() {
			It("should write CSV with English status strings", func() {
				now := time.Now().UTC().Truncate(time.Second)
				empID := "EMP001"
				phone := "+81-90-1234-5678"
				meta := json.RawMessage(`{"foo":"bar"}`)
				amount := 1000.0
				checkedInAt := now

				p := &entity.Participant{
					ID:                uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					EventID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:              "Alice",
					Email:             "alice@example.com",
					EmployeeID:        &empID,
					Phone:             &phone,
					Status:            entity.ParticipantStatusConfirmed,
					QRCode:            "qr123",
					QRCodeGeneratedAt: now,
					QRDistributionURL: "https://example.com/qr",
					PaymentStatus:     entity.PaymentPaid,
					PaymentAmount:     &amount,
					PaymentDate:       &now,
					Metadata:          &meta,
					CreatedAt:         now,
					UpdatedAt:         now,
					CheckedIn:         true,
					CheckedInAt:       &checkedInAt,
				}

				var buf strings.Builder
				err := csvparser.ExportParticipantCSV(&buf, []*entity.Participant{p}, csvparser.StatusFormatEnglish)
				Expect(err).NotTo(HaveOccurred())

				lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
				Expect(lines).To(HaveLen(2))
				Expect(lines[0]).To(Equal(
					"id,name,email,employee_id,phone,qr_email,status,qr_code," +
						"qr_code_generated_at,qr_distribution_url,payment_status," +
						"payment_amount,payment_date,checked_in,checked_in_at,metadata," +
						"created_at,updated_at",
				))
				Expect(lines[1]).To(ContainSubstring("alice@example.com"))
				Expect(lines[1]).To(ContainSubstring("confirmed"))
				Expect(lines[1]).To(ContainSubstring("true"))
			})
		})

		Context("with Japanese format", func() {
			It("should write ○△× symbols for status", func() {
				confirmed := &entity.Participant{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name: "Alice", Email: "alice@example.com",
					Status: entity.ParticipantStatusConfirmed, QRCode: "qr1",
					PaymentStatus: entity.PaymentUnpaid, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
				tentative := &entity.Participant{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name: "Bob", Email: "bob@example.com",
					Status: entity.ParticipantStatusTentative, QRCode: "qr2",
					PaymentStatus: entity.PaymentUnpaid, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
				cancelled := &entity.Participant{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name: "Carol", Email: "carol@example.com",
					Status: entity.ParticipantStatusCancelled, QRCode: "qr3",
					PaymentStatus: entity.PaymentUnpaid, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}

				var buf strings.Builder
				err := csvparser.ExportParticipantCSV(&buf, []*entity.Participant{confirmed, tentative, cancelled}, csvparser.StatusFormatJapanese)
				Expect(err).NotTo(HaveOccurred())

				lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
				Expect(lines).To(HaveLen(4))
				Expect(lines[1]).To(ContainSubstring("○"))
				Expect(lines[2]).To(ContainSubstring("△"))
				Expect(lines[3]).To(ContainSubstring("×"))
			})
		})

		Context("with nil optional fields", func() {
			It("should write empty strings for nil fields", func() {
				p := &entity.Participant{
					ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:          "Bob",
					Email:         "bob@example.com",
					Status:        entity.ParticipantStatusTentative,
					QRCode:        "qr456",
					PaymentStatus: entity.PaymentUnpaid,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}

				var buf strings.Builder
				err := csvparser.ExportParticipantCSV(&buf, []*entity.Participant{p}, csvparser.StatusFormatEnglish)
				Expect(err).NotTo(HaveOccurred())

				lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
				Expect(lines).To(HaveLen(2))
				fields := strings.Split(lines[1], ",")
				Expect(fields).To(HaveLen(18)) // 18 columns
			})
		})

		Context("with empty participant list", func() {
			It("should write only the header row", func() {
				var buf strings.Builder
				err := csvparser.ExportParticipantCSV(&buf, []*entity.Participant{}, csvparser.StatusFormatEnglish)
				Expect(err).NotTo(HaveOccurred())

				lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
				Expect(lines).To(HaveLen(1))
				Expect(lines[0]).To(ContainSubstring("id,name,email"))
			})
		})
	})
})
