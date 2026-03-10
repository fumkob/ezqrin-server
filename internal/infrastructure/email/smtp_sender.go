package email

import (
	"context"
	"fmt"
	"net/smtp"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	"go.uber.org/zap"
)

// SMTPSender sends emails via SMTP using net/smtp (standard library, no extra dependencies).
type SMTPSender struct {
	host        string
	port        int
	user        string
	password    string
	fromAddress string
	fromName    string
	logger      *zap.Logger
}

// NewSMTPSender creates a new SMTPSender.
func NewSMTPSender(
	host string,
	port int,
	user, password, fromAddress, fromName string,
	logger *zap.Logger,
) *SMTPSender {
	return &SMTPSender{
		host:        host,
		port:        port,
		user:        user,
		password:    password,
		fromAddress: fromAddress,
		fromName:    fromName,
		logger:      logger,
	}
}

// Send sends an email message via SMTP.
func (s *SMTPSender) Send(_ context.Context, msg domainemail.Message) error {
	s.logger.Info("attempting smtp send",
		zap.String("host", s.host),
		zap.Int("port", s.port),
		zap.String("user", s.user),
		zap.String("to", msg.To),
	)

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	raw, err := buildRFCMessage(s.fromAddress, s.fromName, msg)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	if err := smtp.SendMail(addr, auth, s.fromAddress, []string{msg.To}, raw); err != nil {
		s.logger.Error("smtp send failed", zap.String("to", msg.To), zap.Error(err))
		return err
	}

	s.logger.Info("smtp email sent", zap.String("to", msg.To))
	return nil
}

var _ domainemail.Sender = (*SMTPSender)(nil)
