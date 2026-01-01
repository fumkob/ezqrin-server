# Common Packages

This directory contains reusable utility packages that are shared across the ezQRin server application.

## Overview

| Package | Description | Key Dependencies |
|---------|-------------|------------------|
| `logger` | Structured logging with context support | `go.uber.org/zap` |
| `errors` | Application error types with HTTP status codes | stdlib `errors`, `net/http` |
| `validator` | Request validation with formatted error messages | `github.com/go-playground/validator/v10` |

## Logger

Provides structured logging using Zap with support for:
- Multiple log levels (debug, info, warn, error)
- JSON output for production, console for development
- Request ID tracking via context
- Configurable based on environment

**Usage:**
```go
import "github.com/fumkob/ezqrin-server/pkg/logger"

// Initialize
cfg := logger.Config{
    Level:       "info",
    Format:      "json",
    Environment: "production",
}
log, _ := logger.New(cfg)
defer log.Sync()

// Log with context
ctx := logger.ContextWithRequestID(ctx, "req-123")
log.WithContext(ctx).Info("processing request",
    zap.String("user_id", userID),
)
```

## Errors

Provides application-specific error types with:
- HTTP status code mapping
- Error codes for API responses
- Error wrapping with context
- Type checking utilities

**Usage:**
```go
import "github.com/fumkob/ezqrin-server/pkg/errors"

// Create errors
return errors.NotFound("user not found")
return errors.Validationf("invalid email: %s", email)

// Wrap errors
if err := repo.GetUser(id); err != nil {
    return errors.Wrap(err, "failed to get user")
}

// Check error types
if errors.IsNotFound(err) {
    // Handle not found
}

// Extract status code
statusCode := errors.GetStatusCode(err) // 404
```

**Error Types:**
- `NotFound` (404) - Resource not found
- `Validation` (400) - Invalid input
- `Unauthorized` (401) - Authentication required
- `Forbidden` (403) - Insufficient permissions
- `Conflict` (409) - Resource conflict (e.g., duplicate email)
- `Internal` (500) - Internal server error
- `BadRequest` (400) - Invalid request
- `TooManyRequests` (429) - Rate limit exceeded
- `ServiceUnavailable` (503) - Service unavailable

## Validator

Provides request validation with:
- Struct validation using tags
- Custom validators (UUID v4, email format)
- User-friendly error messages
- Helper functions for common validations

**Usage:**
```go
import "github.com/fumkob/ezqrin-server/pkg/validator"

// Initialize
v := validator.New()

// Validate struct
type CreateUserRequest struct {
    Email    string `validate:"required,email_format"`
    Password string `validate:"required,min=8,max=72"`
    Name     string `validate:"required,min=2,max=100"`
}

if err := v.Validate(req); err != nil {
    return err // "email must be a valid email address; password must be at least 8 characters"
}

// Helper functions
if err := validator.ValidateEmail(email); err != nil {
    return err
}

if err := validator.ValidateUUID(id); err != nil {
    return err
}
```

**Custom Validators:**
- `uuid4` - Valid UUID version 4
- `email_format` - Valid email address format

## Design Principles

All packages follow Clean Architecture principles:

1. **Independence**: No dependencies on internal application code
2. **Reusability**: Can be used across different layers
3. **Testability**: Easy to test in isolation
4. **Simplicity**: Single responsibility, clear interfaces
5. **Standards**: Follow CLAUDE.md Go coding standards

## Testing

Tests for these packages will be implemented separately by the test-writer agent following Ginkgo/Gomega BDD patterns.

## Dependencies

Added to `go.mod`:
- `go.uber.org/zap` v1.27.1 - Structured logging
- `github.com/go-playground/validator/v10` v10.27.0 - Already present (used by Gin)
- `github.com/google/uuid` v1.6.0 - Already present

All other dependencies are from the Go standard library.
