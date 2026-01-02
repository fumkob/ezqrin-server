package middleware

import (
	"net/http"
	"strings"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/gin-gonic/gin"
)

// CORS is a middleware that handles Cross-Origin Resource Sharing (CORS).
// It configures allowed origins, methods, headers, and credentials based
// on the provided configuration.
func CORS(cfg *config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, cfg.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
			// Allow all origins if configured with wildcard
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// Set allowed methods
		if len(cfg.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		}

		// Set allowed headers
		if len(cfg.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		}

		// Set allow credentials
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set max age for preflight cache (24 hours)
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if the given origin is in the list of allowed origins.
// It supports wildcard matching for subdomains (e.g., "*.example.com").
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomain matching (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
