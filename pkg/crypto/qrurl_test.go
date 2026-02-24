package crypto_test

import (
	"encoding/base64"
	"strings"

	"github.com/fumkob/ezqrin-server/pkg/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QR Distribution URL", func() {
	Describe("GenerateQRDistributionURL", func() {
		When("generating a distribution URL", func() {
			Context("with valid inputs", func() {
				It("should return a URL with base64url-encoded token", func() {
					baseURL := "https://qr.example.com"
					qrToken := "evt_550e8400_prt_770e8400_a1b2c3d4e5f6.signatureHere"

					url := crypto.GenerateQRDistributionURL(baseURL, qrToken)

					Expect(url).To(HavePrefix("https://qr.example.com/qr/"))

					encoded := strings.TrimPrefix(url, "https://qr.example.com/qr/")
					decoded, err := base64.RawURLEncoding.DecodeString(encoded)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(decoded)).To(Equal(qrToken))
				})

				It("should produce URL-safe output with no padding", func() {
					baseURL := "https://qr.example.com"
					qrToken := "evt_550e8400_prt_770e8400_a1b2c3d4e5f6.sig+with/special=chars"

					url := crypto.GenerateQRDistributionURL(baseURL, qrToken)

					encoded := strings.TrimPrefix(url, "https://qr.example.com/qr/")
					Expect(encoded).NotTo(ContainSubstring("+"))
					Expect(encoded).NotTo(ContainSubstring("/"))
					Expect(encoded).NotTo(ContainSubstring("="))
				})

				It("should strip trailing slash from base URL", func() {
					baseURL := "https://qr.example.com/"
					qrToken := "test-token.sig"

					url := crypto.GenerateQRDistributionURL(baseURL, qrToken)

					Expect(url).To(HavePrefix("https://qr.example.com/qr/"))
					Expect(url).NotTo(ContainSubstring("//qr/"))
				})
			})

			Context("with empty base URL", func() {
				It("should return empty string", func() {
					url := crypto.GenerateQRDistributionURL("", "some-token.sig")
					Expect(url).To(BeEmpty())
				})
			})

			Context("with empty QR token", func() {
				It("should return empty string", func() {
					url := crypto.GenerateQRDistributionURL("https://qr.example.com", "")
					Expect(url).To(BeEmpty())
				})
			})
		})
	})
})
