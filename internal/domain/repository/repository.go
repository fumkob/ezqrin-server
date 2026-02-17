// Package repository defines repository interfaces for domain entities.
// Following Clean Architecture, repositories are defined in the domain layer
// and implemented in the infrastructure layer.
package repository

import (
	"context"
	"errors"
)

// Common repository errors
var (
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("not found")
)

// Transactor defines the interface for managing database transactions.
// Repositories can use this to ensure atomic operations across multiple
// repository calls.
type Transactor interface {
	// WithTransaction executes the given function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// BaseRepository defines common repository operations.
// This is intentionally minimal - specific repositories extend this with
// their domain-specific methods.
type BaseRepository interface {
	// HealthCheck verifies the repository's underlying data store is accessible.
	HealthCheck(ctx context.Context) error
}
