package crypto

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the bcrypt cost factor (12 = 2^12 rounds).
	// Higher cost = more secure but slower. Cost 12 provides good balance
	// between security and performance for production use.
	DefaultCost = 12

	// MaxPasswordLength is the maximum password length for bcrypt (72 bytes).
	// Bcrypt silently truncates passwords longer than 72 bytes.
	MaxPasswordLength = 72
)

// Bcrypt errors
var (
	// ErrEmptyPassword indicates the password string is empty.
	ErrEmptyPassword = errors.New("password cannot be empty")

	// ErrPasswordTooLong indicates the password exceeds bcrypt's 72-byte limit.
	// Bcrypt silently truncates passwords longer than 72 bytes, which could lead to security issues.
	ErrPasswordTooLong = errors.New("password cannot exceed 72 bytes")

	// ErrHashMismatch indicates the provided password doesn't match the hash.
	ErrHashMismatch = errors.New("password does not match hash")
)

// HashPassword hashes a password using bcrypt with the default cost factor.
// The cost factor determines the computational complexity of the hash (12 = 2^12 rounds).
// Higher cost makes brute-force attacks more difficult but increases computation time.
//
// Bcrypt automatically generates a random salt and includes it in the hash output.
// The returned hash is safe to store in a database as it contains both salt and hash.
//
// Parameters:
//   - password: The plaintext password to hash
//
// Returns the bcrypt hash string or an error if hashing fails.
func HashPassword(password string) (string, error) {
	// Validate password
	if password == "" {
		return "", ErrEmptyPassword
	}

	// Bcrypt has a maximum password length of 72 bytes
	// Longer passwords are silently truncated which could cause security issues
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// Generate bcrypt hash
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// ComparePassword compares a plaintext password with a bcrypt hash.
// Returns nil if the password matches the hash, or an error if it doesn't match
// or if comparison fails.
//
// This function is constant-time to prevent timing attacks that could
// reveal information about the password or hash.
//
// Parameters:
//   - hashedPassword: The bcrypt hash to compare against
//   - password: The plaintext password to verify
//
// Returns nil if passwords match, ErrHashMismatch if they don't match,
// or another error if validation fails.
func ComparePassword(hashedPassword, password string) error {
	// Validate inputs
	if password == "" {
		return ErrEmptyPassword
	}

	if hashedPassword == "" {
		return fmt.Errorf("hashed password cannot be empty")
	}

	// Bcrypt has a maximum password length of 72 bytes
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	// Compare password with hash
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrHashMismatch
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}

	return nil
}
