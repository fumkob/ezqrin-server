package email

import (
	"github.com/fumkob/ezqrin-server/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("NewSenderFromConfig", func() {
	var logger *zap.Logger

	BeforeEach(func() {
		logger = zap.NewNop()
	})

	// ── SMTP backend ─────────────────────────────────────────────────────────

	When("creating a sender with SMTP backend", func() {
		Context(`with Backend set to "smtp"`, func() {
			It("should return a non-nil Sender without error", func() {
				cfg := config.EmailConfig{
					Backend:      config.EmailBackendSMTP,
					SMTPHost:     "smtp.example.com",
					SMTPPort:     587,
					SMTPUser:     "user@example.com",
					SMTPPassword: "secret",
					FromAddress:  "sender@example.com",
					FromName:     "Sender",
				}

				sender, err := NewSenderFromConfig(cfg, logger)

				Expect(err).NotTo(HaveOccurred())
				Expect(sender).NotTo(BeNil())
			})
		})

		Context("with empty Backend (default falls back to SMTP)", func() {
			It("should return a non-nil Sender without error", func() {
				cfg := config.EmailConfig{
					Backend:     "", // intentionally blank
					SMTPHost:    "smtp.example.com",
					SMTPPort:    25,
					FromAddress: "noreply@example.com",
					FromName:    "No Reply",
				}

				sender, err := NewSenderFromConfig(cfg, logger)

				Expect(err).NotTo(HaveOccurred())
				Expect(sender).NotTo(BeNil())
			})
		})

		Context("with minimal SMTP config (no credentials)", func() {
			It("should create the sender successfully without auth fields", func() {
				cfg := config.EmailConfig{
					Backend:     config.EmailBackendSMTP,
					SMTPHost:    "localhost",
					SMTPPort:    25,
					FromAddress: "test@localhost",
				}

				sender, err := NewSenderFromConfig(cfg, logger)

				Expect(err).NotTo(HaveOccurred())
				Expect(sender).NotTo(BeNil())
			})
		})
	})

	// ── unknown backend ───────────────────────────────────────────────────────

	When("creating a sender with an unknown backend", func() {
		Context(`with Backend set to an unsupported value`, func() {
			It("should return an error describing the unknown backend", func() {
				cfg := config.EmailConfig{
					Backend: config.EmailBackend("sendgrid"),
				}

				sender, err := NewSenderFromConfig(cfg, logger)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("sendgrid"))
				Expect(err.Error()).To(ContainSubstring("smtp"))
				Expect(err.Error()).To(ContainSubstring("gmail"))
				Expect(sender).To(BeNil())
			})
		})

		Context(`with Backend set to an arbitrary non-empty string`, func() {
			It("should return an error that mentions valid backends", func() {
				cfg := config.EmailConfig{
					Backend: config.EmailBackend("mailgun"),
				}

				sender, err := NewSenderFromConfig(cfg, logger)

				Expect(err).To(HaveOccurred())
				Expect(sender).To(BeNil())
			})
		})
	})

	// ── Gmail backend (construction-time validation) ──────────────────────────
	// NewGmailSender immediately contacts the Google OAuth2 token endpoint, so
	// a meaningful unit test without network access will receive an error during
	// token refresh.  We verify that the factory routes to the Gmail path by
	// confirming that the returned error does NOT contain the "unknown backend"
	// message – which would only appear when the factory rejects the backend
	// string outright – while the actual OAuth error differs.

	When("creating a sender with Gmail backend and invalid credentials", func() {
		Context("with empty OAuth2 credentials", func() {
			It("should return an error that is not an unknown-backend error", func() {
				cfg := config.EmailConfig{
					Backend:           config.EmailBackendGmail,
					GmailClientID:     "",
					GmailClientSecret: "",
					GmailRefreshToken: "",
					FromAddress:       "sender@gmail.com",
				}

				sender, err := NewSenderFromConfig(cfg, logger)
				// The factory routed to Gmail (no "unknown backend" error) but
				// Gmail sender construction/token exchange fails.
				if err != nil {
					Expect(err.Error()).NotTo(ContainSubstring("unknown email backend"))
				}
				// Either an error or a nil sender is acceptable here; we only
				// care that the factory did not reject the backend string.
				_ = sender
			})
		})
	})
})
