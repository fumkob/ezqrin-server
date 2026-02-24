package crypto_test

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"

	"github.com/fumkob/ezqrin-server/pkg/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token Generation", func() {
	Describe("GenerateToken", func() {
		When("generating a token", func() {
			Context("with successful generation", func() {
				It("should generate a non-empty token", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(BeEmpty())
				})

				It("should generate token with expected length", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())
					// 24 bytes = 32 base64url characters (without padding)
					Expect(len(token)).To(Equal(32))
				})

				It("should generate URL-safe characters only", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())

					// URL-safe base64 uses: A-Z, a-z, 0-9, -, _
					urlSafePattern := regexp.MustCompile(`^[A-Za-z0-9\-_]+$`)
					Expect(urlSafePattern.MatchString(token)).To(BeTrue(),
						"Token should contain only URL-safe characters (alphanumeric, -, _)")
				})

				It("should not contain padding characters", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(ContainSubstring("="),
						"Token should not contain base64 padding (=)")
				})

				It("should not contain unsafe URL characters", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(ContainSubstring("+"))
					Expect(token).NotTo(ContainSubstring("/"))
					Expect(token).NotTo(ContainSubstring(" "))
				})

				It("should be valid base64url encoding", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())

					// Attempt to decode the token
					decoded, err := base64.RawURLEncoding.DecodeString(token)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(HaveLen(24), "Decoded token should be 24 bytes")
				})
			})

			Context("with uniqueness", func() {
				It("should generate different tokens on successive calls", func() {
					token1, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					token2, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					Expect(token1).NotTo(Equal(token2),
						"Successive token generations should produce different tokens")
				})

				It("should generate unique tokens across multiple generations", func() {
					const numTokens = 100
					tokens := make([]string, numTokens)

					// Generate multiple tokens
					for i := range numTokens {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())
						tokens[i] = token
					}

					// Verify all tokens are unique
					tokenMap := make(map[string]bool)
					for _, token := range tokens {
						Expect(tokenMap[token]).To(BeFalse(),
							"All tokens should be unique - duplicate found: %s", token)
						tokenMap[token] = true
					}

					Expect(tokenMap).To(HaveLen(numTokens))
				})

				It("should have high entropy - no common prefixes", func() {
					const numTokens = 20
					tokens := make([]string, numTokens)

					for i := range numTokens {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())
						tokens[i] = token
					}

					// Check first 4 characters for diversity
					prefixes := make(map[string]int)
					for _, token := range tokens {
						prefix := token[:4]
						prefixes[prefix]++
					}

					// With high entropy, most prefixes should be unique
					// Allow some collisions but expect mostly unique prefixes
					Expect(len(prefixes)).To(BeNumerically(">=", numTokens*8/10),
						"Tokens should have diverse prefixes indicating high entropy")
				})
			})

			Context("with cryptographic properties", func() {
				It("should have balanced character distribution", func() {
					const numTokens = 100
					charCounts := make(map[rune]int)

					for range numTokens {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())

						for _, char := range token {
							charCounts[char]++
						}
					}

					// With 100 tokens of 32 chars = 3200 chars total
					// Expected average per unique char depends on how many appear
					// Just verify we have good variety of characters
					Expect(len(charCounts)).To(BeNumerically(">=", 40),
						"Should use a good variety of characters from the base64url alphabet")
				})

				It("should be unpredictable - no sequential patterns", func() {
					token1, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					token2, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					token3, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					// Tokens should not be sequentially related
					Expect(token1).NotTo(Equal(token2))
					Expect(token2).NotTo(Equal(token3))
					Expect(token1).NotTo(Equal(token3))

					// Calculate byte-level similarity
					similarity12 := calculateTokenSimilarity(token1, token2)
					similarity23 := calculateTokenSimilarity(token2, token3)

					// Random tokens should have low similarity (around 1/64 = ~1.5% for base64)
					Expect(similarity12).To(BeNumerically("<", 0.3))
					Expect(similarity23).To(BeNumerically("<", 0.3))
				})
			})

			Context("with large volume generation", func() {
				It("should consistently generate valid tokens in volume", func() {
					const numTokens = 1000
					tokens := make([]string, numTokens)

					for i := range numTokens {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())
						Expect(token).NotTo(BeEmpty())
						Expect(len(token)).To(Equal(32))
						tokens[i] = token
					}

					// Verify uniqueness of all 1000 tokens
					tokenMap := make(map[string]bool)
					for _, token := range tokens {
						Expect(tokenMap[token]).To(BeFalse())
						tokenMap[token] = true
					}
				})
			})
		})

		When("examining token format", func() {
			Context("with base64url specification", func() {
				It("should comply with RFC 4648 base64url encoding", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())

					// RFC 4648 base64url uses: A-Z a-z 0-9 - _
					// No padding when using RawURLEncoding
					validChars := regexp.MustCompile(`^[A-Za-z0-9\-_]+$`)
					Expect(validChars.MatchString(token)).To(BeTrue())
				})

				It("should be decodable back to 24 bytes", func() {
					token, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					decoded, err := base64.RawURLEncoding.DecodeString(token)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(HaveLen(24))

					// Re-encode and verify it matches
					reencoded := base64.RawURLEncoding.EncodeToString(decoded)
					Expect(reencoded).To(Equal(token))
				})
			})

			Context("with URL safety", func() {
				It("should be safe for use in URL paths", func() {
					token, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					// Characters that need encoding in URLs
					unsafeChars := []string{"+", "/", "=", " ", "?", "&", "#", "%"}
					for _, unsafeChar := range unsafeChars {
						Expect(token).NotTo(ContainSubstring(unsafeChar),
							"Token should not contain URL-unsafe character: %s", unsafeChar)
					}
				})

				It("should be safe for use in URL query parameters", func() {
					token, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					// Should not need URL encoding
					queryUnsafeChars := []string{"&", "=", "?", " ", "+", "#", "%"}
					for _, unsafeChar := range queryUnsafeChars {
						Expect(token).NotTo(ContainSubstring(unsafeChar))
					}
				})
			})
		})

		When("testing concurrent generation", func() {
			Context("with multiple goroutines", func() {
				It("should safely generate tokens concurrently", func() {
					const numGoroutines = 50
					tokens := make(chan string, numGoroutines)
					errors := make(chan error, numGoroutines)

					// Generate tokens concurrently
					for range numGoroutines {
						go func() {
							token, err := crypto.GenerateToken()
							if err != nil {
								errors <- err
							} else {
								tokens <- token
							}
						}()
					}

					// Collect results
					generatedTokens := make([]string, 0, numGoroutines)
					for range numGoroutines {
						select {
						case token := <-tokens:
							generatedTokens = append(generatedTokens, token)
						case err := <-errors:
							Fail("Unexpected error: " + err.Error())
						}
					}

					Expect(generatedTokens).To(HaveLen(numGoroutines))

					// Verify all tokens are unique
					tokenMap := make(map[string]bool)
					for _, token := range generatedTokens {
						Expect(tokenMap[token]).To(BeFalse(),
							"Concurrent generation should produce unique tokens")
						tokenMap[token] = true
					}
				})
			})
		})

		When("testing collision resistance", func() {
			Context("with statistical properties", func() {
				It("should have sufficient entropy to avoid collisions", func() {
					// 24 bytes = 192 bits of entropy = 2^192 possible tokens
					// Birthday paradox: 50% collision probability at sqrt(2^192) ≈ 2^96
					// Generating 1000 tokens has negligible collision probability
					const numTokens = 1000
					tokenSet := make(map[string]bool)

					for range numTokens {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())
						tokenSet[token] = true
					}

					// All tokens should be unique (no collisions)
					Expect(tokenSet).To(HaveLen(numTokens),
						"With 192 bits of entropy, 1000 tokens should all be unique")
				})
			})
		})

		When("verifying token properties for QR code usage", func() {
			Context("with QR code requirements", func() {
				It("should generate token suitable for QR code encoding", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())

					// QR codes handle alphanumeric mode efficiently
					// Our tokens use base64url which is alphanumeric + - and _
					// This is efficient for QR encoding
					Expect(len(token)).To(Equal(32),
						"32 character tokens are suitable for QR code capacity")

					// Verify no problematic characters for QR codes
					Expect(token).NotTo(ContainSubstring(" "))
					Expect(token).NotTo(ContainSubstring("\n"))
					Expect(token).NotTo(ContainSubstring("\t"))
				})

				It("should be short enough for high-density QR codes", func() {
					token, err := crypto.GenerateToken()

					Expect(err).NotTo(HaveOccurred())

					// 32 characters easily fit in QR code version 1 (21x21)
					// at alphanumeric mode with error correction
					Expect(len(token)).To(BeNumerically("<=", 50),
						"Token should be compact for QR code encoding")
				})
			})
		})
	})

	Describe("GenerateHMACSignedToken", func() {
		When("generating an HMAC signed token", func() {
			Context("with valid secret", func() {
				It("should generate a non-empty signed token without error", func() {
					token, err := crypto.GenerateHMACSignedToken("my-secret")

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(BeEmpty())
				})

				It("should generate a token that contains the delimiter", func() {
					token, err := crypto.GenerateHMACSignedToken("my-secret")

					Expect(err).NotTo(HaveOccurred())
					Expect(token).To(ContainSubstring("."))
				})

				It("should generate a token with exactly two parts when split by the delimiter", func() {
					token, err := crypto.GenerateHMACSignedToken("my-secret")

					Expect(err).NotTo(HaveOccurred())

					parts := strings.SplitN(token, ".", 2)
					Expect(parts).To(HaveLen(2))
					Expect(parts[0]).NotTo(BeEmpty())
					Expect(parts[1]).NotTo(BeEmpty())
				})

				It("should generate unique tokens on successive calls with the same secret", func() {
					token1, err := crypto.GenerateHMACSignedToken("my-secret")
					Expect(err).NotTo(HaveOccurred())

					token2, err := crypto.GenerateHMACSignedToken("my-secret")
					Expect(err).NotTo(HaveOccurred())

					Expect(token1).NotTo(Equal(token2))
				})

				It("should generate a token that passes VerifyHMACToken round-trip", func() {
					secret := "round-trip-secret"
					token, err := crypto.GenerateHMACSignedToken(secret)

					Expect(err).NotTo(HaveOccurred())
					Expect(crypto.VerifyHMACToken(secret, token)).To(BeTrue())
				})
			})

			Context("with empty secret", func() {
				It("should return ErrInvalidHMACToken", func() {
					token, err := crypto.GenerateHMACSignedToken("")

					Expect(err).To(HaveOccurred())
					Expect(token).To(BeEmpty())
					Expect(errors.Is(err, crypto.ErrInvalidHMACToken)).To(BeTrue())
				})
			})
		})
	})

	Describe("VerifyHMACToken", func() {
		const validSecret = "verify-test-secret"

		When("verifying an HMAC token", func() {
			Context("with a valid token and matching secret", func() {
				It("should return true", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					Expect(crypto.VerifyHMACToken(validSecret, token)).To(BeTrue())
				})
			})

			Context("with a different secret", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					Expect(crypto.VerifyHMACToken("wrong-secret", token)).To(BeFalse())
				})
			})

			Context("with a tampered rawToken part", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					delimIdx := strings.Index(token, ".")
					originalSig := token[delimIdx:]
					tamperedToken := "tampered" + originalSig

					Expect(crypto.VerifyHMACToken(validSecret, tamperedToken)).To(BeFalse())
				})
			})

			Context("with a tampered signature part", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					originalRaw, _, _ := strings.Cut(token, ".")
					tamperedToken := originalRaw + ".tampered"

					Expect(crypto.VerifyHMACToken(validSecret, tamperedToken)).To(BeFalse())
				})
			})

			Context("with a token that has no delimiter", func() {
				It("should return false", func() {
					tokenWithoutDelimiter := base64.RawURLEncoding.EncodeToString([]byte("nodottoken"))

					Expect(crypto.VerifyHMACToken(validSecret, tokenWithoutDelimiter)).To(BeFalse())
				})
			})

			Context("with an empty token", func() {
				It("should return false", func() {
					Expect(crypto.VerifyHMACToken(validSecret, "")).To(BeFalse())
				})
			})

			Context("with an empty secret", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					Expect(crypto.VerifyHMACToken("", token)).To(BeFalse())
				})
			})

			Context("with an empty rawToken part (format: .{signature})", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					delimIdx := strings.Index(token, ".")
					sigPart := token[delimIdx:]
					tokenWithEmptyRaw := sigPart // starts with "."

					Expect(crypto.VerifyHMACToken(validSecret, tokenWithEmptyRaw)).To(BeFalse())
				})
			})

			Context("with an empty signature part (format: {raw}.)", func() {
				It("should return false", func() {
					token, err := crypto.GenerateHMACSignedToken(validSecret)
					Expect(err).NotTo(HaveOccurred())

					rawPart, _, _ := strings.Cut(token, ".")
					tokenWithEmptySig := rawPart + "."

					Expect(crypto.VerifyHMACToken(validSecret, tokenWithEmptySig)).To(BeFalse())
				})
			})
		})
	})

	Describe("Token Security Properties", func() {
		When("analyzing security characteristics", func() {
			Context("with entropy calculation", func() {
				It("should have 192 bits of entropy", func() {
					// 24 bytes = 192 bits
					// This provides sufficient security for participant tokens
					// 2^192 possible tokens is astronomically large

					token, err := crypto.GenerateToken()
					Expect(err).NotTo(HaveOccurred())

					decoded, err := base64.RawURLEncoding.DecodeString(token)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(HaveLen(24), "24 bytes = 192 bits of entropy")
				})
			})

			Context("with unpredictability", func() {
				It("should not have predictable patterns", func() {
					tokens := make([]string, 10)

					for i := range 10 {
						token, err := crypto.GenerateToken()
						Expect(err).NotTo(HaveOccurred())
						tokens[i] = token
					}

					// Check no two tokens share common substrings indicating patterns
					for i := 0; i < len(tokens)-1; i++ {
						for j := i + 1; j < len(tokens); j++ {
							similarity := calculateTokenSimilarity(tokens[i], tokens[j])
							Expect(similarity).To(BeNumerically("<", 0.5),
								"Tokens should not have significant similarity")
						}
					}
				})
			})
		})
	})
})

// Helper function to calculate similarity between two tokens (0.0 to 1.0)
func calculateTokenSimilarity(token1, token2 string) float64 {
	if len(token1) != len(token2) {
		return 0
	}

	matches := 0
	for i := 0; i < len(token1); i++ {
		if token1[i] == token2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(token1))
}
