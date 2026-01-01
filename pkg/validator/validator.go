// Package validator provides request validation with user-friendly error messages.
//
// This package wraps go-playground/validator/v10 with custom validators and
// formatted error messages. It includes validators for common patterns like
// UUID v4, email format, and standard validations.
//
// Example usage:
//
//	// Initialize validator
//	v := validator.New()
//
//	// Define request struct with validation tags
//	type CreateUserRequest struct {
//		Email    string `json:"email" validate:"required,email_format"`
//		Password string `json:"password" validate:"required,min=8,max=72"`
//		Name     string `json:"name" validate:"required,min=2,max=100"`
//		Role     string `json:"role" validate:"required,oneof=admin organizer staff"`
//	}
//
//	// Validate struct
//	req := CreateUserRequest{Email: "invalid", Password: "short"}
//	if err := v.Validate(req); err != nil {
//		// Returns: "email must be a valid email address; password must be at least 8 characters"
//		return err
//	}
//
//	// Validate single field
//	if err := v.ValidateVar(email, "required,email_format"); err != nil {
//		return err
//	}
//
//	// Helper functions
//	if err := validator.ValidateEmail(email); err != nil {
//		return err
//	}
//
//	if err := validator.ValidateUUID(id); err != nil {
//		return err
//	}
package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Validator wraps go-playground/validator with custom validators
type Validator struct {
	validate *validator.Validate
}

var (
	// emailRegex is a simple email validation regex
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// New creates a new Validator instance with custom validators registered
func New() *Validator {
	v := validator.New()

	// Register custom validators
	_ = v.RegisterValidation("uuid4", isUUIDv4)
	_ = v.RegisterValidation("email_format", isEmailFormat)

	return &Validator{
		validate: v,
	}
}

// Validate validates a struct and returns formatted error messages
func (v *Validator) Validate(data interface{}) error {
	if err := v.validate.Struct(data); err != nil {
		return v.formatValidationErrors(err)
	}
	return nil
}

// ValidateVar validates a single variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	if err := v.validate.Var(field, tag); err != nil {
		return v.formatValidationErrors(err)
	}
	return nil
}

// formatValidationErrors converts validator errors to user-friendly messages
func (v *Validator) formatValidationErrors(err error) error {
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var messages []string
	for _, e := range validationErrs {
		messages = append(messages, formatFieldError(e))
	}

	return fmt.Errorf("%s", strings.Join(messages, "; "))
}

// formatFieldError formats a single field validation error
func formatFieldError(e validator.FieldError) string {
	field := toSnakeCase(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "email_format":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID v4", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a valid number", field)
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format %s", field, e.Param())
	default:
		return fmt.Sprintf("%s failed validation on tag '%s'", field, e.Tag())
	}
}

// toSnakeCase converts PascalCase or camelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isUUIDv4 validates if a field is a valid UUID v4
func isUUIDv4(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Let 'required' tag handle empty values
	}

	parsedUUID, err := uuid.Parse(value)
	if err != nil {
		return false
	}

	// Check if it's version 4
	return parsedUUID.Version() == 4
}

// isEmailFormat validates email format using regex
func isEmailFormat(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Let 'required' tag handle empty values
	}

	return emailRegex.MatchString(value)
}

// ValidateEmail validates email address format
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateUUID validates UUID v4 format
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("UUID is required")
	}

	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	if parsedUUID.Version() != 4 {
		return fmt.Errorf("UUID must be version 4")
	}

	return nil
}

// ValidateRequired validates that a string is not empty
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateMinLength validates minimum string length
func ValidateMinLength(value string, minLength int, fieldName string) error {
	if len(value) < minLength {
		return fmt.Errorf("%s must be at least %d characters", fieldName, minLength)
	}
	return nil
}

// ValidateMaxLength validates maximum string length
func ValidateMaxLength(value string, maxLength int, fieldName string) error {
	if len(value) > maxLength {
		return fmt.Errorf("%s must be at most %d characters", fieldName, maxLength)
	}
	return nil
}
