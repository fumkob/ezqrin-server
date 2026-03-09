package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
)

// SMTPSender sends emails via SMTP using net/smtp (standard library, no extra dependencies).
type SMTPSender struct {
	host        string
	port        int
	user        string
	password    string
	fromAddress string
	fromName    string
	useTLS      bool
}

// NewSMTPSender creates a new SMTPSender.
func NewSMTPSender(host string, port int, user, password, fromAddress, fromName string, useTLS bool) *SMTPSender {
	return &SMTPSender{
		host:        host,
		port:        port,
		user:        user,
		password:    password,
		fromAddress: fromAddress,
		fromName:    fromName,
		useTLS:      useTLS,
	}
}

// Send sends an email message via SMTP.
func (s *SMTPSender) Send(_ context.Context, msg domainemail.Message) error {
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	raw, err := s.buildRaw(msg)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.fromAddress, []string{msg.To}, raw)
}

// buildRaw constructs the full RFC 2822 message bytes.
func (s *SMTPSender) buildRaw(msg domainemail.Message) ([]byte, error) {
	from := mime.QEncoding.Encode("utf-8", s.fromName) + " <" + s.fromAddress + ">"
	subject := mime.QEncoding.Encode("utf-8", msg.Subject)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// Headers
	headers := []string{
		"MIME-Version: 1.0",
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + subject,
		"Content-Type: multipart/related; boundary=" + mw.Boundary(),
	}
	buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")

	// HTML part
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := mw.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprint(htmlPart, msg.Body); err != nil {
		return nil, err
	}

	// Inline attachments (e.g. embedded QR code image)
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

var _ domainemail.Sender = (*SMTPSender)(nil)
