package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// TokenBytes is the number of random bytes to generate for a token.
	// 24 bytes of entropy produces ~32 character base64url-encoded tokens
	// which provides sufficient uniqueness (2^192 possible tokens).
	TokenBytes = 24
)

// Token errors
var (
	// ErrTokenGeneration indicates random token generation failed.
	ErrTokenGeneration = errors.New("failed to generate random token")
)

// GenerateToken generates a cryptographically secure random token.
// The token is URL-safe and base64-encoded, suitable for use in URLs and QR codes.
//
// The function uses crypto/rand to generate 24 bytes of random data (2^192 entropy),
// then encodes it using base64url encoding (RFC 4648), resulting in a ~32 character
// token containing only alphanumeric characters, hyphens, and underscores.
//
// Returns a URL-safe base64-encoded token string (~32 characters) or an error
// if random number generation fails.
func GenerateToken() (string, error) {
	// Create buffer for random bytes
	bytes := make([]byte, TokenBytes)

	// Generate cryptographically secure random bytes
	// crypto/rand.Read fills the entire slice and only returns an error
	// if the system's random number generator fails (extremely rare).
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("%w: %w", ErrTokenGeneration, err)
	}

	// Encode to base64url format (RFC 4648)
	// URL-safe encoding uses - and _ instead of + and /
	// No padding (=) is added, making it cleaner for URLs
	token := base64.RawURLEncoding.EncodeToString(bytes)

	return token, nil
}
