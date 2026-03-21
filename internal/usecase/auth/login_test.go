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

var _ = Describe("LoginUseCase", func() {
	var (
		ctrl         *gomock.Controller
		mockUserRepo *mocks.MockUserRepository
		useCase      *auth.LoginUseCase
		ctx          context.Context
		nopLogger    *logger.Logger
		testUser     *entity.User
		passwordHash string
	)

	const testPassword = "ValidPassword1!"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mocks.NewMockUserRepository(ctrl)
		nopLogger = &logger.Logger{Logger: zap.NewNop()}
		useCase = auth.NewLoginUseCase(
			mockUserRepo,
			testJWTSecret,
			auth.RefreshTokenExpiryWeb,
			auth.RefreshTokenExpiryMobile,
			nopLogger,
		)
		ctx = context.Background()

		// Pre-compute a real bcrypt hash for testPassword
		var err error
		passwordHash, err = crypto.HashPassword(testPassword)
		Expect(err).NotTo(HaveOccurred())

		testUser = &entity.User{
			ID:           uuid.New(),
			Email:        "bob@example.com",
			PasswordHash: passwordHash,
			Name:         "Bob",
			Role:         entity.RoleOrganizer,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Execute", func() {
		When("logging in with valid credentials", func() {
			Context("with default (web) client type", func() {
				It("should return access and refresh tokens", func() {
					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "bob@example.com").
						Return(testUser, nil)

					req := &auth.LoginRequest{
						Email:    "bob@example.com",
						Password: testPassword,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.AccessToken).NotTo(BeEmpty())
					Expect(result.RefreshToken).NotTo(BeEmpty())
					Expect(result.TokenType).To(Equal("Bearer"))
					Expect(result.ExpiresIn).To(Equal(int(auth.AccessTokenExpiry.Seconds())))
					Expect(result.User).NotTo(BeNil())
					Expect(result.User.Email).To(Equal("bob@example.com"))
					// Password hash must be cleared before returning
					Expect(result.User.PasswordHash).To(BeEmpty())
				})
			})

			Context("with mobile client type", func() {
				It("should embed mobile client type in the refresh token claims", func() {
					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "bob@example.com").
						Return(testUser, nil)

					req := &auth.LoginRequest{
						Email:      "bob@example.com",
						Password:   testPassword,
						ClientType: "mobile",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.RefreshToken).NotTo(BeEmpty())

					// Verify the client type is embedded in the refresh token claims
					claims, parseErr := crypto.ParseToken(result.RefreshToken, testJWTSecret)
					Expect(parseErr).NotTo(HaveOccurred())
					Expect(claims.ClientType).To(Equal("mobile"))
				})
			})

			Context("with explicit web client type", func() {
				It("should embed web client type in the refresh token claims", func() {
					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "bob@example.com").
						Return(testUser, nil)

					req := &auth.LoginRequest{
						Email:      "bob@example.com",
						Password:   testPassword,
						ClientType: "web",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())

					claims, parseErr := crypto.ParseToken(result.RefreshToken, testJWTSecret)
					Expect(parseErr).NotTo(HaveOccurred())
					Expect(claims.ClientType).To(Equal("web"))
				})
			})
		})

		When("validating the login request", func() {
			Context("with empty email", func() {
				It("should return a validation error", func() {
					req := &auth.LoginRequest{
						Email:    "",
						Password: testPassword,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with invalid email format", func() {
				It("should return a validation error", func() {
					req := &auth.LoginRequest{
						Email:    "invalid-email",
						Password: testPassword,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with empty password", func() {
				It("should return a validation error", func() {
					req := &auth.LoginRequest{
						Email:    "bob@example.com",
						Password: "",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})
		})

		When("the email is not found in the repository", func() {
			Context("and FindByEmailWithPassword returns an error", func() {
				It("should return an unauthorized error to avoid user enumeration", func() {
					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "unknown@example.com").
						Return(nil, errors.New("user not found"))

					req := &auth.LoginRequest{
						Email:    "unknown@example.com",
						Password: testPassword,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(Equal("invalid credentials"))
				})
			})
		})

		When("the user has been soft-deleted", func() {
			Context("and user.IsDeleted() is true", func() {
				It("should return an unauthorized error to avoid user enumeration", func() {
					deletedAt := time.Now()
					deletedUser := &entity.User{
						ID:           uuid.New(),
						Email:        "deleted@example.com",
						PasswordHash: passwordHash,
						Name:         "Deleted User",
						Role:         entity.RoleOrganizer,
						DeletedAt:    &deletedAt,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}

					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "deleted@example.com").
						Return(deletedUser, nil)

					req := &auth.LoginRequest{
						Email:    "deleted@example.com",
						Password: testPassword,
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(Equal("invalid credentials"))
				})
			})
		})

		When("the password does not match", func() {
			Context("and ComparePassword fails", func() {
				It("should return an unauthorized error to avoid user enumeration", func() {
					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "bob@example.com").
						Return(testUser, nil)

					req := &auth.LoginRequest{
						Email:    "bob@example.com",
						Password: "WrongPassword!",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeUnauthorized))
					Expect(appErr.Message).To(Equal("invalid credentials"))
				})
			})
		})

		When("generating tokens fails due to an empty JWT secret", func() {
			Context("and the use case is constructed with an empty secret", func() {
				It("should return an internal error", func() {
					useCaseNoSecret := auth.NewLoginUseCase(
						mockUserRepo,
						"", // empty secret causes token generation to fail
						auth.RefreshTokenExpiryWeb,
						auth.RefreshTokenExpiryMobile,
						nopLogger,
					)

					mockUserRepo.EXPECT().
						FindByEmailWithPassword(ctx, "bob@example.com").
						Return(testUser, nil)

					req := &auth.LoginRequest{
						Email:    "bob@example.com",
						Password: testPassword,
					}

					result, err := useCaseNoSecret.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeInternal))
				})
			})
		})
	})
})
