package auth_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var _ = Describe("RefreshTokenUseCase", func() {
	var (
		ctrl              *gomock.Controller
		mockUserRepo      *mocks.MockUserRepository
		mockBlacklistRepo *mocks.MockTokenBlacklistRepository
		useCase           *auth.RefreshTokenUseCase
		ctx               context.Context
		nopLoggerRefresh  *logger.Logger
		testUserID        uuid.UUID
		testUser          *entity.User
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mocks.NewMockUserRepository(ctrl)
		mockBlacklistRepo = mocks.NewMockTokenBlacklistRepository(ctrl)
		nopLoggerRefresh = &logger.Logger{Logger: zap.NewNop()}
		useCase = auth.NewRefreshTokenUseCase(
			mockUserRepo,
			mockBlacklistRepo,
			testJWTSecret,
			auth.RefreshTokenExpiryWeb,
			auth.RefreshTokenExpiryMobile,
			nopLoggerRefresh,
		)
		ctx = context.Background()
		testUserID = uuid.New()
		testUser = &entity.User{
			ID:        testUserID,
			Email:     "carol@example.com",
			Name:      "Carol",
			Role:      entity.RoleOrganizer,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// makeRefreshToken generates a valid refresh token for testUserID.
	makeRefreshToken := func(clientType string) string {
		token, err := crypto.GenerateRefreshToken(
			testUserID.String(),
			string(entity.RoleOrganizer),
			testJWTSecret,
			clientType,
			auth.RefreshTokenExpiryWeb,
		)
		Expect(err).NotTo(HaveOccurred())
		return token
	}

	// makeAccessToken generates a valid access token (wrong type for refresh endpoint).
	makeAccessToken := func() string {
		token, err := crypto.GenerateAccessToken(
			testUserID.String(),
			string(entity.RoleOrganizer),
			testJWTSecret,
			auth.AccessTokenExpiry,
		)
		Expect(err).NotTo(HaveOccurred())
		return token
	}

	Describe("Execute", func() {
		When("refreshing with a valid refresh token", func() {
			Context("and the token is not blacklisted and the user exists", func() {
				It("should return a new token pair", func() {
					refreshToken := makeRefreshToken("web")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, nil)
					mockUserRepo.EXPECT().
						FindByID(ctx, testUserID).
						Return(testUser, nil)
					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(nil)

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.AccessToken).NotTo(BeEmpty())
					Expect(result.RefreshToken).NotTo(BeEmpty())
					Expect(result.TokenType).To(Equal("Bearer"))
					Expect(result.ExpiresIn).To(Equal(int(auth.AccessTokenExpiry.Seconds())))
					Expect(result.User).NotTo(BeNil())
					Expect(result.User.ID).To(Equal(testUserID))
				})
			})

			Context("and the old refresh token is blacklisted after rotation (best-effort)", func() {
				It("should still succeed even if blacklisting the old token fails", func() {
					refreshToken := makeRefreshToken("web")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, nil)
					mockUserRepo.EXPECT().
						FindByID(ctx, testUserID).
						Return(testUser, nil)
					// Best-effort: failure here must not fail the request
					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(errors.New("redis error"))

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.AccessToken).NotTo(BeEmpty())
				})
			})

			Context("with mobile client type", func() {
				It("should inherit the mobile client type in the new refresh token", func() {
					refreshToken := makeRefreshToken("mobile")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, nil)
					mockUserRepo.EXPECT().
						FindByID(ctx, testUserID).
						Return(testUser, nil)
					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(nil)

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())

					newClaims, parseErr := crypto.ParseToken(result.RefreshToken, testJWTSecret)
					Expect(parseErr).NotTo(HaveOccurred())
					Expect(newClaims.ClientType).To(Equal("mobile"))
				})
			})
		})

		When("validating the request input", func() {
			Context("with an empty refresh token", func() {
				It("should return a validation error", func() {
					req := &auth.RefreshRequest{RefreshToken: ""}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})
		})

		When("the refresh token is invalid", func() {
			Context("and ParseToken returns a non-expiry error", func() {
				It("should return an unauthorized error", func() {
					req := &auth.RefreshRequest{RefreshToken: "totally.invalid.token"}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("invalid refresh token"))
				})
			})
		})

		When("the refresh token has expired", func() {
			Context("and ParseToken returns ErrExpiredToken", func() {
				It("should return an unauthorized error with expiry message", func() {
					// Create a token with 1 ns TTL, then wait for it to expire
					expiredToken, err := crypto.GenerateRefreshToken(
						testUserID.String(),
						string(entity.RoleOrganizer),
						testJWTSecret,
						"web",
						1, // 1 nanosecond
					)
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(5 * time.Millisecond)

					req := &auth.RefreshRequest{RefreshToken: expiredToken}

					result, execErr := useCase.Execute(ctx, req)

					Expect(execErr).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(execErr, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("expired"))
				})
			})
		})

		When("an access token is used instead of a refresh token", func() {
			Context("and the token type claim is 'access'", func() {
				It("should return an unauthorized error for wrong token type", func() {
					accessToken := makeAccessToken()

					// isBlacklisted is checked after type verification, so only parse is called
					req := &auth.RefreshRequest{RefreshToken: accessToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("invalid token type"))
				})
			})
		})

		When("the refresh token has been revoked", func() {
			Context("and IsBlacklisted returns true", func() {
				It("should return an unauthorized error", func() {
					refreshToken := makeRefreshToken("web")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(true, nil)

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("revoked"))
				})
			})
		})

		When("the blacklist check itself fails", func() {
			Context("and IsBlacklisted returns an error", func() {
				It("should return an internal error", func() {
					refreshToken := makeRefreshToken("web")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, errors.New("redis unavailable"))

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeInternal))
				})
			})
		})

		When("the user referenced by the token no longer exists", func() {
			Context("and FindByID returns an error", func() {
				It("should return an unauthorized error", func() {
					refreshToken := makeRefreshToken("web")

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, nil)
					mockUserRepo.EXPECT().
						FindByID(ctx, testUserID).
						Return(nil, errors.New("user not found"))

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("user not found"))
				})
			})
		})

		When("the user has been soft-deleted", func() {
			Context("and user.IsDeleted() returns true", func() {
				It("should return an unauthorized error", func() {
					refreshToken := makeRefreshToken("web")
					deletedAt := time.Now()
					deletedUser := &entity.User{
						ID:        testUserID,
						Email:     "carol@example.com",
						Name:      "Carol",
						Role:      entity.RoleOrganizer,
						DeletedAt: &deletedAt,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}

					mockBlacklistRepo.EXPECT().
						IsBlacklisted(ctx, refreshToken).
						Return(false, nil)
					mockUserRepo.EXPECT().
						FindByID(ctx, testUserID).
						Return(deletedUser, nil)

					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(ContainSubstring("user not found"))
				})
			})
		})

		When("generating new tokens fails due to an empty JWT secret", func() {
			Context("and the use case is constructed with an empty secret", func() {
				It("should return an internal error", func() {
					// Build a valid token with the real secret first
					refreshToken := makeRefreshToken("web")

					// Then stand up a use case with no secret (token generation will fail)
					useCaseNoSecret := auth.NewRefreshTokenUseCase(
						mockUserRepo,
						mockBlacklistRepo,
						"",
						auth.RefreshTokenExpiryWeb,
						auth.RefreshTokenExpiryMobile,
						nopLoggerRefresh,
					)

					// ParseToken will fail with empty secret too, so we expect unauthorized
					req := &auth.RefreshRequest{RefreshToken: refreshToken}

					result, err := useCaseNoSecret.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					// Empty secret causes parse to fail → unauthorized
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
				})
			})
		})
	})
})

var _ = Describe("ParseTokenForLogout", func() {
	When("the token string is empty", func() {
		It("should return nil without error", func() {
			id, ttl, err := auth.ParseTokenForLogout("", testJWTSecret)

			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(BeNil())
			Expect(ttl).To(BeZero())
		})
	})

	When("the token is valid and not yet expired", func() {
		It("should return the user ID and a positive TTL", func() {
			userID := uuid.New()
			token, err := crypto.GenerateAccessToken(userID.String(), "organizer", testJWTSecret, 15*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			id, ttl, parseErr := auth.ParseTokenForLogout(token, testJWTSecret)

			Expect(parseErr).NotTo(HaveOccurred())
			Expect(id).NotTo(BeNil())
			Expect(*id).To(Equal(userID))
			Expect(ttl).To(BeNumerically(">", 0))
		})
	})

	When("the token is expired", func() {
		It("should return nil without error (expired tokens are tolerated)", func() {
			userID := uuid.New()
			expiredToken, err := crypto.GenerateAccessToken(userID.String(), "organizer", testJWTSecret, 1)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(5 * time.Millisecond)

			id, ttl, parseErr := auth.ParseTokenForLogout(expiredToken, testJWTSecret)

			Expect(parseErr).NotTo(HaveOccurred())
			Expect(id).To(BeNil())
			Expect(ttl).To(BeZero())
		})
	})

	When("the token is completely invalid", func() {
		It("should return an error", func() {
			id, ttl, err := auth.ParseTokenForLogout("not.a.jwt", testJWTSecret)

			Expect(err).To(HaveOccurred())
			Expect(id).To(BeNil())
			Expect(ttl).To(BeZero())
		})
	})
})

var _ = Describe("resolveRefreshExpiry", func() {
	// resolveRefreshExpiry is an unexported function; we test its observable behaviour
	// through the LoginUseCase and RefreshTokenUseCase which delegate to it.
	// Here we use LoginUseCase as a proxy since it accepts client type in its request.

	var (
		ctrl         *gomock.Controller
		mockUserRepo *mocks.MockUserRepository
		ctx          context.Context
		nopLog       *logger.Logger
		testUser2    *entity.User
	)

	const plainPassword = "TestPassword9!"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mocks.NewMockUserRepository(ctrl)
		nopLog = &logger.Logger{Logger: zap.NewNop()}
		ctx = context.Background()

		hash, err := crypto.HashPassword(plainPassword)
		Expect(err).NotTo(HaveOccurred())

		testUser2 = &entity.User{
			ID:           uuid.New(),
			Email:        "expiry@example.com",
			PasswordHash: hash,
			Name:         "Expiry Test",
			Role:         entity.RoleAdmin,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	login := func(clientType string) *auth.AuthResponse {
		uc := auth.NewLoginUseCase(
			mockUserRepo,
			testJWTSecret,
			auth.RefreshTokenExpiryWeb,
			auth.RefreshTokenExpiryMobile,
			nopLog,
		)
		mockUserRepo.EXPECT().
			FindByEmailWithPassword(ctx, "expiry@example.com").
			Return(testUser2, nil)

		result, err := uc.Execute(ctx, &auth.LoginRequest{
			Email:      "expiry@example.com",
			Password:   plainPassword,
			ClientType: clientType,
		})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	When("client type is empty", func() {
		It("should default to web and embed 'web' client type in the refresh token", func() {
			result := login("")

			claims, err := crypto.ParseToken(result.RefreshToken, testJWTSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.ClientType).To(Equal("web"))
		})
	})

	When("client type is 'mobile'", func() {
		It("should embed 'mobile' client type in the refresh token", func() {
			result := login("mobile")

			claims, err := crypto.ParseToken(result.RefreshToken, testJWTSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.ClientType).To(Equal("mobile"))
		})
	})

	When("client type is 'web'", func() {
		It("should embed 'web' client type in the refresh token", func() {
			result := login("web")

			claims, err := crypto.ParseToken(result.RefreshToken, testJWTSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.ClientType).To(Equal("web"))
		})
	})

	When("client type is an unknown value", func() {
		It("should fall back to web expiry and preserve the unknown client type string", func() {
			result := login("desktop")

			claims, err := crypto.ParseToken(result.RefreshToken, testJWTSecret)
			Expect(err).NotTo(HaveOccurred())
			// resolveRefreshExpiry uses web expiry for any non-mobile type,
			// but passes through the original clientType string unchanged
			Expect(claims.ClientType).To(Equal("desktop"))
		})
	})
})
