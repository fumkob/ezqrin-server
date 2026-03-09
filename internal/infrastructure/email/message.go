package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
)

// buildRFCMessage constructs RFC 2822 multipart/related message bytes from a Message.
// Shared by SMTPSender and GmailSender.
func buildRFCMessage(fromAddress, fromName string, msg domainemail.Message) ([]byte, error) {
	from := mime.QEncoding.Encode("utf-8", fromName) + " <" + fromAddress + ">"
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
