package validator_test

import (
	"testing"

	"github.com/fumkob/ezqrin-server/pkg/validator"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator Package Suite")
}

var _ = Describe("Validator", func() {
	var v *validator.Validator

	BeforeEach(func() {
		v = validator.New()
	})

	When("creating a new validator", func() {
		Context("with New constructor", func() {
			It("should create validator instance", func() {
				val := validator.New()

				Expect(val).NotTo(BeNil())
			})

			It("should register custom validators", func() {
				val := validator.New()

				Expect(val).NotTo(BeNil())
				// Custom validators should be available
			})
		})
	})

	When("validating structs", func() {
		Context("with required fields", func() {
			type TestStruct struct {
				Name  string `validate:"required"`
				Email string `validate:"required,email"`
			}

			It("should pass with valid data", func() {
				data := TestStruct{
					Name:  "John Doe",
					Email: "john@example.com",
				}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail when required field is empty", func() {
				data := TestStruct{
					Name:  "",
					Email: "john@example.com",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("name is required"))
			})

			It("should fail with multiple missing required fields", func() {
				data := TestStruct{
					Name:  "",
					Email: "",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})
		})

		Context("with email validation", func() {
			type EmailStruct struct {
				Email string `validate:"required,email"`
			}

			It("should accept valid email", func() {
				data := EmailStruct{Email: "test@example.com"}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept email with subdomain", func() {
				data := EmailStruct{Email: "user@mail.example.com"}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid email format", func() {
				data := EmailStruct{Email: "invalid-email"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("email"))
			})

			It("should reject email without domain", func() {
				data := EmailStruct{Email: "user@"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
			})

			It("should reject email without @", func() {
				data := EmailStruct{Email: "userexample.com"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("with custom email_format validator", func() {
			type CustomEmailStruct struct {
				Email string `validate:"email_format"`
			}

			It("should accept valid email", func() {
				data := CustomEmailStruct{Email: "test@example.com"}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept empty email when not required", func() {
				data := CustomEmailStruct{Email: ""}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid email format", func() {
				data := CustomEmailStruct{Email: "not-an-email"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valid email"))
			})
		})

		Context("with UUID validation", func() {
			type UUIDStruct struct {
				ID string `validate:"required,uuid4"`
			}

			It("should accept valid UUID v4", func() {
				validUUID := uuid.New().String()
				data := UUIDStruct{ID: validUUID}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid UUID format", func() {
				data := UUIDStruct{ID: "not-a-uuid"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("UUID"))
			})

			It("should reject empty UUID when required", func() {
				data := UUIDStruct{ID: ""}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})

			It("should accept empty UUID when not required", func() {
				type OptionalUUIDStruct struct {
					ID string `validate:"uuid4"`
				}
				data := OptionalUUIDStruct{ID: ""}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with length constraints", func() {
			type LengthStruct struct {
				MinField   string `validate:"min=3"`
				MaxField   string `validate:"max=10"`
				ExactField string `validate:"len=5"`
			}

			It("should pass with valid lengths", func() {
				data := LengthStruct{
					MinField:   "abc",
					MaxField:   "1234567890",
					ExactField: "12345",
				}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail when string too short", func() {
				data := LengthStruct{
					MinField:   "ab",
					MaxField:   "valid",
					ExactField: "12345",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least 3 characters"))
			})

			It("should fail when string too long", func() {
				data := LengthStruct{
					MinField:   "valid",
					MaxField:   "12345678901",
					ExactField: "12345",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at most 10 characters"))
			})

			It("should fail when exact length not met", func() {
				data := LengthStruct{
					MinField:   "valid",
					MaxField:   "valid",
					ExactField: "1234",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("exactly 5 characters"))
			})
		})

		Context("with numeric constraints", func() {
			type NumericStruct struct {
				Greater        int `validate:"gt=0"`
				GreaterOrEqual int `validate:"gte=1"`
				Less           int `validate:"lt=100"`
				LessOrEqual    int `validate:"lte=10"`
			}

			It("should pass with valid values", func() {
				data := NumericStruct{
					Greater:        1,
					GreaterOrEqual: 1,
					Less:           99,
					LessOrEqual:    10,
				}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail when value not greater than", func() {
				data := NumericStruct{
					Greater:        0,
					GreaterOrEqual: 1,
					Less:           50,
					LessOrEqual:    5,
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("greater than"))
			})

			It("should fail when value not less than", func() {
				data := NumericStruct{
					Greater:        1,
					GreaterOrEqual: 1,
					Less:           100,
					LessOrEqual:    5,
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("less than"))
			})
		})

		Context("with oneof validation", func() {
			type OneOfStruct struct {
				Status string `validate:"oneof=active inactive pending"`
			}

			It("should accept valid option", func() {
				data := OneOfStruct{Status: "active"}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept another valid option", func() {
				data := OneOfStruct{Status: "pending"}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid option", func() {
				data := OneOfStruct{Status: "unknown"}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("one of"))
			})
		})

		Context("with multiple validation errors", func() {
			type MultiErrorStruct struct {
				Name  string `validate:"required,min=3"`
				Email string `validate:"required,email"`
				Age   int    `validate:"required,gte=18"`
			}

			It("should report all validation errors", func() {
				data := MultiErrorStruct{
					Name:  "ab",
					Email: "invalid",
					Age:   15,
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				// Should contain multiple error messages
				errMsg := err.Error()
				Expect(errMsg).To(ContainSubstring("name"))
				Expect(errMsg).To(ContainSubstring("email"))
				Expect(errMsg).To(ContainSubstring("age"))
			})
		})

		Context("with nested structs", func() {
			type Address struct {
				Street string `validate:"required"`
				City   string `validate:"required"`
			}

			type Person struct {
				Name    string  `validate:"required"`
				Address Address `validate:"required"`
			}

			It("should validate nested struct fields", func() {
				data := Person{
					Name: "John",
					Address: Address{
						Street: "123 Main St",
						City:   "New York",
					},
				}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail on nested validation errors", func() {
				data := Person{
					Name: "John",
					Address: Address{
						Street: "",
						City:   "New York",
					},
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("street"))
			})
		})
	})

	When("validating single variables", func() {
		Context("with ValidateVar method", func() {
			It("should validate email format", func() {
				err := v.ValidateVar("test@example.com", "email")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject invalid email", func() {
				err := v.ValidateVar("not-an-email", "email")

				Expect(err).To(HaveOccurred())
			})

			It("should validate required field", func() {
				err := v.ValidateVar("", "required")

				Expect(err).To(HaveOccurred())
			})

			It("should validate UUID", func() {
				validUUID := uuid.New().String()
				err := v.ValidateVar(validUUID, "uuid4")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate minimum length", func() {
				err := v.ValidateVar("ab", "min=3")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least 3"))
			})
		})
	})

	When("using helper functions", func() {
		Context("with ValidateEmail", func() {
			It("should accept valid email", func() {
				err := validator.ValidateEmail("test@example.com")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept email with plus sign", func() {
				err := validator.ValidateEmail("test+tag@example.com")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept email with subdomain", func() {
				err := validator.ValidateEmail("user@mail.example.co.uk")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject empty email", func() {
				err := validator.ValidateEmail("")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})

			It("should reject invalid email format", func() {
				err := validator.ValidateEmail("not-an-email")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid email format"))
			})

			It("should reject email without @", func() {
				err := validator.ValidateEmail("testexample.com")

				Expect(err).To(HaveOccurred())
			})

			It("should reject email without domain", func() {
				err := validator.ValidateEmail("test@")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("with ValidateUUID", func() {
			It("should accept valid UUID v4", func() {
				validUUID := uuid.New().String()
				err := validator.ValidateUUID(validUUID)

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject empty UUID", func() {
				err := validator.ValidateUUID("")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})

			It("should reject invalid UUID format", func() {
				err := validator.ValidateUUID("not-a-uuid")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid UUID format"))
			})

			It("should reject UUID with wrong version", func() {
				// Create a UUID v1 (time-based) manually
				uuidV1 := "550e8400-e29b-11d4-a716-446655440000"
				err := validator.ValidateUUID(uuidV1)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version 4"))
			})

			It("should reject malformed UUID", func() {
				err := validator.ValidateUUID("123-456-789")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("with ValidateRequired", func() {
			It("should accept non-empty string", func() {
				err := validator.ValidateRequired("value", "field_name")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject empty string", func() {
				err := validator.ValidateRequired("", "field_name")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("field_name is required"))
			})

			It("should reject whitespace-only string", func() {
				err := validator.ValidateRequired("   ", "field_name")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required"))
			})

			It("should use provided field name in error", func() {
				err := validator.ValidateRequired("", "email_address")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("email_address"))
			})
		})

		Context("with ValidateMinLength", func() {
			It("should accept string meeting minimum length", func() {
				err := validator.ValidateMinLength("abc", 3, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept string exceeding minimum length", func() {
				err := validator.ValidateMinLength("abcde", 3, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject string shorter than minimum", func() {
				err := validator.ValidateMinLength("ab", 3, "password")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("password must be at least 3 characters"))
			})

			It("should handle zero minimum length", func() {
				err := validator.ValidateMinLength("", 0, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with ValidateMaxLength", func() {
			It("should accept string within maximum length", func() {
				err := validator.ValidateMaxLength("abc", 5, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept string at maximum length", func() {
				err := validator.ValidateMaxLength("abcde", 5, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject string exceeding maximum", func() {
				err := validator.ValidateMaxLength("abcdef", 5, "username")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("username must be at most 5 characters"))
			})

			It("should accept empty string", func() {
				err := validator.ValidateMaxLength("", 10, "field_name")

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("formatting validation errors", func() {
		Context("with snake_case conversion", func() {
			type FieldNameStruct struct {
				FirstName string `validate:"required"`
				LastName  string `validate:"required"`
				EmailAddr string `validate:"required,email"`
			}

			It("should convert PascalCase to snake_case in errors", func() {
				data := FieldNameStruct{}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("first_name"))
				Expect(err.Error()).To(ContainSubstring("last_name"))
				Expect(err.Error()).To(ContainSubstring("email_addr"))
			})
		})

		Context("with different validation tags", func() {
			It("should format required error", func() {
				type T struct {
					Field string `validate:"required"`
				}
				err := v.Validate(T{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is required"))
			})

			It("should format email error", func() {
				type T struct {
					Field string `validate:"email"`
				}
				err := v.Validate(T{Field: "invalid"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valid email"))
			})

			It("should format min error", func() {
				type T struct {
					Field string `validate:"min=5"`
				}
				err := v.Validate(T{Field: "abc"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least 5"))
			})

			It("should format max error", func() {
				type T struct {
					Field string `validate:"max=3"`
				}
				err := v.Validate(T{Field: "abcd"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at most 3"))
			})

			It("should format url error", func() {
				type T struct {
					Field string `validate:"url"`
				}
				err := v.Validate(T{Field: "not-a-url"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valid URL"))
			})
		})
	})

	When("handling edge cases", func() {
		Context("with empty structs", func() {
			type EmptyStruct struct{}

			It("should validate empty struct without error", func() {
				data := EmptyStruct{}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with nil pointers", func() {
			type PointerStruct struct {
				Name *string `validate:"required"`
			}

			It("should fail validation for nil required pointer", func() {
				data := PointerStruct{Name: nil}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
			})

			It("should pass validation for non-nil pointer", func() {
				name := "John"
				data := PointerStruct{Name: &name}

				err := v.Validate(data)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with zero values", func() {
			type ZeroValueStruct struct {
				Count int    `validate:"required"`
				Flag  bool   `validate:"required"`
				Text  string `validate:"required"`
			}

			It("should handle zero values correctly", func() {
				data := ZeroValueStruct{
					Count: 0,
					Flag:  false,
					Text:  "",
				}

				err := v.Validate(data)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("text is required"))
			})
		})

		Context("with special characters", func() {
			It("should validate email with special characters", func() {
				err := validator.ValidateEmail("user+tag@example.com")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate email with dots", func() {
				err := validator.ValidateEmail("first.last@example.com")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate email with hyphens", func() {
				err := validator.ValidateEmail("user-name@example-domain.com")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate email with underscores", func() {
				err := validator.ValidateEmail("user_name@example.com")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with boundary values", func() {
			It("should validate exact minimum length", func() {
				err := validator.ValidateMinLength("abc", 3, "field")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate exact maximum length", func() {
				err := validator.ValidateMaxLength("abc", 3, "field")

				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject one character below minimum", func() {
				err := validator.ValidateMinLength("ab", 3, "field")

				Expect(err).To(HaveOccurred())
			})

			It("should reject one character above maximum", func() {
				err := validator.ValidateMaxLength("abcd", 3, "field")

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
