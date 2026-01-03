package crypto_test

import (
	"errors"
	"strings"

	"github.com/fumkob/ezqrin-server/pkg/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Password Hashing with Bcrypt", func() {
	var testPassword string

	BeforeEach(func() {
		testPassword = "SecureP@ssw0rd123"
	})

	Describe("HashPassword", func() {
		When("hashing a password", func() {
			Context("with valid password", func() {
				It("should generate a bcrypt hash successfully", func() {
					hash, err := crypto.HashPassword(testPassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
					Expect(hash).NotTo(Equal(testPassword), "Hash should not be the same as plaintext password")
				})

				It("should generate hash with correct bcrypt format", func() {
					hash, err := crypto.HashPassword(testPassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).To(HavePrefix("$2a$"), "Bcrypt hash should start with $2a$ prefix")
					Expect(
						strings.Count(hash, "$"),
					).To(BeNumerically(">=", 3), "Bcrypt hash should have at least 3 $ separators")
				})

				It("should use cost factor 12", func() {
					hash, err := crypto.HashPassword(testPassword)

					Expect(err).NotTo(HaveOccurred())

					// Extract cost from hash (format: $2a$12$...)
					parts := strings.Split(hash, "$")
					Expect(parts).To(HaveLen(4), "Bcrypt hash should have 4 parts")
					Expect(parts[2]).To(Equal("12"), "Cost factor should be 12")
				})

				It("should generate different hashes for same password due to different salts", func() {
					hash1, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					hash2, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					Expect(hash1).NotTo(Equal(hash2), "Different salts should produce different hashes")
				})

				It("should generate hash that can be verified", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					// Verify using golang.org/x/crypto/bcrypt directly
					err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(testPassword))
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with empty password", func() {
				It("should return ErrEmptyPassword", func() {
					hash, err := crypto.HashPassword("")

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptyPassword)).To(BeTrue())
					Expect(hash).To(BeEmpty())
				})
			})

			Context("with password at maximum length (72 bytes)", func() {
				It("should hash successfully", func() {
					// Create password exactly 72 bytes
					maxPassword := strings.Repeat("a", 72)

					hash, err := crypto.HashPassword(maxPassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})
			})

			Context("with password exceeding maximum length (>72 bytes)", func() {
				It("should return ErrPasswordTooLong", func() {
					// Create password with 73 bytes
					longPassword := strings.Repeat("a", 73)

					hash, err := crypto.HashPassword(longPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrPasswordTooLong)).To(BeTrue())
					Expect(hash).To(BeEmpty())
				})

				It("should return ErrPasswordTooLong for very long password", func() {
					longPassword := strings.Repeat("a", 200)

					hash, err := crypto.HashPassword(longPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrPasswordTooLong)).To(BeTrue())
					Expect(hash).To(BeEmpty())
				})
			})

			Context("with special characters in password", func() {
				It("should hash password with special characters", func() {
					specialPassword := "P@ssw0rd!#$%^&*()_+-=[]{}|;:',.<>?/~`"

					hash, err := crypto.HashPassword(specialPassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})
			})

			Context("with unicode characters in password", func() {
				It("should hash password with unicode characters", func() {
					unicodePassword := "パスワード密码🔐"

					hash, err := crypto.HashPassword(unicodePassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})

				It("should enforce byte length not character count", func() {
					// Unicode characters can be multiple bytes
					// Create a password that's short in characters but exceeds 72 bytes
					// Each emoji is ~4 bytes, so 19 emojis = 76 bytes
					longUnicodePassword := strings.Repeat("🔐", 19)

					hash, err := crypto.HashPassword(longUnicodePassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrPasswordTooLong)).To(BeTrue())
					Expect(hash).To(BeEmpty())
				})
			})

			Context("with whitespace in password", func() {
				It("should hash password with leading whitespace", func() {
					password := "  password"

					hash, err := crypto.HashPassword(password)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})

				It("should hash password with trailing whitespace", func() {
					password := "password  "

					hash, err := crypto.HashPassword(password)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})

				It("should hash password with spaces in middle", func() {
					password := "pass word with spaces"

					hash, err := crypto.HashPassword(password)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})
			})

			Context("with numeric password", func() {
				It("should hash numeric-only password", func() {
					numericPassword := "123456789012345"

					hash, err := crypto.HashPassword(numericPassword)

					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())
				})
			})
		})
	})

	Describe("ComparePassword", func() {
		When("comparing password with hash", func() {
			Context("with correct password", func() {
				It("should return no error for matching password", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, testPassword)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should handle password with special characters", func() {
					specialPassword := "P@ssw0rd!#$%"
					hash, err := crypto.HashPassword(specialPassword)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, specialPassword)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should handle password at maximum length", func() {
					maxPassword := strings.Repeat("a", 72)
					hash, err := crypto.HashPassword(maxPassword)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, maxPassword)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with incorrect password", func() {
				It("should return ErrHashMismatch", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					wrongPassword := "WrongPassword123"
					err = crypto.ComparePassword(hash, wrongPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrHashMismatch)).To(BeTrue())
				})

				It("should return ErrHashMismatch for slightly different password", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					// One character different
					wrongPassword := testPassword + "x"
					err = crypto.ComparePassword(hash, wrongPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrHashMismatch)).To(BeTrue())
				})

				It("should return ErrHashMismatch for case-different password", func() {
					hash, err := crypto.HashPassword("Password123")
					Expect(err).NotTo(HaveOccurred())

					wrongPassword := "password123" // lowercase
					err = crypto.ComparePassword(hash, wrongPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrHashMismatch)).To(BeTrue())
				})
			})

			Context("with empty password", func() {
				It("should return ErrEmptyPassword", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, "")

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptyPassword)).To(BeTrue())
				})
			})

			Context("with empty hash", func() {
				It("should return error for empty hash string", func() {
					err := crypto.ComparePassword("", testPassword)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("hashed password cannot be empty"))
				})
			})

			Context("with password too long", func() {
				It("should return ErrPasswordTooLong", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					longPassword := strings.Repeat("a", 73)
					err = crypto.ComparePassword(hash, longPassword)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrPasswordTooLong)).To(BeTrue())
				})
			})

			Context("with invalid hash format", func() {
				It("should return error for malformed hash", func() {
					invalidHash := "not-a-valid-bcrypt-hash"

					err := crypto.ComparePassword(invalidHash, testPassword)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to compare password"))
				})

				It("should return error for corrupted hash", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					// Corrupt the hash by truncating it
					corruptedHash := hash[:len(hash)-5]

					err = crypto.ComparePassword(corruptedHash, testPassword)

					Expect(err).To(HaveOccurred())
				})
			})

			Context("with hash from different algorithm", func() {
				It("should return error for non-bcrypt hash", func() {
					// SHA-256 hash (not bcrypt)
					nonBcryptHash := "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"

					err := crypto.ComparePassword(nonBcryptHash, testPassword)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to compare password"))
				})
			})
		})
	})

	Describe("Password Hashing Security Properties", func() {
		When("examining security properties", func() {
			Context("with salt randomness", func() {
				It("should generate unique hashes for same password", func() {
					hashes := make([]string, 10)

					for i := 0; i < 10; i++ {
						hash, err := crypto.HashPassword(testPassword)
						Expect(err).NotTo(HaveOccurred())
						hashes[i] = hash
					}

					// Verify all hashes are unique
					hashMap := make(map[string]bool)
					for _, hash := range hashes {
						Expect(hashMap[hash]).To(BeFalse(), "Each hash should be unique due to random salt")
						hashMap[hash] = true
					}
				})
			})

			Context("with consistent verification", func() {
				It("should consistently verify correct password", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					// Verify multiple times
					for i := 0; i < 5; i++ {
						err = crypto.ComparePassword(hash, testPassword)
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("should consistently reject incorrect password", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					wrongPassword := "WrongPassword"

					// Verify rejection multiple times
					for i := 0; i < 5; i++ {
						err = crypto.ComparePassword(hash, wrongPassword)
						Expect(err).To(HaveOccurred())
						Expect(errors.Is(err, crypto.ErrHashMismatch)).To(BeTrue())
					}
				})
			})

			Context("with password similarity", func() {
				It("should generate completely different hashes for similar passwords", func() {
					password1 := "Password123"
					password2 := "Password124" // Only last character different

					hash1, err := crypto.HashPassword(password1)
					Expect(err).NotTo(HaveOccurred())

					hash2, err := crypto.HashPassword(password2)
					Expect(err).NotTo(HaveOccurred())

					// Hashes should be completely different
					Expect(hash1).NotTo(Equal(hash2))

					// Calculate similarity (should be very low)
					similarity := calculateSimilarity(hash1, hash2)
					Expect(similarity).To(BeNumerically("<", 0.3), "Hashes should have low similarity")
				})
			})
		})
	})

	Describe("Edge Cases", func() {
		When("handling edge cases", func() {
			Context("with boundary length passwords", func() {
				It("should handle password with exactly 1 byte", func() {
					password := "a"

					hash, err := crypto.HashPassword(password)
					Expect(err).NotTo(HaveOccurred())
					Expect(hash).NotTo(BeEmpty())

					err = crypto.ComparePassword(hash, password)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should handle password with 71 bytes (one below limit)", func() {
					password := strings.Repeat("a", 71)

					hash, err := crypto.HashPassword(password)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, password)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should handle password with 72 bytes (exactly at limit)", func() {
					password := strings.Repeat("a", 72)

					hash, err := crypto.HashPassword(password)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, password)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should reject password with 73 bytes (one over limit)", func() {
					password := strings.Repeat("a", 73)

					hash, err := crypto.HashPassword(password)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrPasswordTooLong)).To(BeTrue())
					Expect(hash).To(BeEmpty())
				})
			})

			Context("with null bytes in password", func() {
				It("should handle password with null byte", func() {
					password := "pass\x00word"

					hash, err := crypto.HashPassword(password)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, password)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with newline characters", func() {
				It("should handle password with newline", func() {
					password := "pass\nword"

					hash, err := crypto.HashPassword(password)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ComparePassword(hash, password)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Describe("Concurrent Operations", func() {
		When("performing concurrent hashing", func() {
			Context("with multiple goroutines hashing passwords", func() {
				It("should safely hash passwords concurrently", func() {
					const numGoroutines = 10
					results := make(chan string, numGoroutines)
					errors := make(chan error, numGoroutines)

					for i := 0; i < numGoroutines; i++ {
						go func(index int) {
							password := testPassword + string(rune(index))
							hash, err := crypto.HashPassword(password)
							if err != nil {
								errors <- err
							} else {
								results <- hash
							}
						}(i)
					}

					// Collect results
					hashes := make([]string, 0, numGoroutines)
					for i := 0; i < numGoroutines; i++ {
						select {
						case hash := <-results:
							hashes = append(hashes, hash)
						case err := <-errors:
							Fail("Unexpected error: " + err.Error())
						}
					}

					Expect(hashes).To(HaveLen(numGoroutines))

					// Verify all hashes are unique
					hashMap := make(map[string]bool)
					for _, hash := range hashes {
						Expect(hashMap[hash]).To(BeFalse())
						hashMap[hash] = true
					}
				})
			})

			Context("with multiple goroutines comparing passwords", func() {
				It("should safely compare passwords concurrently", func() {
					hash, err := crypto.HashPassword(testPassword)
					Expect(err).NotTo(HaveOccurred())

					const numGoroutines = 10
					results := make(chan error, numGoroutines)

					for i := 0; i < numGoroutines; i++ {
						go func() {
							err := crypto.ComparePassword(hash, testPassword)
							results <- err
						}()
					}

					// Verify all comparisons succeeded
					for i := 0; i < numGoroutines; i++ {
						err := <-results
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})
		})
	})

	Describe("Integration with Real Use Cases", func() {
		When("simulating real-world scenarios", func() {
			Context("with user registration flow", func() {
				It("should hash and verify password in registration scenario", func() {
					// Simulate user registration
					userPassword := "MySecurePassword123!"

					// Hash password for storage
					hashedPassword, err := crypto.HashPassword(userPassword)
					Expect(err).NotTo(HaveOccurred())

					// Simulate user login with correct password
					err = crypto.ComparePassword(hashedPassword, userPassword)
					Expect(err).NotTo(HaveOccurred())

					// Simulate user login with incorrect password
					err = crypto.ComparePassword(hashedPassword, "WrongPassword")
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrHashMismatch)).To(BeTrue())
				})
			})

			Context("with password change flow", func() {
				It("should handle password change correctly", func() {
					oldPassword := "OldPassword123"
					newPassword := "NewPassword456"

					// Hash old password
					oldHash, err := crypto.HashPassword(oldPassword)
					Expect(err).NotTo(HaveOccurred())

					// Verify old password
					err = crypto.ComparePassword(oldHash, oldPassword)
					Expect(err).NotTo(HaveOccurred())

					// Hash new password
					newHash, err := crypto.HashPassword(newPassword)
					Expect(err).NotTo(HaveOccurred())

					// Old password should not match new hash
					err = crypto.ComparePassword(newHash, oldPassword)
					Expect(err).To(HaveOccurred())

					// New password should match new hash
					err = crypto.ComparePassword(newHash, newPassword)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})

// Helper function to calculate similarity between two strings
func calculateSimilarity(s1, s2 string) float64 {
	if len(s1) != len(s2) {
		return 0
	}

	matches := 0
	for i := 0; i < len(s1); i++ {
		if s1[i] == s2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(s1))
}
