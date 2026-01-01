// Package response provides standard HTTP response structures for the API.
//
// This package defines consistent response formats for both success and error cases.
// Request IDs are provided in the X-Request-ID HTTP header (added by middleware),
// not in response bodies, following OpenAPI specification and industry standards.
//
// Example usage:
//
//	// Success response
//	response.Success(c, http.StatusOK, userData, "User retrieved successfully")
//
//	// Error response
//	response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
//
//	// Error from AppError
//	if err := userUsecase.GetUser(id); err != nil {
//		response.ErrorFromAppError(c, err)
//		return
//	}
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
)

// SuccessResponse represents a successful API response
// Request ID is provided in the X-Request-ID HTTP header, not in the response body
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
// Request ID is provided in the X-Request-ID HTTP header, not in the response body
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Success sends a successful JSON response with the given data and message
// Request ID is automatically added to response headers by the RequestID middleware
func Success(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Error sends an error JSON response with the given status code, error code, and message
func Error(c *gin.Context, statusCode int, errorCode, message string) {
	ErrorWithDetails(c, statusCode, errorCode, message, nil)
}

// ErrorWithDetails sends an error JSON response with additional details
// Request ID is automatically added to response headers by the RequestID middleware
func ErrorWithDetails(c *gin.Context, statusCode int, errorCode, message string, details interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}

// ErrorFromAppError sends an error response based on an AppError
func ErrorFromAppError(c *gin.Context, err error) {
	if err == nil {
		Success(c, http.StatusOK, nil, "")
		return
	}

	// Extract status code and error code from AppError
	statusCode := apperrors.GetStatusCode(err)
	errorCode := apperrors.GetErrorCode(err)

	// Get the error message
	message := err.Error()

	// Check if it's an AppError to extract clean message
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		message = appErr.Message
	}

	Error(c, statusCode, errorCode, message)
}

// ValidationErrors sends a validation error response with field-level details
func ValidationErrors(c *gin.Context, validationErrors interface{}) {
	ErrorWithDetails(
		c,
		http.StatusBadRequest,
		apperrors.CodeValidation,
		"Validation failed",
		validationErrors,
	)
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, apperrors.CodeInternal, message)
}
