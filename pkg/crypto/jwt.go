// Package crypto provides cryptographic utilities for JWT token management and password hashing.
//
// This package handles JWT token generation, validation, and parsing with custom claims.
// It supports both access tokens (short-lived) and refresh tokens (long-lived) with
// different expiry durations for web and mobile platforms.
//
// Example usage:
//
//	// Generate access token
//	token, err := crypto.GenerateAccessToken(userID, "organizer", secret, 15*time.Minute)
//
//	// Validate and parse token
//	claims, err := crypto.ParseToken(tokenString, secret)
//	if err != nil {
//		// Handle invalid/expired token
//	}
//
//	// Generate refresh token for web
//	refreshToken, err := crypto.GenerateRefreshToken(userID, "attendee", secret, 168*time.Hour)
package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType represents the type of JWT token (access or refresh).
type TokenType string

const (
	// TokenTypeAccess represents an access token (short-lived, used for API authentication).
	TokenTypeAccess TokenType = "access"

	// TokenTypeRefresh represents a refresh token (long-lived, used to obtain new access tokens).
	TokenTypeRefresh TokenType = "refresh"
)

// Common JWT errors
var (
	// ErrInvalidToken indicates the token is malformed or has invalid signature.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken indicates the token has expired.
	ErrExpiredToken = errors.New("token has expired")

	// ErrInvalidClaims indicates the token claims are invalid or missing required fields.
	ErrInvalidClaims = errors.New("invalid token claims")

	// ErrEmptySecret indicates the JWT secret is empty.
	ErrEmptySecret = errors.New("jwt secret cannot be empty")

	// ErrEmptyUserID indicates the user ID is empty.
	ErrEmptyUserID = errors.New("user id cannot be empty")

	// ErrInvalidExpiry indicates the expiry duration is invalid.
	ErrInvalidExpiry = errors.New("expiry duration must be positive")
)

// Claims represents the custom JWT claims for ezqrin tokens.
// It embeds jwt.RegisteredClaims to include standard claims like expiry and issuer.
type Claims struct {
	jwt.RegisteredClaims // Standard JWT claims (iss, sub, exp, iat, etc.)

	UserID    uuid.UUID `json:"user_id"`    // Unique identifier of the authenticated user
	Role      string    `json:"role"`       // User role (e.g., "organizer", "attendee")
	TokenType TokenType `json:"token_type"` // Type of token (access or refresh)
}

// GenerateAccessToken creates a new access token with the given parameters.
// Access tokens are short-lived (typically 15 minutes) and used for API authentication.
//
// Parameters:
//   - userID: UUID of the user as string
//   - role: User role (e.g., "organizer", "attendee")
//   - secret: Secret key for signing the token
//   - expiry: Duration until token expires (e.g., 15*time.Minute)
//
// Returns the signed JWT token string or an error if generation fails.
func GenerateAccessToken(userID, role, secret string, expiry time.Duration) (string, error) {
	return generateToken(userID, role, secret, expiry, TokenTypeAccess)
}

// GenerateRefreshToken creates a new refresh token with the given parameters.
// Refresh tokens are long-lived and used to obtain new access tokens without re-authentication.
// Different expiry durations are used for web (7 days) and mobile (90 days) platforms.
//
// Parameters:
//   - userID: UUID of the user as string
//   - role: User role (e.g., "organizer", "attendee")
//   - secret: Secret key for signing the token
//   - expiry: Duration until token expires (e.g., 168*time.Hour for web, 2160*time.Hour for mobile)
//
// Returns the signed JWT token string or an error if generation fails.
func GenerateRefreshToken(userID, role, secret string, expiry time.Duration) (string, error) {
	return generateToken(userID, role, secret, expiry, TokenTypeRefresh)
}

// generateToken is a private helper function that creates and signs a JWT token.
// It validates inputs and generates a token with custom claims.
func generateToken(userID, role, secret string, expiry time.Duration, tokenType TokenType) (string, error) {
	// Validate inputs
	if secret == "" {
		return "", ErrEmptySecret
	}
	if userID == "" {
		return "", ErrEmptyUserID
	}
	if expiry <= 0 {
		return "", ErrInvalidExpiry
	}

	// Parse and validate UUID
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user id format: %w", err)
	}

	now := time.Now()

	// Create custom claims
	claims := &Claims{
		UserID:    parsedUserID,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ezqrin-server",
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token's signature and expiry without parsing claims.
// This is useful for quick validation checks without needing the full claims data.
//
// Parameters:
//   - tokenString: The JWT token string to validate
//   - secret: Secret key used to sign the token
//
// Returns an error if the token is invalid, expired, or has an invalid signature.
func ValidateToken(tokenString, secret string) error {
	// Validate inputs
	if secret == "" {
		return ErrEmptySecret
	}
	if tokenString == "" {
		return ErrInvalidToken
	}

	// Parse and validate token
	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrExpiredToken
		}
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	return nil
}

// ParseToken parses and validates a JWT token, returning the custom claims.
// This function verifies the token signature, checks expiry, and extracts claims.
//
// Parameters:
//   - tokenString: The JWT token string to parse
//   - secret: Secret key used to sign the token
//
// Returns the parsed Claims or an error if validation fails.
func ParseToken(tokenString, secret string) (*Claims, error) {
	// Validate inputs
	if secret == "" {
		return nil, ErrEmptySecret
	}
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Parse token with claims
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract and validate claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Validate required claim fields
	if claims.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: user_id is missing or invalid", ErrInvalidClaims)
	}

	return claims, nil
}
