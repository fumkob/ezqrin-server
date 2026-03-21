package auth_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var _ = Describe("LogoutUseCase", func() {
	var (
		ctrl              *gomock.Controller
		mockBlacklistRepo *mocks.MockTokenBlacklistRepository
		useCase           *auth.LogoutUseCase
		ctx               context.Context
		nopLoggerLogout   *logger.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockBlacklistRepo = mocks.NewMockTokenBlacklistRepository(ctrl)
		nopLoggerLogout = &logger.Logger{Logger: zap.NewNop()}
		useCase = auth.NewLogoutUseCase(mockBlacklistRepo, testJWTSecret, nopLoggerLogout)
		ctx = context.Background()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// generateTokenPair creates a real access + refresh token pair for logout tests.
	generateTokenPair := func() (accessToken, refreshToken string) {
		userID := uuid.New().String()
		var err error
		accessToken, err = crypto.GenerateAccessToken(userID, "organizer", testJWTSecret, 15*time.Minute)
		Expect(err).NotTo(HaveOccurred())
		refreshToken, err = crypto.GenerateRefreshToken(
			userID,
			"organizer",
			testJWTSecret,
			"web",
			auth.RefreshTokenExpiryWeb,
		)
		Expect(err).NotTo(HaveOccurred())
		return
	}

	Describe("Execute", func() {
		When("logging out with both tokens provided", func() {
			Context("and both tokens are valid and not yet expired", func() {
				It("should blacklist both tokens and return success", func() {
					accessToken, refreshToken := generateTokenPair()

					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, accessToken, gomock.Any()).
						Return(nil)
					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(nil)

					req := &auth.LogoutRequest{
						AccessToken:  accessToken,
						RefreshToken: refreshToken,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("logging out with only the access token provided", func() {
			Context("and the refresh token field is empty", func() {
				It("should blacklist only the access token and return success", func() {
					accessToken, _ := generateTokenPair()

					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, accessToken, gomock.Any()).
						Return(nil)
					// No expectation for refresh token blacklisting

					req := &auth.LogoutRequest{
						AccessToken:  accessToken,
						RefreshToken: "",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("logging out with only the refresh token provided", func() {
			Context("and the access token field is empty", func() {
				It("should blacklist only the refresh token and return success", func() {
					_, refreshToken := generateTokenPair()

					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(nil)
					// No expectation for access token blacklisting

					req := &auth.LogoutRequest{
						AccessToken:  "",
						RefreshToken: refreshToken,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("blacklisting fails (best-effort behaviour)", func() {
			Context("and AddToBlacklist returns an error for both tokens", func() {
				It("should still return success without propagating the error", func() {
					accessToken, refreshToken := generateTokenPair()

					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, accessToken, gomock.Any()).
						Return(errors.New("redis unavailable"))
					mockBlacklistRepo.EXPECT().
						AddToBlacklist(ctx, refreshToken, gomock.Any()).
						Return(errors.New("redis unavailable"))

					req := &auth.LogoutRequest{
						AccessToken:  accessToken,
						RefreshToken: refreshToken,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("tokens are invalid (malformed JWT)", func() {
			Context("and both tokens cannot be parsed", func() {
				It("should skip blacklisting and return success", func() {
					// No mock expectations – blacklisting should never be called for invalid tokens

					req := &auth.LogoutRequest{
						AccessToken:  "not.a.valid.jwt",
						RefreshToken: "also.not.valid",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("tokens are already expired", func() {
			Context("and the token TTL is zero or negative", func() {
				It("should skip blacklisting and return success", func() {
					// Generate a token with a very short expiry and wait for it to expire.
					// For unit-test speed, use a 1-nanosecond expiry.
					userID := uuid.New().String()
					expiredToken, err := crypto.GenerateAccessToken(userID, "organizer", testJWTSecret, 1)
					Expect(err).NotTo(HaveOccurred())

					// Sleep just enough to ensure the token is expired
					time.Sleep(5 * time.Millisecond)

					// No AddToBlacklist calls expected because token is expired
					req := &auth.LogoutRequest{
						AccessToken: expiredToken,
					}

					result, logoutErr := useCase.Execute(ctx, req)

					Expect(logoutErr).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})

		When("both token fields are empty", func() {
			Context("and the request carries no tokens at all", func() {
				It("should return success without calling the blacklist repository", func() {
					// No mock expectations at all

					req := &auth.LogoutRequest{
						AccessToken:  "",
						RefreshToken: "",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Message).To(Equal("Successfully logged out"))
				})
			})
		})
	})
})
