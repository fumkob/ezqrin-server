package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

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
	tokenSource oauth2.TokenSource
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

	return &GmailSender{
		fromAddress: fromAddress,
		fromName:    fromName,
		tokenSource: ts,
	}, nil
}

// Send sends an email via the Gmail API.
func (s *GmailSender) Send(ctx context.Context, msg domainemail.Message) error {
	svc, err := gmailapi.NewService(ctx, option.WithTokenSource(s.tokenSource))
	if err != nil {
		return fmt.Errorf("gmail: failed to create service: %w", err)
	}

	raw, err := s.buildRaw(msg)
	if err != nil {
		return fmt.Errorf("gmail: failed to build message: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(raw)
	gMsg := &gmailapi.Message{Raw: encoded}

	if _, err := svc.Users.Messages.Send("me", gMsg).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail: send failed: %w", err)
	}
	return nil
}

// buildRaw constructs RFC 2822 bytes suitable for the Gmail API.
func (s *GmailSender) buildRaw(msg domainemail.Message) ([]byte, error) {
	from := mime.QEncoding.Encode("utf-8", s.fromName) + " <" + s.fromAddress + ">"
	subject := mime.QEncoding.Encode("utf-8", msg.Subject)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	headers := []string{
		"MIME-Version: 1.0",
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + subject,
		"Content-Type: multipart/related; boundary=" + mw.Boundary(),
	}
	buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")

	// HTML part
	htmlH := textproto.MIMEHeader{}
	htmlH.Set("Content-Type", "text/html; charset=utf-8")
	htmlH.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := mw.CreatePart(htmlH)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprint(htmlPart, msg.Body); err != nil {
		return nil, err
	}

	// Inline attachments
	for _, att := range msg.Attachments {
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", att.ContentType+`; name="`+att.Filename+`"`)
		h.Set("Content-Transfer-Encoding", "base64")
		h.Set("Content-Disposition", `inline; filename="`+att.Filename+`"`)
		if att.ContentID != "" {
			h.Set("Content-Id", "<"+att.ContentID+">")
		}
		part, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err := fmt.Fprint(part, base64.StdEncoding.EncodeToString(att.Data)); err != nil {
			return nil, err
		}
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ domainemail.Sender = (*GmailSender)(nil)
