// Package response provides RFC 9457 Problem Details responses and direct data responses.
//
// This package implements RFC 9457 (Problem Details for HTTP APIs) for error responses
// and returns data directly (without wrappers) for successful responses.
//
// Request IDs are provided in the X-Request-ID HTTP header (added by middleware),
// not in response bodies, following OpenAPI specification and industry standards.
//
// Example usage:
//
//	// Success response (single entity)
//	response.Data(c, http.StatusOK, userData)
//
//	// Success response (collection with pagination)
//	response.List(c, http.StatusOK, events, paginationMeta)
//
//	// Empty success response
//	response.NoContent(c)
//
//	// Error response from AppError
//	response.ProblemFromError(c, err)
//
//	// Validation error
//	response.ValidationProblem(c, validationErrors)
package response

import (
	"errors"
	"net/http"

	"github.com/fumkob/ezqrin-server/internal/interface/api/generated"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/gin-gonic/gin"
)

// ProblemDetails represents an RFC 9457 Problem Details response
type ProblemDetails struct {
	Type     string                      `json:"type"`             // URI reference identifying the problem type
	Title    string                      `json:"title"`            // Short, human-readable summary
	Status   int                         `json:"status"`           // HTTP status code
	Detail   string                      `json:"detail"`           // Human-readable explanation
	Instance string                      `json:"instance"`         // URI reference identifying the specific occurrence
	Code     string                      `json:"code,omitempty"`   // Extension: Application error code
	Errors   []generated.ValidationError `json:"errors,omitempty"` // Extension: Validation errors array
}

// ListResponse represents a paginated list response with data and metadata
type ListResponse struct {
	Data interface{}              `json:"data"`
	Meta generated.PaginationMeta `json:"meta"`
}

// Data sends a successful JSON response with the data directly (no wrapper)
func Data(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// List sends a successful JSON response with paginated data
func List(c *gin.Context, statusCode int, data interface{}, meta generated.PaginationMeta) {
	c.JSON(statusCode, ListResponse{
		Data: data,
		Meta: meta,
	})
}

// NoContent sends a 204 No Content response with no body
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Problem sends an RFC 9457 Problem Details error response
func Problem(c *gin.Context, statusCode int, problemType, title, detail string) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(statusCode, ProblemDetails{
		Type:     problemType,
		Title:    title,
		Status:   statusCode,
		Detail:   detail,
		Instance: c.Request.URL.Path,
	})
}

// ProblemWithCode sends an RFC 9457 Problem Details error response with error code extension
func ProblemWithCode(c *gin.Context, statusCode int, code, detail string) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(statusCode, ProblemDetails{
		Type:     apperrors.ToTypeURL(code),
		Title:    apperrors.GetTitle(code),
		Status:   statusCode,
		Detail:   detail,
		Instance: c.Request.URL.Path,
		Code:     code,
	})
}

// ProblemFromError sends an RFC 9457 Problem Details response based on an AppError
func ProblemFromError(c *gin.Context, err error) {
	if err == nil {
		NoContent(c)
		return
	}

	// Extract status code and error code from AppError
	statusCode := apperrors.GetStatusCode(err)
	errorCode := apperrors.GetErrorCode(err)

	// Get the error message
	message := err.Error()

	// Check if it's an AppError to extract details
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		message = appErr.Message

		// If validation errors exist, include them
		if len(appErr.ValidationErrors) > 0 {
			c.Header("Content-Type", "application/problem+json")
			c.JSON(statusCode, ProblemDetails{
				Type:     apperrors.ToTypeURL(errorCode),
				Title:    apperrors.GetTitle(errorCode),
				Status:   statusCode,
				Detail:   message,
				Instance: c.Request.URL.Path,
				Code:     errorCode,
				Errors:   appErr.ValidationErrors,
			})
			return
		}
	}

	ProblemWithCode(c, statusCode, errorCode, message)
}

// ValidationProblem sends an RFC 9457 Problem Details validation error response
func ValidationProblem(c *gin.Context, validationErrors []generated.ValidationError) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(http.StatusBadRequest, ProblemDetails{
		Type:     apperrors.ToTypeURL(apperrors.CodeValidation),
		Title:    apperrors.GetTitle(apperrors.CodeValidation),
		Status:   http.StatusBadRequest,
		Detail:   "One or more validation errors occurred",
		Instance: c.Request.URL.Path,
		Code:     apperrors.CodeValidation,
		Errors:   validationErrors,
	})
}

// InternalProblem sends an RFC 9457 Problem Details 500 Internal Server Error response
func InternalProblem(c *gin.Context, detail string) {
	ProblemWithCode(c, http.StatusInternalServerError, apperrors.CodeInternal, detail)
}

// NotFoundProblem sends an RFC 9457 Problem Details 404 Not Found response
func NotFoundProblem(c *gin.Context, detail string) {
	ProblemWithCode(c, http.StatusNotFound, apperrors.CodeNotFound, detail)
}

// UnauthorizedProblem sends an RFC 9457 Problem Details 401 Unauthorized response
func UnauthorizedProblem(c *gin.Context, detail string) {
	ProblemWithCode(c, http.StatusUnauthorized, apperrors.CodeUnauthorized, detail)
}

// ForbiddenProblem sends an RFC 9457 Problem Details 403 Forbidden response
func ForbiddenProblem(c *gin.Context, detail string) {
	ProblemWithCode(c, http.StatusForbidden, apperrors.CodeForbidden, detail)
}
