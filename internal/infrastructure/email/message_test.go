// Package email provides white-box tests for unexported message-building helpers.
// The package declaration must match the production code so that unexported symbols
// (buildRFCMessage and friends) are reachable from this test file.
package email

import (
	"encoding/base64"
	"mime"
	"strings"

	domainemail "github.com/fumkob/ezqrin-server/internal/domain/email"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ─── helpers ────────────────────────────────────────────────────────────────

const (
	testFromAddress = "sender@example.com"
	testFromName    = "Test Sender"
	testTo          = "recipient@example.com"
	testSubject     = "Hello World"
	testHTMLBody    = "<p>Hello <b>World</b></p>"
	testTextBody    = "Hello World"
)

// encodedFrom returns the expected From header value for the test constants.
func encodedFrom() string {
	return mime.QEncoding.Encode("utf-8", testFromName) + " <" + testFromAddress + ">"
}

// encodedSubject returns the expected encoded Subject value for testSubject.
func encodedSubject() string {
	return mime.QEncoding.Encode("utf-8", testSubject)
}

// msgStr is a convenience wrapper that calls buildRFCMessage and returns the
// result as a string, failing the spec on error.
func msgStr(fromAddr, fromName string, msg domainemail.Message) string {
	raw, err := buildRFCMessage(fromAddr, fromName, msg)
	Expect(err).NotTo(HaveOccurred(), "buildRFCMessage should not return an error")
	return string(raw)
}

// ─── specs ──────────────────────────────────────────────────────────────────

var _ = Describe("buildRFCMessage", func() {

	// ── plain-text only ──────────────────────────────────────────────────────

	When("building a plain-text-only message", func() {
		Context("with TextBody set and no HTML Body or Attachments", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:       testTo,
					Subject:  testSubject,
					TextBody: testTextBody,
					// Body intentionally left empty
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should include the correct MIME-Version header", func() {
				Expect(output).To(ContainSubstring("MIME-Version: 1.0"))
			})

			It("should include the encoded From header", func() {
				Expect(output).To(ContainSubstring("From: " + encodedFrom()))
			})

			It("should include the To header", func() {
				Expect(output).To(ContainSubstring("To: " + testTo))
			})

			It("should include the encoded Subject header", func() {
				Expect(output).To(ContainSubstring("Subject: " + encodedSubject()))
			})

			It("should declare text/plain content type", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/plain; charset=utf-8"))
			})

			It("should declare quoted-printable transfer encoding", func() {
				Expect(output).To(ContainSubstring("Content-Transfer-Encoding: quoted-printable"))
			})

			It("should embed the text body verbatim", func() {
				Expect(output).To(ContainSubstring(testTextBody))
			})

			It("should not contain any multipart boundary", func() {
				Expect(output).NotTo(ContainSubstring("multipart/"))
			})
		})
	})

	// ── HTML-only (legacy multipart/related) ─────────────────────────────────

	When("building an HTML-only message", func() {
		Context("with Body set and no TextBody or Attachments", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:      testTo,
					Subject: testSubject,
					Body:    testHTMLBody,
					// TextBody intentionally left empty → legacy multipart/related
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should include the correct MIME-Version header", func() {
				Expect(output).To(ContainSubstring("MIME-Version: 1.0"))
			})

			It("should include the encoded From header", func() {
				Expect(output).To(ContainSubstring("From: " + encodedFrom()))
			})

			It("should include the To header", func() {
				Expect(output).To(ContainSubstring("To: " + testTo))
			})

			It("should use multipart/related as the outer content type", func() {
				Expect(output).To(ContainSubstring("Content-Type: multipart/related;"))
			})

			It("should include text/html content type for the HTML part", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/html; charset=utf-8"))
			})

			It("should embed the HTML body", func() {
				Expect(output).To(ContainSubstring(testHTMLBody))
			})
		})
	})

	// ── HTML + text fallback (multipart/alternative) ─────────────────────────

	When("building a message with both HTML and plain-text fallback", func() {
		Context("with Body and TextBody set and no Attachments", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:       testTo,
					Subject:  testSubject,
					Body:     testHTMLBody,
					TextBody: testTextBody,
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should use multipart/alternative as the outer content type", func() {
				Expect(output).To(ContainSubstring("Content-Type: multipart/alternative;"))
			})

			It("should include the text/plain part", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/plain; charset=utf-8"))
			})

			It("should include the text/html part", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/html; charset=utf-8"))
			})

			It("should embed the plain-text body", func() {
				Expect(output).To(ContainSubstring(testTextBody))
			})

			It("should embed the HTML body", func() {
				Expect(output).To(ContainSubstring(testHTMLBody))
			})

			It("should declare quoted-printable transfer encoding for body parts", func() {
				Expect(output).To(ContainSubstring("Content-Transfer-Encoding: quoted-printable"))
			})
		})
	})

	// ── HTML-only with inline attachment (multipart/related) ─────────────────

	When("building an HTML message with an inline attachment", func() {
		Context("with Body and one Attachment but no TextBody", func() {
			var (
				output     string
				attachment domainemail.Attachment
			)

			BeforeEach(func() {
				attachment = domainemail.Attachment{
					Filename:    "logo.png",
					ContentType: "image/png",
					Data:        []byte("fake-png-data"),
					ContentID:   "logo",
				}
				msg := domainemail.Message{
					To:          testTo,
					Subject:     testSubject,
					Body:        testHTMLBody,
					Attachments: []domainemail.Attachment{attachment},
					// TextBody intentionally left empty
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should use multipart/related as the outer content type", func() {
				Expect(output).To(ContainSubstring("Content-Type: multipart/related;"))
			})

			It("should include the attachment content type and filename", func() {
				Expect(output).To(ContainSubstring(`Content-Type: image/png; name="logo.png"`))
			})

			It("should use base64 transfer encoding for the attachment", func() {
				Expect(output).To(ContainSubstring("Content-Transfer-Encoding: base64"))
			})

			It("should include the Content-Disposition inline header with filename", func() {
				Expect(output).To(ContainSubstring(`Content-Disposition: inline; filename="logo.png"`))
			})

			It("should include the Content-Id header with the correct value", func() {
				Expect(output).To(ContainSubstring("Content-Id: <logo>"))
			})

			It("should encode the attachment data in base64", func() {
				encoded := base64.StdEncoding.EncodeToString([]byte("fake-png-data"))
				Expect(output).To(ContainSubstring(encoded))
			})
		})
	})

	// ── HTML + text fallback + inline attachment (nested multipart) ──────────

	When("building a message with text fallback and an inline attachment", func() {
		Context("with Body, TextBody, and one Attachment", func() {
			var output string

			BeforeEach(func() {
				attachment := domainemail.Attachment{
					Filename:    "qr.png",
					ContentType: "image/png",
					Data:        []byte("qr-image-bytes"),
					ContentID:   "qrcode",
				}
				msg := domainemail.Message{
					To:          testTo,
					Subject:     testSubject,
					Body:        testHTMLBody,
					TextBody:    testTextBody,
					Attachments: []domainemail.Attachment{attachment},
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should use multipart/alternative as the outer content type", func() {
				Expect(output).To(ContainSubstring("Content-Type: multipart/alternative;"))
			})

			It("should nest a multipart/related part inside the alternative", func() {
				Expect(output).To(ContainSubstring("Content-Type: multipart/related;"))
			})

			It("should include the text/plain part", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/plain; charset=utf-8"))
			})

			It("should include the text/html part", func() {
				Expect(output).To(ContainSubstring("Content-Type: text/html; charset=utf-8"))
			})

			It("should embed the inline attachment with base64 encoding", func() {
				Expect(output).To(ContainSubstring("Content-Transfer-Encoding: base64"))
			})

			It("should include the Content-Id header for the inline attachment", func() {
				Expect(output).To(ContainSubstring("Content-Id: <qrcode>"))
			})

			It("should contain the base64-encoded attachment data", func() {
				encoded := base64.StdEncoding.EncodeToString([]byte("qr-image-bytes"))
				Expect(output).To(ContainSubstring(encoded))
			})
		})
	})

	// ── attachment without ContentID ─────────────────────────────────────────

	When("building a message with an attachment that has no ContentID", func() {
		Context("with an Attachment whose ContentID is empty", func() {
			var output string

			BeforeEach(func() {
				attachment := domainemail.Attachment{
					Filename:    "report.pdf",
					ContentType: "application/pdf",
					Data:        []byte("pdf-content"),
					ContentID:   "", // intentionally empty
				}
				msg := domainemail.Message{
					To:          testTo,
					Subject:     testSubject,
					Body:        testHTMLBody,
					Attachments: []domainemail.Attachment{attachment},
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should not include a Content-Id header", func() {
				Expect(output).NotTo(ContainSubstring("Content-Id:"))
			})

			It("should still include the attachment with correct content type", func() {
				Expect(output).To(ContainSubstring(`Content-Type: application/pdf; name="report.pdf"`))
			})
		})
	})

	// ── non-ASCII sender name ─────────────────────────────────────────────────

	When("building a message with a non-ASCII sender display name", func() {
		Context("with a Japanese sender name", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:       testTo,
					Subject:  testSubject,
					TextBody: testTextBody,
				}
				output = msgStr(testFromAddress, "テスト送信者", msg)
			})

			It("should Q-encode the From display name", func() {
				// The encoded name must appear as a Q-encoded word (=?utf-8?q?...?=)
				Expect(output).To(MatchRegexp(`From: =\?utf-8\?[qQbB]\?`))
			})
		})
	})

	// ── non-ASCII subject ─────────────────────────────────────────────────────

	When("building a message with a non-ASCII subject", func() {
		Context("with a Japanese subject line", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:       testTo,
					Subject:  "テスト件名",
					TextBody: testTextBody,
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should Q-encode the Subject header", func() {
				Expect(output).To(MatchRegexp(`Subject: =\?utf-8\?[qQbB]\?`))
			})

			It("should not include the raw non-ASCII subject verbatim", func() {
				Expect(output).NotTo(ContainSubstring("テスト件名"))
			})
		})
	})

	// ── header ordering ──────────────────────────────────────────────────────

	When("examining the header block of a plain-text message", func() {
		Context("with all standard fields populated", func() {
			var output string

			BeforeEach(func() {
				msg := domainemail.Message{
					To:       testTo,
					Subject:  testSubject,
					TextBody: testTextBody,
				}
				output = msgStr(testFromAddress, testFromName, msg)
			})

			It("should place headers before the blank-line body separator", func() {
				// Headers and body are separated by \r\n\r\n
				parts := strings.SplitN(output, "\r\n\r\n", 2)
				Expect(parts).To(HaveLen(2))
				headerBlock := parts[0]
				Expect(headerBlock).To(ContainSubstring("MIME-Version: 1.0"))
				Expect(headerBlock).To(ContainSubstring("From: "))
				Expect(headerBlock).To(ContainSubstring("To: "))
				Expect(headerBlock).To(ContainSubstring("Subject: "))
			})
		})
	})
})
