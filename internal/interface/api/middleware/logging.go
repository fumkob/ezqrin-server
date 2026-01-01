package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// Logging is a middleware that logs HTTP request and response information.
// It logs the request method, path, status code, duration, and request ID
// using structured logging with zap.
func Logging(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		startTime := time.Now()

		// Get request ID from context
		requestID, _ := c.Get("request_id")
		reqID := ""
		if id, ok := requestID.(string); ok {
			reqID = id
		}

		// Log request start
		log.WithRequestID(reqID).Info("incoming request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log request completion with status and duration
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Int("response_size", c.Writer.Size()),
		}

		// Log at appropriate level based on status code
		statusCode := c.Writer.Status()
		if statusCode >= 500 {
			log.WithRequestID(reqID).Error("request completed with server error", fields...)
		} else if statusCode >= 400 {
			log.WithRequestID(reqID).Warn("request completed with client error", fields...)
		} else {
			log.WithRequestID(reqID).Info("request completed", fields...)
		}

		// Log any errors that were captured
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.WithRequestID(reqID).Error("request error",
					zap.Error(e.Err),
					zap.Any("type", e.Type),
				)
			}
		}
	}
}
