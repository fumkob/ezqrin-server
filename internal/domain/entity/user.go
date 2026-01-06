// Package entity defines core business entities and their validation rules.
package entity

import (
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/pkg/validator"
	"github.com/google/uuid"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	// RoleAdmin has full system access including user management
	RoleAdmin UserRole = "admin"
	// RoleOrganizer can create and manage events
	RoleOrganizer UserRole = "organizer"
	// RoleStaff can be assigned to events for check-in operations
	RoleStaff UserRole = "staff"
)

// Validation constants for User entity
const (
	UserNameMinLength = 2
	UserNameMaxLength = 255
)

// Common validation errors for User entity
var (
	ErrUserEmailRequired     = errors.New("email is required")
	ErrUserEmailInvalid      = errors.New("email format is invalid")
	ErrUserPasswordRequired  = errors.New("password hash is required")
	ErrUserNameRequired      = errors.New("name is required")
	ErrUserNameTooShort      = errors.New("name must be at least 2 characters")
	ErrUserNameTooLong       = errors.New("name must not exceed 255 characters")
	ErrUserRoleRequired      = errors.New("role is required")
	ErrUserRoleInvalid       = errors.New("role must be one of: admin, organizer, staff")
	ErrUserAlreadyDeleted    = errors.New("user is already deleted")
	ErrUserNotDeleted        = errors.New("user is not deleted")
	ErrUserAlreadyAnonymized = errors.New("user is already anonymized")
)

// User represents a system user who can create and manage events
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         UserRole
	DeletedAt    *time.Time // Soft delete timestamp
	DeletedBy    *uuid.UUID // User who performed deletion
	IsAnonymized bool       // PII anonymization flag
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate validates the User entity fields
func (u *User) Validate() error {
	// Email validation
	if u.Email == "" {
		return ErrUserEmailRequired
	}
	if err := validator.ValidateEmail(u.Email); err != nil {
		return ErrUserEmailInvalid
	}

	// Password hash validation
	if u.PasswordHash == "" {
		return ErrUserPasswordRequired
	}

	// Name validation
	if u.Name == "" {
		return ErrUserNameRequired
	}
	if len(u.Name) < UserNameMinLength {
		return ErrUserNameTooShort
	}
	if len(u.Name) > UserNameMaxLength {
		return ErrUserNameTooLong
	}

	// Role validation
	if u.Role == "" {
		return ErrUserRoleRequired
	}
	if !u.IsValidRole() {
		return ErrUserRoleInvalid
	}

	return nil
}

// IsValidRole checks if the user's role is one of the valid roles
func (u *User) IsValidRole() bool {
	switch u.Role {
	case RoleAdmin, RoleOrganizer, RoleStaff:
		return true
	default:
		return false
	}
}

// IsDeleted returns true if the user has been soft deleted
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsOrganizer returns true if the user has organizer role
func (u *User) IsOrganizer() bool {
	return u.Role == RoleOrganizer
}

// IsStaff returns true if the user has staff role
func (u *User) IsStaff() bool {
	return u.Role == RoleStaff
}

// CanManageEvents returns true if the user can create and manage events
func (u *User) CanManageEvents() bool {
	return u.IsAdmin() || u.IsOrganizer()
}

// ValidateRole validates if a role string is valid
func ValidateRole(role string) error {
	switch UserRole(role) {
	case RoleAdmin, RoleOrganizer, RoleStaff:
		return nil
	default:
		return ErrUserRoleInvalid
	}
}
