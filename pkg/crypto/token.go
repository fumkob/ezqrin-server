package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	// TOKEN_BYTES is the number of random bytes to generate for a token.
	// 24 bytes of entropy produces ~32 character base64url-encoded tokens
	// which provides sufficient uniqueness (2^192 possible tokens).
	TOKEN_BYTES = 24

	// tokenDelimiter separates the random token and HMAC signature.
	tokenDelimiter = "."
)

// Token errors
var (
	// ErrTokenGeneration indicates random token generation failed.
	ErrTokenGeneration = errors.New("failed to generate random token")

	// ErrInvalidHMACToken indicates the token format is invalid for HMAC operations.
	ErrInvalidHMACToken = errors.New("invalid HMAC token format")
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
	bytes := make([]byte, TOKEN_BYTES)

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

// GenerateHMACSignedToken generates a cryptographically secure random token
// and signs it with HMAC-SHA256 using the provided secret.
//
// Format: {base64url_random}.{base64url_hmac_sha256_of_random}
//
// Parameters:
//   - secret: The HMAC signing secret (must be non-empty)
//
// Returns the signed token string or an error.
func GenerateHMACSignedToken(secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("%w: secret cannot be empty", ErrInvalidHMACToken)
	}

	// Generate random base token
	rawToken, err := GenerateToken()
	if err != nil {
		return "", err
	}

	// Compute HMAC-SHA256 signature of the raw token
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return rawToken + tokenDelimiter + signature, nil
}

// GenerateParticipantQRToken generates a structured QR token for a participant.
// Format: evt_{event_id[:8]}_prt_{participant_id[:8]}_{random_12hex}.{base64url_hmac_sha256}
//
// Parameters:
//   - eventID: The UUID of the event
//   - participantID: The UUID of the participant
//   - secret: The HMAC signing secret (must be non-empty)
//
// Returns the signed structured token string or an error.
func GenerateParticipantQRToken(eventID, participantID uuid.UUID, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("%w: secret cannot be empty", ErrInvalidHMACToken)
	}

	// Generate 6 random bytes (= 12 hex characters for the random part)
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("%w: %w", ErrTokenGeneration, err)
	}
	randomPart := hex.EncodeToString(randomBytes)

	// Build structured raw token
	rawToken := fmt.Sprintf("evt_%s_prt_%s_%s",
		eventID.String()[:8],
		participantID.String()[:8],
		randomPart,
	)

	// Sign with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return rawToken + tokenDelimiter + signature, nil
}

// VerifyHMACToken verifies that the signed token was generated with the given secret.
// Returns true if the token is valid, false otherwise.
//
// Parameters:
//   - secret: The HMAC signing secret
//   - signedToken: The token to verify (format: {raw}.{signature})
func VerifyHMACToken(secret, signedToken string) bool {
	if secret == "" || signedToken == "" {
		return false
	}

	// Split the signed token into raw token and signature
	delimIdx := strings.LastIndex(signedToken, tokenDelimiter)
	if delimIdx == -1 {
		return false
	}

	rawToken := signedToken[:delimIdx]
	providedSig := signedToken[delimIdx+1:]

	if rawToken == "" || providedSig == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawToken))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(providedSig), []byte(expectedSig))
}
