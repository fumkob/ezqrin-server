package repository

import (
	"context"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/mock_user_repository.go -package=mocks . UserRepository

// UserRepository defines the interface for user data persistence operations.
// Following Clean Architecture, this interface is defined in the domain layer
// and implemented in the infrastructure layer.
type UserRepository interface {
	BaseRepository

	// Create creates a new user in the database.
	// Returns an error if the email already exists (unique constraint violation).
	Create(ctx context.Context, user *entity.User) error

	// FindByID retrieves a user by their unique ID.
	// Returns ErrNotFound if the user does not exist.
	// Excludes password_hash from the result for security.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)

	// FindByEmail retrieves a user by their email address.
	// Returns ErrNotFound if no user with the email exists.
	// Excludes password_hash from the result for security.
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByEmailWithPassword retrieves a user by email including password hash.
	// This should only be used for authentication purposes.
	// Returns ErrNotFound if no user with the email exists.
	FindByEmailWithPassword(ctx context.Context, email string) (*entity.User, error)

	// Update updates an existing user's information.
	// Returns ErrNotFound if the user does not exist.
	Update(ctx context.Context, user *entity.User) error

	// List retrieves a paginated list of users.
	// Returns the users and the total count of users matching the criteria.
	// Excludes soft-deleted users from the results.
	// Excludes password_hash from the results for security.
	List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error)

	// SoftDelete marks a user as deleted without removing from database.
	// Sets deleted_at timestamp, deleted_by user ID, and anonymizes PII.
	// Returns ErrNotFound if the user does not exist.
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error

	// ExistsByEmail checks if a user with the given email exists.
	// Returns true if a user exists, false otherwise.
	// Includes soft-deleted users in the check.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
