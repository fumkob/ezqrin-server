// Package middleware provides HTTP middleware for the Gin framework.
package middleware

import (
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header name for request ID
	RequestIDHeader = "X-Request-ID"
)

// RequestID is a middleware that generates a unique request ID for each request.
// If the client provides an X-Request-ID header, it will be used; otherwise,
// a new UUID is generated. The request ID is added to both the Gin context
// and the HTTP response headers for traceability.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client provided a request ID
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store request ID in Gin context for handlers
		c.Set("request_id", requestID)

		// Store request ID in request context for logging
		ctx := logger.ContextWithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Add request ID to response headers
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
