package csvparser_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/pkg/csvparser"
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
