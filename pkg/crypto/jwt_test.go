package crypto_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWT Token Management", func() {
	var (
		testUserID  string
		testRole    string
		testSecret  string
		validExpiry time.Duration
	)

	BeforeEach(func() {
		testUserID = uuid.New().String()
		testRole = "organizer"
		testSecret = "test-secret-key-minimum-32-chars-long"
		validExpiry = 15 * time.Minute
	})

	Describe("GenerateAccessToken", func() {
		When("generating an access token", func() {
			Context("with valid parameters", func() {
				It("should generate a valid token with correct claims", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(BeEmpty())
					Expect(strings.Count(token, ".")).To(Equal(2), "JWT should have 3 parts separated by dots")

					// Verify token can be parsed
					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.UserID.String()).To(Equal(testUserID))
					Expect(claims.Role).To(Equal(testRole))
					Expect(claims.TokenType).To(Equal(crypto.TokenTypeAccess))
					Expect(claims.Issuer).To(Equal("ezqrin-server"))
				})

				It("should set correct expiry duration", func() {
					customExpiry := 30 * time.Minute
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, customExpiry)

					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					// Verify expiry is approximately correct (within 1 second tolerance)
					expectedExpiry := time.Now().Add(customExpiry)
					actualExpiry := claims.ExpiresAt.Time
					Expect(actualExpiry).To(BeTemporally("~", expectedExpiry, 1*time.Second))
				})

				It("should set issued at and not before times to now", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)

					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					now := time.Now()
					Expect(claims.IssuedAt.Time).To(BeTemporally("~", now, 1*time.Second))
					Expect(claims.NotBefore.Time).To(BeTemporally("~", now, 1*time.Second))
				})
			})

			Context("with empty secret", func() {
				It("should return ErrEmptySecret", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, "", validExpiry)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptySecret)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})
			})

			Context("with empty user ID", func() {
				It("should return ErrEmptyUserID", func() {
					token, err := crypto.GenerateAccessToken("", testRole, testSecret, validExpiry)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptyUserID)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})
			})

			Context("with invalid user ID format", func() {
				It("should return error for non-UUID string", func() {
					token, err := crypto.GenerateAccessToken("not-a-uuid", testRole, testSecret, validExpiry)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("invalid user id format"))
					Expect(token).To(BeEmpty())
				})
			})

			Context("with zero expiry", func() {
				It("should return ErrInvalidExpiry", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, 0)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidExpiry)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})
			})

			Context("with negative expiry", func() {
				It("should return ErrInvalidExpiry", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, -15*time.Minute)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidExpiry)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})
			})

			Context("with empty role", func() {
				It("should generate token with empty role", func() {
					token, err := crypto.GenerateAccessToken(testUserID, "", testSecret, validExpiry)

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(BeEmpty())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.Role).To(BeEmpty())
				})
			})
		})
	})

	Describe("GenerateRefreshToken", func() {
		When("generating a refresh token", func() {
			Context("with valid parameters", func() {
				It("should generate a valid token with correct claims", func() {
					refreshExpiry := 168 * time.Hour // 7 days
					token, err := crypto.GenerateRefreshToken(testUserID, testRole, testSecret, refreshExpiry)

					Expect(err).NotTo(HaveOccurred())
					Expect(token).NotTo(BeEmpty())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.UserID.String()).To(Equal(testUserID))
					Expect(claims.Role).To(Equal(testRole))
					Expect(claims.TokenType).To(Equal(crypto.TokenTypeRefresh))
					Expect(claims.Issuer).To(Equal("ezqrin-server"))
				})

				It("should generate token with long expiry for web platform", func() {
					webExpiry := 168 * time.Hour // 7 days
					token, err := crypto.GenerateRefreshToken(testUserID, testRole, testSecret, webExpiry)

					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					expectedExpiry := time.Now().Add(webExpiry)
					Expect(claims.ExpiresAt.Time).To(BeTemporally("~", expectedExpiry, 1*time.Second))
				})

				It("should generate token with longer expiry for mobile platform", func() {
					mobileExpiry := 2160 * time.Hour // 90 days
					token, err := crypto.GenerateRefreshToken(testUserID, testRole, testSecret, mobileExpiry)

					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					expectedExpiry := time.Now().Add(mobileExpiry)
					Expect(claims.ExpiresAt.Time).To(BeTemporally("~", expectedExpiry, 1*time.Second))
				})
			})

			Context("with invalid parameters", func() {
				It("should return ErrEmptySecret for empty secret", func() {
					token, err := crypto.GenerateRefreshToken(testUserID, testRole, "", 168*time.Hour)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptySecret)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})

				It("should return ErrEmptyUserID for empty user ID", func() {
					token, err := crypto.GenerateRefreshToken("", testRole, testSecret, 168*time.Hour)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptyUserID)).To(BeTrue())
					Expect(token).To(BeEmpty())
				})
			})
		})
	})

	Describe("Token Type Differentiation", func() {
		When("comparing access and refresh tokens", func() {
			Context("with both token types generated", func() {
				It("should correctly differentiate token types", func() {
					accessToken, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, 15*time.Minute)
					Expect(err).NotTo(HaveOccurred())

					refreshToken, err := crypto.GenerateRefreshToken(testUserID, testRole, testSecret, 168*time.Hour)
					Expect(err).NotTo(HaveOccurred())

					accessClaims, err := crypto.ParseToken(accessToken, testSecret)
					Expect(err).NotTo(HaveOccurred())

					refreshClaims, err := crypto.ParseToken(refreshToken, testSecret)
					Expect(err).NotTo(HaveOccurred())

					Expect(accessClaims.TokenType).To(Equal(crypto.TokenTypeAccess))
					Expect(refreshClaims.TokenType).To(Equal(crypto.TokenTypeRefresh))
					Expect(accessClaims.TokenType).NotTo(Equal(refreshClaims.TokenType))
				})
			})
		})
	})

	Describe("ValidateToken", func() {
		When("validating a token", func() {
			Context("with valid token", func() {
				It("should validate successfully", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ValidateToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with expired token", func() {
				It("should return ErrExpiredToken", func() {
					// Generate token that expires immediately
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, 1*time.Nanosecond)
					Expect(err).NotTo(HaveOccurred())

					// Wait for token to expire
					time.Sleep(10 * time.Millisecond)

					err = crypto.ValidateToken(token, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrExpiredToken)).To(BeTrue())
				})
			})

			Context("with invalid signature", func() {
				It("should return ErrInvalidToken", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					wrongSecret := "wrong-secret-key-different-from-original"
					err = crypto.ValidateToken(token, wrongSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})
			})

			Context("with malformed token", func() {
				It("should return ErrInvalidToken for invalid JWT format", func() {
					malformedToken := "not.a.valid.jwt.token"

					err := crypto.ValidateToken(malformedToken, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})

				It("should return ErrInvalidToken for completely invalid string", func() {
					malformedToken := "completely-invalid"

					err := crypto.ValidateToken(malformedToken, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})

				It("should return ErrInvalidToken for empty token string", func() {
					err := crypto.ValidateToken("", testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})
			})

			Context("with empty secret", func() {
				It("should return ErrEmptySecret", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ValidateToken(token, "")
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptySecret)).To(BeTrue())
				})
			})

			Context("with token using wrong signing method", func() {
				It("should return ErrInvalidToken", func() {
					// Create a token with RS256 instead of HS256
					claims := &crypto.Claims{
						UserID:    uuid.MustParse(testUserID),
						Role:      testRole,
						TokenType: crypto.TokenTypeAccess,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(validExpiry)),
							IssuedAt:  jwt.NewNumericDate(time.Now()),
							Issuer:    "ezqrin-server",
						},
					}

					// Create token with unsupported signing method (none)
					token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
					tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
					Expect(err).NotTo(HaveOccurred())

					err = crypto.ValidateToken(tokenString, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})
			})
		})
	})

	Describe("ParseToken", func() {
		When("parsing a token", func() {
			Context("with valid token", func() {
				It("should parse and return correct claims", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims).NotTo(BeNil())
					Expect(claims.UserID.String()).To(Equal(testUserID))
					Expect(claims.Role).To(Equal(testRole))
					Expect(claims.TokenType).To(Equal(crypto.TokenTypeAccess))
					Expect(claims.Issuer).To(Equal("ezqrin-server"))
					Expect(claims.ExpiresAt).NotTo(BeNil())
					Expect(claims.IssuedAt).NotTo(BeNil())
					Expect(claims.NotBefore).NotTo(BeNil())
				})

				It("should parse token with all claim fields correctly", func() {
					token, err := crypto.GenerateRefreshToken(testUserID, "attendee", testSecret, 168*time.Hour)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.UserID).NotTo(Equal(uuid.Nil))
					Expect(claims.Role).To(Equal("attendee"))
					Expect(claims.TokenType).To(Equal(crypto.TokenTypeRefresh))
				})
			})

			Context("with expired token", func() {
				It("should return ErrExpiredToken", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, 1*time.Nanosecond)
					Expect(err).NotTo(HaveOccurred())

					time.Sleep(10 * time.Millisecond)

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrExpiredToken)).To(BeTrue())
					Expect(claims).To(BeNil())
				})
			})

			Context("with invalid signature", func() {
				It("should return ErrInvalidToken", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, "wrong-secret")
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
					Expect(claims).To(BeNil())
				})
			})

			Context("with malformed token", func() {
				It("should return ErrInvalidToken for invalid format", func() {
					claims, err := crypto.ParseToken("not.valid.token", testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
					Expect(claims).To(BeNil())
				})

				It("should return ErrInvalidToken for empty token", func() {
					claims, err := crypto.ParseToken("", testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
					Expect(claims).To(BeNil())
				})
			})

			Context("with empty secret", func() {
				It("should return ErrEmptySecret", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, "")
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrEmptySecret)).To(BeTrue())
					Expect(claims).To(BeNil())
				})
			})

			Context("with token containing nil user ID", func() {
				It("should return ErrInvalidClaims", func() {
					// Manually create a token with nil UUID
					claims := &crypto.Claims{
						UserID:    uuid.Nil,
						Role:      testRole,
						TokenType: crypto.TokenTypeAccess,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(validExpiry)),
							IssuedAt:  jwt.NewNumericDate(time.Now()),
							Issuer:    "ezqrin-server",
						},
					}

					token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
					tokenString, err := token.SignedString([]byte(testSecret))
					Expect(err).NotTo(HaveOccurred())

					parsedClaims, err := crypto.ParseToken(tokenString, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidClaims)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("user_id is missing or invalid"))
					Expect(parsedClaims).To(BeNil())
				})
			})

			Context("with token using wrong signing method", func() {
				It("should return ErrInvalidToken", func() {
					claims := &crypto.Claims{
						UserID:    uuid.MustParse(testUserID),
						Role:      testRole,
						TokenType: crypto.TokenTypeAccess,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(validExpiry)),
							IssuedAt:  jwt.NewNumericDate(time.Now()),
							Issuer:    "ezqrin-server",
						},
					}

					token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
					tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
					Expect(err).NotTo(HaveOccurred())

					parsedClaims, err := crypto.ParseToken(tokenString, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
					Expect(parsedClaims).To(BeNil())
				})
			})
		})
	})

	Describe("Claims Extraction", func() {
		When("extracting claims from a parsed token", func() {
			Context("with complete claims", func() {
				It("should extract all standard and custom claims", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					// Custom claims
					Expect(claims.UserID).NotTo(Equal(uuid.Nil))
					Expect(claims.Role).To(Equal(testRole))
					Expect(claims.TokenType).To(Equal(crypto.TokenTypeAccess))

					// Standard claims
					Expect(claims.Issuer).To(Equal("ezqrin-server"))
					Expect(claims.ExpiresAt).NotTo(BeNil())
					Expect(claims.IssuedAt).NotTo(BeNil())
					Expect(claims.NotBefore).NotTo(BeNil())

					// Verify times
					now := time.Now()
					Expect(claims.IssuedAt.Time).To(BeTemporally("~", now, 1*time.Second))
					Expect(claims.ExpiresAt.Time).To(BeTemporally(">", now))
				})
			})

			Context("with different roles", func() {
				It("should correctly extract organizer role", func() {
					token, err := crypto.GenerateAccessToken(testUserID, "organizer", testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.Role).To(Equal("organizer"))
				})

				It("should correctly extract attendee role", func() {
					token, err := crypto.GenerateAccessToken(testUserID, "attendee", testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.Role).To(Equal("attendee"))
				})
			})
		})
	})

	Describe("Edge Cases", func() {
		When("handling edge cases", func() {
			Context("with very short expiry (1 second)", func() {
				It("should generate valid token that expires quickly", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, 1*time.Second)
					Expect(err).NotTo(HaveOccurred())

					// Token should be valid immediately
					err = crypto.ValidateToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					// Wait and verify it expires
					time.Sleep(1100 * time.Millisecond)
					err = crypto.ValidateToken(token, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrExpiredToken)).To(BeTrue())
				})
			})

			Context("with very long expiry", func() {
				It("should generate valid token with long lifetime", func() {
					longExpiry := 8760 * time.Hour // 365 days
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, longExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())

					expectedExpiry := time.Now().Add(longExpiry)
					Expect(claims.ExpiresAt.Time).To(BeTemporally("~", expectedExpiry, 1*time.Second))
				})
			})

			Context("with special characters in role", func() {
				It("should handle special characters in role field", func() {
					specialRole := "admin-super_user@org"
					token, err := crypto.GenerateAccessToken(testUserID, specialRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.Role).To(Equal(specialRole))
				})
			})

			Context("with minimum valid UUID", func() {
				It("should handle UUID with all zeros except version bits", func() {
					minUUID := "00000000-0000-4000-8000-000000000000"
					token, err := crypto.GenerateAccessToken(minUUID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					claims, err := crypto.ParseToken(token, testSecret)
					Expect(err).NotTo(HaveOccurred())
					Expect(claims.UserID.String()).To(Equal(minUUID))
				})
			})

			Context("with different users generating different tokens", func() {
				It("should generate different tokens for different users", func() {
					userID1 := uuid.New().String()
					userID2 := uuid.New().String()

					token1, err := crypto.GenerateAccessToken(userID1, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					token2, err := crypto.GenerateAccessToken(userID2, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					Expect(token1).NotTo(Equal(token2))
				})
			})

			Context("with token validated multiple times", func() {
				It("should remain valid across multiple validations", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					// Validate multiple times
					for i := 0; i < 5; i++ {
						err = crypto.ValidateToken(token, testSecret)
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})
		})
	})

	Describe("Error Messages", func() {
		When("encountering errors", func() {
			Context("with various error conditions", func() {
				It("should provide descriptive error for invalid UUID format", func() {
					_, err := crypto.GenerateAccessToken("invalid-uuid-format", testRole, testSecret, validExpiry)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("invalid user id format"))
				})

				It("should wrap errors with context for invalid token", func() {
					_, err := crypto.ParseToken("malformed.token", testSecret)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, crypto.ErrInvalidToken)).To(BeTrue())
				})

				It("should provide clear error for nil UUID in claims", func() {
					claims := &crypto.Claims{
						UserID:    uuid.Nil,
						Role:      testRole,
						TokenType: crypto.TokenTypeAccess,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(validExpiry)),
							IssuedAt:  jwt.NewNumericDate(time.Now()),
							Issuer:    "ezqrin-server",
						},
					}

					token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
					tokenString, err := token.SignedString([]byte(testSecret))
					Expect(err).NotTo(HaveOccurred())

					_, err = crypto.ParseToken(tokenString, testSecret)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("user_id is missing or invalid"))
				})
			})
		})
	})

	Describe("Concurrent Token Operations", func() {
		When("performing concurrent operations", func() {
			Context("with multiple goroutines generating tokens", func() {
				It("should safely generate tokens concurrently", func() {
					const numGoroutines = 10
					results := make(chan string, numGoroutines)
					errors := make(chan error, numGoroutines)

					for i := 0; i < numGoroutines; i++ {
						go func() {
							userID := uuid.New().String()
							token, err := crypto.GenerateAccessToken(userID, testRole, testSecret, validExpiry)
							if err != nil {
								errors <- err
							} else {
								results <- token
							}
						}()
					}

					// Collect results
					tokens := make([]string, 0, numGoroutines)
					for i := 0; i < numGoroutines; i++ {
						select {
						case token := <-results:
							tokens = append(tokens, token)
						case err := <-errors:
							Fail(fmt.Sprintf("Unexpected error: %v", err))
						}
					}

					Expect(tokens).To(HaveLen(numGoroutines))

					// Verify all tokens are unique
					tokenMap := make(map[string]bool)
					for _, token := range tokens {
						Expect(tokenMap[token]).To(BeFalse(), "Token should be unique")
						tokenMap[token] = true
					}
				})
			})

			Context("with multiple goroutines validating tokens", func() {
				It("should safely validate tokens concurrently", func() {
					token, err := crypto.GenerateAccessToken(testUserID, testRole, testSecret, validExpiry)
					Expect(err).NotTo(HaveOccurred())

					const numGoroutines = 10
					errors := make(chan error, numGoroutines)

					for i := 0; i < numGoroutines; i++ {
						go func() {
							err := crypto.ValidateToken(token, testSecret)
							errors <- err
						}()
					}

					// Verify all validations succeeded
					for i := 0; i < numGoroutines; i++ {
						err := <-errors
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})
		})
	})
})
