// Package errors provides application-specific error types with HTTP status codes.
//
// This package defines standard error types that map to HTTP status codes and
// error codes for API consumption. It supports error wrapping following Go 1.13+
// error handling patterns with errors.Is and errors.As.
//
// Example usage:
//
//	// Create typed errors
//	err := errors.NotFound("user not found")
//	err := errors.Validationf("invalid email: %s", email)
//	err := errors.Unauthorized("invalid credentials")
//
//	// Wrap errors with context
//	if err := repo.GetUser(id); err != nil {
//		return errors.Wrap(err, "failed to get user")
//	}
//
//	// Check error types
//	if errors.IsNotFound(err) {
//		// Handle not found case
//	}
//
//	// Extract HTTP status code
//	statusCode := errors.GetStatusCode(err) // Returns 404
//	errorCode := errors.GetErrorCode(err)   // Returns "NOT_FOUND"
//
//	// Wrap underlying error in AppError
//	appErr := errors.NotFound("user not found")
//	return errors.WrapAppError(appErr, sqlErr)
package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
)

// Common error codes for API responses
const (
	CodeNotFound           = "NOT_FOUND"
	CodeValidation         = "VALIDATION_ERROR"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeInternal           = "INTERNAL_ERROR"
	CodeConflict           = "CONFLICT"
	CodeBadRequest         = "BAD_REQUEST"
	CodeTooManyRequests    = "TOO_MANY_REQUESTS"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ProblemTypeBaseURL is the base URL for RFC 9457 problem type URIs.
// This should be configurable via application config in production.
// For now, using a placeholder URL that will be updated when the production domain is finalized.
const ProblemTypeBaseURL = "https://api.ezqrin.com/problems"

// ErrorTitles maps error codes to human-readable titles for RFC 9457.
// These titles provide a short, human-readable summary of the problem type.
var ErrorTitles = map[string]string{
	CodeNotFound:           "Resource Not Found",
	CodeValidation:         "Validation Error",
	CodeUnauthorized:       "Unauthorized",
	CodeForbidden:          "Forbidden",
	CodeInternal:           "Internal Server Error",
	CodeConflict:           "Conflict",
	CodeBadRequest:         "Bad Request",
	CodeTooManyRequests:    "Too Many Requests",
	CodeServiceUnavailable: "Service Unavailable",
}

// ValidationError is an alias for the OpenAPI-generated ValidationError type.
// Used in RFC 9457 Problem Details responses as an extension member.
type ValidationError = generated.ValidationError

// AppError represents an application-specific error with code and HTTP status
type AppError struct {
	Code             string            // Error code for API consumption
	Message          string            // Human-readable error message
	StatusCode       int               // HTTP status code
	Err              error             // Wrapped underlying error
	ValidationErrors []ValidationError // Field-level validation errors (for validation errors only)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// NotFound creates a 404 Not Found error
func NotFound(message string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

// NotFoundf creates a 404 Not Found error with formatting
func NotFoundf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusNotFound,
	}
}

// Validation creates a 400 Bad Request validation error
func Validation(message string) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// Validationf creates a 400 Bad Request validation error with formatting
func Validationf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusBadRequest,
	}
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// Unauthorizedf creates a 401 Unauthorized error with formatting
func Unauthorizedf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// Forbiddenf creates a 403 Forbidden error with formatting
func Forbiddenf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusForbidden,
	}
}

// Internal creates a 500 Internal Server Error
func Internal(message string) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

// Internalf creates a 500 Internal Server Error with formatting
func Internalf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusInternalServerError,
	}
}

// Conflict creates a 409 Conflict error
func Conflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// Conflictf creates a 409 Conflict error with formatting
func Conflictf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusConflict,
	}
}

// BadRequest creates a 400 Bad Request error
func BadRequest(message string) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// BadRequestf creates a 400 Bad Request error with formatting
func BadRequestf(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: http.StatusBadRequest,
	}
}

// TooManyRequests creates a 429 Too Many Requests error
func TooManyRequests(message string) *AppError {
	return &AppError{
		Code:       CodeTooManyRequests,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
	}
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(message string) *AppError {
	return &AppError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		StatusCode: http.StatusServiceUnavailable,
	}
}

// Wrap wraps an error with additional context while preserving the original error.
// Uses %w to maintain the error chain, enabling errors.Is and errors.As to traverse
// and check for specific error types even after multiple wrapping operations.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context while preserving the error chain.
// Uses %w to enable errors.Is and errors.As compatibility for type checking.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// WrapAppError wraps an underlying error into an AppError
func WrapAppError(appErr *AppError, err error) *AppError {
	if appErr == nil {
		return Internal("unexpected nil app error")
	}
	appErr.Err = err
	return appErr
}

// GetStatusCode extracts HTTP status code from error, defaults to 500
func GetStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	return http.StatusInternalServerError
}

// GetErrorCode extracts error code from error, defaults to INTERNAL_ERROR
func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	return CodeInternal
}

// IsNotFound checks if error is a NotFound error
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return false
}

// IsValidation checks if error is a Validation error
func IsValidation(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeValidation
	}
	return false
}

// IsUnauthorized checks if error is an Unauthorized error
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized
	}
	return false
}

// IsForbidden checks if error is a Forbidden error
func IsForbidden(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeForbidden
	}
	return false
}

// IsConflict checks if error is a Conflict error
func IsConflict(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeConflict
	}
	return false
}

// ToTypeURL converts an error code to an RFC 9457 type URL.
// Converts UPPER_SNAKE_CASE error codes to lowercase-kebab-case URLs.
// Example: "NOT_FOUND" -> "https://api.ezqrin.com/problems/not-found"
func ToTypeURL(code string) string {
	// Convert UPPER_SNAKE_CASE to lowercase-kebab-case
	lowerCode := strings.ToLower(strings.ReplaceAll(code, "_", "-"))
	return fmt.Sprintf("%s/%s", ProblemTypeBaseURL, lowerCode)
}

// GetTitle returns the human-readable title for an error code.
// Returns "Error" as a fallback if the code is not found.
func GetTitle(code string) string {
	if title, ok := ErrorTitles[code]; ok {
		return title
	}
	return "Error"
}

// WithValidationErrors adds validation errors to an AppError.
// This is used for validation error responses that need field-level details.
// Returns the same AppError for method chaining.
func (e *AppError) WithValidationErrors(errors []ValidationError) *AppError {
	e.ValidationErrors = errors
	return e
}
