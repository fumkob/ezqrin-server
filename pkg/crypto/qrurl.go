package crypto

import (
	"encoding/base64"
	"strings"
)

// GenerateQRDistributionURL creates a distribution URL for a QR code token.
// The token is base64url-encoded (RFC 4648, no padding) and appended to the base URL.
// Returns empty string if baseURL or qrToken is empty.
func GenerateQRDistributionURL(baseURL, qrToken string) string {
	if baseURL == "" || qrToken == "" {
		return ""
	}

	baseURL = strings.TrimRight(baseURL, "/")
	encoded := base64.RawURLEncoding.EncodeToString([]byte(qrToken))
	return baseURL + "/qr/" + encoded
}
