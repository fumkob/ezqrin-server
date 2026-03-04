package participant_test

import (
	"encoding/base64"
	"strings"

	"github.com/fumkob/ezqrin-server/pkg/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QR Distribution URL Population", func() {
	When("QR_HOSTING_BASE_URL is configured", func() {
		Context("with a participant that has a QR code", func() {
			It("should generate qr_distribution_url from QR code token", func() {
				baseURL := "https://qr.example.com"
				qrToken := "evt_abc_prt_def_random.signature"

				result := crypto.GenerateQRDistributionURL(baseURL, qrToken)

				Expect(result).NotTo(BeEmpty())
				Expect(result).To(HavePrefix("https://qr.example.com/qr/"))
			})

			It("should produce a URL that decodes back to the original QR token", func() {
				baseURL := "https://qr.example.com"
				qrToken := "evt_550e8400_prt_770e8400_a1b2c3d4e5f6.signatureHere"

				result := crypto.GenerateQRDistributionURL(baseURL, qrToken)

				encoded := strings.TrimPrefix(result, "https://qr.example.com/qr/")
				decoded, err := base64.RawURLEncoding.DecodeString(encoded)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(decoded)).To(Equal(qrToken))
			})

			It("should strip a trailing slash from the base URL", func() {
				baseURL := "https://qr.example.com/"
				qrToken := "evt_abc_prt_def_random.signature"

				result := crypto.GenerateQRDistributionURL(baseURL, qrToken)

				Expect(result).To(HavePrefix("https://qr.example.com/qr/"))
				Expect(result).NotTo(ContainSubstring("//qr/"))
			})

			It("should produce a URL-safe base64 encoded token with no padding", func() {
				baseURL := "https://qr.example.com"
				qrToken := "evt_abc_prt_def_random.sig+with/special=chars"

				result := crypto.GenerateQRDistributionURL(baseURL, qrToken)

				encoded := strings.TrimPrefix(result, "https://qr.example.com/qr/")
				Expect(encoded).NotTo(ContainSubstring("+"))
				Expect(encoded).NotTo(ContainSubstring("/"))
				Expect(encoded).NotTo(ContainSubstring("="))
			})
		})
	})

	When("QR_HOSTING_BASE_URL is empty", func() {
		Context("with a participant that has a QR code", func() {
			It("should return empty qr_distribution_url", func() {
				result := crypto.GenerateQRDistributionURL("", "some-token")

				Expect(result).To(BeEmpty())
			})
		})
	})

	When("QR code token is empty", func() {
		It("should return empty qr_distribution_url", func() {
			result := crypto.GenerateQRDistributionURL("https://qr.example.com", "")

			Expect(result).To(BeEmpty())
		})
	})

	When("both QR_HOSTING_BASE_URL and QR code token are empty", func() {
		It("should return empty qr_distribution_url", func() {
			result := crypto.GenerateQRDistributionURL("", "")

			Expect(result).To(BeEmpty())
		})
	})
})
