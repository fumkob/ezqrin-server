package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/fumkob/ezqrin-server/internal/interface/api/response"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// Recovery is a middleware that recovers from panics and logs the stack trace.
// It returns a 500 Internal Server Error response to the client and logs
// the panic details for debugging.
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID for correlation
				requestID, _ := c.Get("request_id")
				reqID := ""
				if id, ok := requestID.(string); ok {
					reqID = id
				}

				// Log panic with stack trace
				log.WithRequestID(reqID).Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)

				// Abort request with 500 error response
				c.Abort()

				// Send error response if headers not already sent
				if !c.Writer.Written() {
					response.Error(
						c,
						http.StatusInternalServerError,
						apperrors.CodeInternal,
						fmt.Sprintf("Internal server error: %v", err),
					)
				}
			}
		}()

		c.Next()
	}
}
