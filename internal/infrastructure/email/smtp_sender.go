package email

import (
	"context"
	"fmt"
	"net/smtp"

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
}

// NewSMTPSender creates a new SMTPSender.
func NewSMTPSender(host string, port int, user, password, fromAddress, fromName string) *SMTPSender {
	return &SMTPSender{
		host:        host,
		port:        port,
		user:        user,
		password:    password,
		fromAddress: fromAddress,
		fromName:    fromName,
	}
}

// Send sends an email message via SMTP.
func (s *SMTPSender) Send(_ context.Context, msg domainemail.Message) error {
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	raw, err := buildRFCMessage(s.fromAddress, s.fromName, msg)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.fromAddress, []string{msg.To}, raw)
}

var _ domainemail.Sender = (*SMTPSender)(nil)
