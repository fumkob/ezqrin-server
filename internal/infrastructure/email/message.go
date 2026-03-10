package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
)

const (
	mimeTextPlain = "text/plain; charset=utf-8"
	mimeTextHTML  = "text/html; charset=utf-8"
	mimeQP        = "quoted-printable"
	mimeBase64    = "base64"
)

// buildRFCMessage constructs RFC 2822 message bytes from a Message.
// Shared by SMTPSender and GmailSender.
//
//   - TextBody == "": legacy multipart/related (HTML + optional inline attachments).
//   - TextBody != "" and Body == "": plain text only (no HTML).
//   - TextBody != "" and Body != "": multipart/alternative (text/plain fallback + text/html).
func buildRFCMessage(fromAddress, fromName string, msg domainemail.Message) ([]byte, error) {
	from := mime.QEncoding.Encode("utf-8", fromName) + " <" + fromAddress + ">"
	subject := mime.QEncoding.Encode("utf-8", msg.Subject)

	if msg.TextBody == "" {
		return buildRelatedMessage(from, subject, msg)
	}
	if msg.Body == "" {
		return buildPlainTextMessage(from, subject, msg)
	}
	return buildAlternativeMessage(from, subject, msg)
}

// buildPlainTextMessage constructs a simple text/plain message with no HTML part.
func buildPlainTextMessage(from, subject string, msg domainemail.Message) ([]byte, error) {
	var buf bytes.Buffer
	headers := []string{
		"MIME-Version: 1.0",
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + subject,
		"Content-Type: " + mimeTextPlain,
		"Content-Transfer-Encoding: " + mimeQP,
	}
	buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")
	buf.WriteString(msg.TextBody)
	return buf.Bytes(), nil
}

// buildRelatedMessage constructs a legacy multipart/related message (HTML + optional inline attachments).
func buildRelatedMessage(from, subject string, msg domainemail.Message) ([]byte, error) {
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

	if err := writeHTMLPart(mw, msg.Body); err != nil {
		return nil, err
	}
	if err := writeAttachmentParts(mw, msg.Attachments); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildAlternativeMessage constructs a multipart/alternative message with a plain-text fallback and an HTML part.
// When attachments are present, the HTML part is wrapped in a nested multipart/related.
func buildAlternativeMessage(from, subject string, msg domainemail.Message) ([]byte, error) {
	var buf bytes.Buffer
	altWriter := multipart.NewWriter(&buf)

	headers := []string{
		"MIME-Version: 1.0",
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + subject,
		"Content-Type: multipart/alternative; boundary=" + altWriter.Boundary(),
	}
	buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")

	if err := writeTextPart(altWriter, msg.TextBody); err != nil {
		return nil, err
	}
	if err := writeHTMLOrRelatedPart(altWriter, msg); err != nil {
		return nil, err
	}
	if err := altWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeHTMLOrRelatedPart writes the HTML part to parent, wrapping it in a nested
// multipart/related when inline attachments are present.
func writeHTMLOrRelatedPart(parent *multipart.Writer, msg domainemail.Message) error {
	if len(msg.Attachments) == 0 {
		return writeHTMLPart(parent, msg.Body)
	}
	return writeRelatedPart(parent, msg)
}

// writeRelatedPart writes a nested multipart/related part (HTML + inline attachments) to parent.
func writeRelatedPart(parent *multipart.Writer, msg domainemail.Message) error {
	relBoundary := multipart.NewWriter(io.Discard).Boundary() // generate a random boundary
	relH := textproto.MIMEHeader{}
	relH.Set("Content-Type", "multipart/related; boundary="+relBoundary)
	relPart, err := parent.CreatePart(relH)
	if err != nil {
		return err
	}
	innerWriter := multipart.NewWriter(relPart)
	if err := innerWriter.SetBoundary(relBoundary); err != nil {
		return err
	}
	if err := writeHTMLPart(innerWriter, msg.Body); err != nil {
		return err
	}
	if err := writeAttachmentParts(innerWriter, msg.Attachments); err != nil {
		return err
	}
	return innerWriter.Close()
}

// writeBodyPart writes a quoted-printable MIME part with the given content type to mw.
func writeBodyPart(mw *multipart.Writer, contentType, body string) error {
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", mimeQP)
	part, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(part, body)
	return err
}

func writeTextPart(mw *multipart.Writer, body string) error {
	return writeBodyPart(mw, mimeTextPlain, body)
}

func writeHTMLPart(mw *multipart.Writer, body string) error {
	return writeBodyPart(mw, mimeTextHTML, body)
}

// writeAttachmentParts writes each attachment as an inline MIME part to mw.
func writeAttachmentParts(mw *multipart.Writer, attachments []domainemail.Attachment) error {
	for _, att := range attachments {
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", att.ContentType+`; name="`+att.Filename+`"`)
		h.Set("Content-Transfer-Encoding", mimeBase64)
		h.Set("Content-Disposition", `inline; filename="`+att.Filename+`"`)
		if att.ContentID != "" {
			h.Set("Content-Id", "<"+att.ContentID+">")
		}
		part, err := mw.CreatePart(h)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(part, base64.StdEncoding.EncodeToString(att.Data)); err != nil {
			return err
		}
	}
	return nil
}
