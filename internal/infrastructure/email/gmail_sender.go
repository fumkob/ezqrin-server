package email

import (
	"context"
	"encoding/base64"
	"fmt"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailSender sends emails via the Gmail API using OAuth2 with a refresh token.
// The refresh token is obtained once via the OAuth2 consent flow and stored in config.
type GmailSender struct {
	fromAddress string
	fromName    string
	svc         *gmailapi.Service
}

// NewGmailSender creates a new GmailSender.
// clientID, clientSecret, and refreshToken come from Google Cloud Console / OAuth2 consent.
// fromAddress must be the Gmail address that owns the OAuth2 credentials.
func NewGmailSender(clientID, clientSecret, refreshToken, fromAddress, fromName string) (*GmailSender, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmailapi.GmailSendScope},
	}
	token := &oauth2.Token{RefreshToken: refreshToken}
	ts := cfg.TokenSource(context.Background(), token)

	// Validate credentials eagerly by fetching an access token.
	if _, err := ts.Token(); err != nil {
		return nil, fmt.Errorf("gmail: invalid OAuth2 credentials: %w", err)
	}

	svc, err := gmailapi.NewService(context.Background(), option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("gmail: failed to create service: %w", err)
	}

	return &GmailSender{
		fromAddress: fromAddress,
		fromName:    fromName,
		svc:         svc,
	}, nil
}

// Send sends an email via the Gmail API.
func (s *GmailSender) Send(ctx context.Context, msg domainemail.Message) error {
	raw, err := buildRFCMessage(s.fromAddress, s.fromName, msg)
	if err != nil {
		return fmt.Errorf("gmail: failed to build message: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(raw)
	gMsg := &gmailapi.Message{Raw: encoded}

	if _, err := s.svc.Users.Messages.Send("me", gMsg).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail: send failed: %w", err)
	}
	return nil
}

var _ domainemail.Sender = (*GmailSender)(nil)
