package email

import (
	"fmt"

	"github.com/fumkob/ezqrin-server/config"
	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
)

// NewSenderFromConfig creates a Sender based on the Email configuration.
// Backend "smtp" (default): uses net/smtp.
// Backend "gmail": uses Gmail API v1 with OAuth2 refresh token.
func NewSenderFromConfig(cfg config.EmailConfig) (domainemail.Sender, error) {
	switch cfg.Backend {
	case config.EmailBackendGmail:
		return NewGmailSender(
			cfg.GmailClientID,
			cfg.GmailClientSecret,
			cfg.GmailRefreshToken,
			cfg.FromAddress,
			cfg.FromName,
		)
	case config.EmailBackendSMTP, "":
		return NewSMTPSender(
			cfg.SMTPHost,
			cfg.SMTPPort,
			cfg.SMTPUser,
			cfg.SMTPPassword,
			cfg.FromAddress,
			cfg.FromName,
			cfg.SMTPTLS,
		), nil
	default:
		return nil, fmt.Errorf("unknown email backend %q: must be \"smtp\" or \"gmail\"", cfg.Backend)
	}
}
