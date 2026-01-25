package qrcode_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/qrcode"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QR Code Generator", func() {
	var (
		generator *qrcode.Generator
		ctx       context.Context
		token     string
	)

	BeforeEach(func() {
		generator = qrcode.NewGenerator()
		ctx = context.Background()
		token = "test-token-abc123-xyz789-foobar"
	})

	Describe("NewGenerator", func() {
		When("creating a new generator", func() {
			Context("with default settings", func() {
				It("should create generator successfully", func() {
					gen := qrcode.NewGenerator()
					Expect(gen).NotTo(BeNil())
				})
			})

			Context("with custom error correction", func() {
				It("should create generator with low error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionLow)
					Expect(gen).NotTo(BeNil())
				})

				It("should create generator with medium error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionMedium)
					Expect(gen).NotTo(BeNil())
				})

				It("should create generator with high error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionHigh)
					Expect(gen).NotTo(BeNil())
				})

				It("should create generator with highest error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionHighest)
					Expect(gen).NotTo(BeNil())
				})
			})
		})
	})

	Describe("GeneratePNG", func() {
		When("generating PNG QR codes", func() {
			Context("with valid input", func() {
				It("should generate PNG data successfully", func() {
					png, err := generator.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should generate valid PNG binary data", func() {
					png, err := generator.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					// PNG files start with signature: \x89PNG\r\n\x1a\n
					Expect(png[0:8]).To(Equal([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}),
						"PNG should have valid magic number")
				})

				It("should generate different QR codes for different tokens", func() {
					token1 := "token-one"
					token2 := "token-two"

					png1, err := generator.GeneratePNG(ctx, token1, 256)
					Expect(err).NotTo(HaveOccurred())

					png2, err := generator.GeneratePNG(ctx, token2, 256)
					Expect(err).NotTo(HaveOccurred())

					Expect(png1).NotTo(Equal(png2),
						"Different tokens should produce different QR codes")
				})

				It("should generate larger data for larger size", func() {
					smallPNG, err := generator.GeneratePNG(ctx, token, 128)
					Expect(err).NotTo(HaveOccurred())

					largePNG, err := generator.GeneratePNG(ctx, token, 512)
					Expect(err).NotTo(HaveOccurred())

					Expect(len(largePNG)).To(BeNumerically(">", len(smallPNG)),
						"Larger size should produce more bytes")
				})
			})

			Context("with different sizes", func() {
				It("should generate QR code with minimum size", func() {
					png, err := generator.GeneratePNG(ctx, token, 64)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should generate QR code with default size", func() {
					png, err := generator.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should generate QR code with maximum size", func() {
					png, err := generator.GeneratePNG(ctx, token, 2048)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should generate QR code with custom size", func() {
					png, err := generator.GeneratePNG(ctx, token, 300)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with empty token", func() {
				It("should return ErrEmptyToken", func() {
					png, err := generator.GeneratePNG(ctx, "", 256)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrEmptyToken)).To(BeTrue())
					Expect(png).To(BeEmpty())
				})
			})

			Context("with invalid size", func() {
				It("should return error for size below minimum", func() {
					png, err := generator.GeneratePNG(ctx, token, 32)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(png).To(BeEmpty())
				})

				It("should return error for size above maximum", func() {
					png, err := generator.GeneratePNG(ctx, token, 3000)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(png).To(BeEmpty())
				})

				It("should return error for zero size", func() {
					png, err := generator.GeneratePNG(ctx, token, 0)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(png).To(BeEmpty())
				})

				It("should return error for negative size", func() {
					png, err := generator.GeneratePNG(ctx, token, -100)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(png).To(BeEmpty())
				})
			})

			Context("with special tokens", func() {
				It("should handle token with URL-safe characters", func() {
					urlSafeToken := "abc-123_xyz-789_ABC-XYZ"

					png, err := generator.GeneratePNG(ctx, urlSafeToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle token with special characters", func() {
					specialToken := "token@example.com!#$%"

					png, err := generator.GeneratePNG(ctx, specialToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle long token", func() {
					longToken := strings.Repeat("a", 100)

					png, err := generator.GeneratePNG(ctx, longToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle short token", func() {
					shortToken := "x"

					png, err := generator.GeneratePNG(ctx, shortToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle numeric token", func() {
					numericToken := "1234567890"

					png, err := generator.GeneratePNG(ctx, numericToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})
		})
	})

	Describe("GeneratePNGBase64", func() {
		When("generating base64-encoded PNG QR codes", func() {
			Context("with valid input", func() {
				It("should generate base64 string successfully", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(encoded).NotTo(BeEmpty())
				})

				It("should generate valid base64 encoding", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())

					// Should be decodable
					decoded, err := base64.StdEncoding.DecodeString(encoded)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).NotTo(BeEmpty())
				})

				It("should decode to valid PNG data", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					// Decode base64
					decoded, err := base64.StdEncoding.DecodeString(encoded)
					Expect(err).NotTo(HaveOccurred())

					// Verify PNG signature
					Expect(decoded[0:8]).To(Equal([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}))
				})

				It("should match PNG generation output", func() {
					png, err := generator.GeneratePNG(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					base64String, err := generator.GeneratePNGBase64(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					// Decode base64 string
					decoded, err := base64.StdEncoding.DecodeString(base64String)
					Expect(err).NotTo(HaveOccurred())

					// Should match PNG output
					Expect(decoded).To(Equal(png))
				})

				It("should be usable in HTML data URI", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(encoded).NotTo(BeEmpty())

					// Should not contain whitespace or special chars
					Expect(encoded).NotTo(ContainSubstring("\n"))
					Expect(encoded).NotTo(ContainSubstring(" "))
					Expect(encoded).NotTo(ContainSubstring("\r"))
				})

				It("should generate different base64 for different tokens", func() {
					token1 := "token-alpha"
					token2 := "token-beta"

					encoded1, err := generator.GeneratePNGBase64(ctx, token1, 256)
					Expect(err).NotTo(HaveOccurred())

					encoded2, err := generator.GeneratePNGBase64(ctx, token2, 256)
					Expect(err).NotTo(HaveOccurred())

					Expect(encoded1).NotTo(Equal(encoded2))
				})
			})

			Context("with different sizes", func() {
				It("should generate longer base64 for larger sizes", func() {
					small, err := generator.GeneratePNGBase64(ctx, token, 128)
					Expect(err).NotTo(HaveOccurred())

					large, err := generator.GeneratePNGBase64(ctx, token, 512)
					Expect(err).NotTo(HaveOccurred())

					Expect(len(large)).To(BeNumerically(">", len(small)))
				})
			})

			Context("with empty token", func() {
				It("should return ErrEmptyToken", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, "", 256)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrEmptyToken)).To(BeTrue())
					Expect(encoded).To(BeEmpty())
				})
			})

			Context("with invalid size", func() {
				It("should return error for invalid size", func() {
					encoded, err := generator.GeneratePNGBase64(ctx, token, 3000)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(encoded).To(BeEmpty())
				})
			})
		})
	})

	Describe("GenerateSVG", func() {
		When("generating SVG QR codes", func() {
			Context("with valid input", func() {
				It("should generate SVG string successfully", func() {
					svg, err := generator.GenerateSVG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(svg).NotTo(BeEmpty())
				})

				It("should generate valid QR code representation", func() {
					svg, err := generator.GenerateSVG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					// The library returns ASCII art representation
					// Should contain block characters used for QR code
					Expect(len(svg)).To(BeNumerically(">", 100),
						"QR code representation should be substantial")
				})

				It("should generate different SVG for different tokens", func() {
					token1 := "svg-token-one"
					token2 := "svg-token-two"

					svg1, err := generator.GenerateSVG(ctx, token1, 256)
					Expect(err).NotTo(HaveOccurred())

					svg2, err := generator.GenerateSVG(ctx, token2, 256)
					Expect(err).NotTo(HaveOccurred())

					Expect(svg1).NotTo(Equal(svg2))
				})
			})

			Context("with different sizes", func() {
				It("should generate SVG with small size", func() {
					svg, err := generator.GenerateSVG(ctx, token, 64)

					Expect(err).NotTo(HaveOccurred())
					Expect(svg).NotTo(BeEmpty())
				})

				It("should generate SVG with large size", func() {
					svg, err := generator.GenerateSVG(ctx, token, 1024)

					Expect(err).NotTo(HaveOccurred())
					Expect(svg).NotTo(BeEmpty())
				})
			})

			Context("with empty token", func() {
				It("should return ErrEmptyToken", func() {
					svg, err := generator.GenerateSVG(ctx, "", 256)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrEmptyToken)).To(BeTrue())
					Expect(svg).To(BeEmpty())
				})
			})

			Context("with invalid size", func() {
				It("should return error for size out of range", func() {
					svg, err := generator.GenerateSVG(ctx, token, 5000)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, qrcode.ErrInvalidSize)).To(BeTrue())
					Expect(svg).To(BeEmpty())
				})
			})
		})
	})

	Describe("Error Correction Levels", func() {
		When("using different error correction levels", func() {
			Context("with low error correction", func() {
				It("should generate QR code with low error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionLow)

					png, err := gen.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with medium error correction", func() {
				It("should generate QR code with medium error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionMedium)

					png, err := gen.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with high error correction", func() {
				It("should generate QR code with high error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionHigh)

					png, err := gen.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with highest error correction", func() {
				It("should generate QR code with highest error correction", func() {
					gen := qrcode.NewGeneratorWithErrorCorrection(qrcode.ErrorCorrectionHighest)

					png, err := gen.GeneratePNG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with runtime error correction change", func() {
				It("should allow changing error correction level", func() {
					gen := qrcode.NewGenerator()

					// Generate with default (medium)
					png1, err := gen.GeneratePNG(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					// Change to highest
					gen.SetErrorCorrection(qrcode.ErrorCorrectionHighest)

					// Generate with new setting
					png2, err := gen.GeneratePNG(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					// QR codes should be different due to different error correction
					Expect(png1).NotTo(Equal(png2))
				})
			})
		})
	})

	Describe("Concurrent Generation", func() {
		When("generating QR codes concurrently", func() {
			Context("with multiple goroutines", func() {
				It("should safely generate PNG QR codes concurrently", func() {
					const numGoroutines = 20
					results := make(chan []byte, numGoroutines)
					errors := make(chan error, numGoroutines)

					for i := range numGoroutines {
						go func(index int) {
							token := "concurrent-token-" + fmt.Sprintf("%d", index)
							png, err := generator.GeneratePNG(ctx, token, 256)
							if err != nil {
								errors <- err
							} else {
								results <- png
							}
						}(i)
					}

					// Collect results
					qrCodes := make([][]byte, 0, numGoroutines)
					for range numGoroutines {
						select {
						case png := <-results:
							qrCodes = append(qrCodes, png)
						case err := <-errors:
							Fail("Unexpected error: " + err.Error())
						}
					}

					Expect(qrCodes).To(HaveLen(numGoroutines))
				})

				It("should safely generate base64 QR codes concurrently", func() {
					const numGoroutines = 20
					results := make(chan string, numGoroutines)
					errors := make(chan error, numGoroutines)

					for i := range numGoroutines {
						go func(index int) {
							token := "base64-token-" + fmt.Sprintf("%d", index)
							encoded, err := generator.GeneratePNGBase64(ctx, token, 256)
							if err != nil {
								errors <- err
							} else {
								results <- encoded
							}
						}(i)
					}

					// Collect results
					encodedQRs := make([]string, 0, numGoroutines)
					for range numGoroutines {
						select {
						case encoded := <-results:
							encodedQRs = append(encodedQRs, encoded)
						case err := <-errors:
							Fail("Unexpected error: " + err.Error())
						}
					}

					Expect(encodedQRs).To(HaveLen(numGoroutines))
				})
			})
		})
	})

	Describe("QR Code Properties", func() {
		When("examining QR code characteristics", func() {
			Context("with data capacity", func() {
				It("should handle token up to reasonable length", func() {
					// 32 character token (typical for our use case)
					token32 := strings.Repeat("a", 32)

					png, err := generator.GeneratePNG(ctx, token32, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle longer token", func() {
					// 64 character token
					token64 := strings.Repeat("a", 64)

					png, err := generator.GeneratePNG(ctx, token64, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})

				It("should handle very long token", func() {
					// 200 character token
					token200 := strings.Repeat("a", 200)

					png, err := generator.GeneratePNG(ctx, token200, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
				})
			})

			Context("with consistency", func() {
				It("should generate identical QR code for same input", func() {
					png1, err := generator.GeneratePNG(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					png2, err := generator.GeneratePNG(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					Expect(png1).To(Equal(png2),
						"Same input should produce identical QR code")
				})

				It("should generate consistent base64 for same input", func() {
					encoded1, err := generator.GeneratePNGBase64(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					encoded2, err := generator.GeneratePNGBase64(ctx, token, 256)
					Expect(err).NotTo(HaveOccurred())

					Expect(encoded1).To(Equal(encoded2))
				})
			})
		})
	})

	Describe("Real-world Use Cases", func() {
		When("simulating real-world scenarios", func() {
			Context("with participant check-in flow", func() {
				It("should generate QR code for participant token", func() {
					participantToken := "PTK-abc123-xyz789-000001"

					png, err := generator.GeneratePNG(ctx, participantToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())

					// Verify it's valid PNG
					Expect(png[0:8]).To(Equal([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}))
				})

				It("should generate base64 for web display", func() {
					participantToken := "PTK-web-display-token"

					base64String, err := generator.GeneratePNGBase64(ctx, participantToken, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(base64String).NotTo(BeEmpty())

					// Should be valid base64
					_, err = base64.StdEncoding.DecodeString(base64String)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with different event requirements", func() {
				It("should generate small QR code for email", func() {
					token := "EMAIL-TOKEN-123"

					png, err := generator.GeneratePNG(ctx, token, 128)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
					// Smaller files for email
					Expect(len(png)).To(BeNumerically("<", 10000))
				})

				It("should generate large QR code for printing", func() {
					token := "PRINT-TOKEN-456"

					png, err := generator.GeneratePNG(ctx, token, 1024)

					Expect(err).NotTo(HaveOccurred())
					Expect(png).NotTo(BeEmpty())
					// Verify size is valid PNG
					Expect(png[0:8]).To(Equal([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}))
				})

				It("should generate SVG for scalable display", func() {
					token := "SCALABLE-TOKEN-789"

					svg, err := generator.GenerateSVG(ctx, token, 256)

					Expect(err).NotTo(HaveOccurred())
					Expect(svg).NotTo(BeEmpty())
				})
			})
		})
	})
})
