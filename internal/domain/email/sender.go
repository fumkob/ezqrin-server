// Package email defines the interface for sending emails.
package email

import "context"

// Message represents an outgoing email message.
type Message struct {
	To      string
	Subject string
	// Body is the HTML body of the email.
	Body string
	// Attachments holds inline or attached files.
	Attachments []Attachment
}

// Attachment represents an email attachment or inline file.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
	// ContentID enables inline embedding via cid: references in HTML (e.g., <img src="cid:qrcode">).
	ContentID string
}

// Sender defines the interface for sending emails.
// Implementations include SMTPSender and GmailSender.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
