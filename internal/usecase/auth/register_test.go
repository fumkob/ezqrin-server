package auth_test

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/repository/mocks"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const testJWTSecret = "test-jwt-secret-for-unit-tests-only"

var _ = Describe("RegisterUseCase", func() {
	var (
		ctrl         *gomock.Controller
		mockUserRepo *mocks.MockUserRepository
		useCase      *auth.RegisterUseCase
		ctx          context.Context
		nopLogger    *logger.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mocks.NewMockUserRepository(ctrl)
		nopLogger = &logger.Logger{Logger: zap.NewNop()}
		useCase = auth.NewRegisterUseCase(mockUserRepo, testJWTSecret, nopLogger)
		ctx = context.Background()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Execute", func() {
		When("registering a new user", func() {
			Context("with valid input", func() {
				It("should create user and return auth tokens", func() {
					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, "alice@example.com").
						Return(false, nil)
					mockUserRepo.EXPECT().
						Create(ctx, gomock.Any()).
						Return(nil)

					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.AccessToken).NotTo(BeEmpty())
					Expect(result.RefreshToken).NotTo(BeEmpty())
					Expect(result.TokenType).To(Equal("Bearer"))
					Expect(result.ExpiresIn).To(Equal(int(auth.AccessTokenExpiry.Seconds())))
					Expect(result.User).NotTo(BeNil())
					Expect(result.User.Email).To(Equal("alice@example.com"))
					Expect(result.User.Name).To(Equal("Alice"))
					// Password hash should be present (not cleared on register)
					Expect(result.User.PasswordHash).NotTo(BeEmpty())
				})
			})

			Context("with empty email", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
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
					req := &auth.RegisterRequest{
						Email:    "not-an-email",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
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
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with password shorter than minimum length", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "short",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with empty name", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with name shorter than minimum length", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "A",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with name exceeding maximum length", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     strings.Repeat("a", 256),
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with empty role", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeValidation))
				})
			})

			Context("with invalid role", func() {
				It("should return a validation error", func() {
					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "superuser",
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

		When("the email already exists", func() {
			Context("and ExistsByEmail returns true", func() {
				It("should return a conflict error", func() {
					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, "alice@example.com").
						Return(true, nil)

					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeConflict))
				})
			})
		})

		When("checking email existence fails", func() {
			Context("and ExistsByEmail returns an error", func() {
				It("should return an internal error", func() {
					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, "alice@example.com").
						Return(false, errors.New("db connection failed"))

					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeInternal))
				})
			})
		})

		When("persisting the user fails", func() {
			Context("and Create returns an error", func() {
				It("should return an internal error", func() {
					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, "alice@example.com").
						Return(false, nil)
					mockUserRepo.EXPECT().
						Create(ctx, gomock.Any()).
						Return(errors.New("insert failed"))

					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCase.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeInternal))
					Expect(appErr.Message).To(ContainSubstring("failed to create user"))
				})
			})
		})

		When("token generation would fail due to empty secret", func() {
			Context("and the JWT secret is empty", func() {
				It("should return an internal error", func() {
					useCaseWithEmptySecret := auth.NewRegisterUseCase(mockUserRepo, "", nopLogger)

					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, "alice@example.com").
						Return(false, nil)
					mockUserRepo.EXPECT().
						Create(ctx, gomock.Any()).
						Return(nil)

					req := &auth.RegisterRequest{
						Email:    "alice@example.com",
						Password: "SecurePass1!",
						Name:     "Alice",
						Role:     "organizer",
					}

					result, err := useCaseWithEmptySecret.Execute(ctx, req)

					Expect(err).To(HaveOccurred())
					Expect(result).To(BeNil())
					var appErr *apperrors.AppError
					Expect(errors.As(err, &appErr)).To(BeTrue())
					Expect(appErr.Code).To(Equal(apperrors.CodeInternal))
				})
			})
		})

		When("registering with all valid roles", func() {
			for _, role := range []string{"admin", "organizer", "staff"} {
				// capture loop variable
				Context("with role "+role, func() {
					It("should succeed", func() {
						mockUserRepo.EXPECT().
							ExistsByEmail(ctx, gomock.Any()).
							Return(false, nil)
						mockUserRepo.EXPECT().
							Create(ctx, gomock.Any()).
							Return(nil)

						req := &auth.RegisterRequest{
							Email:    "user@example.com",
							Password: "SecurePass1!",
							Name:     "Test User",
							Role:     role,
						}

						result, err := useCase.Execute(ctx, req)

						Expect(err).NotTo(HaveOccurred())
						Expect(result).NotTo(BeNil())
						Expect(result.User.Role).To(BeEquivalentTo(role))
					})
				})
			}
		})

		When("timing is checked", func() {
			Context("and the user is successfully created", func() {
				It("should set CreatedAt and UpdatedAt within the last second", func() {
					mockUserRepo.EXPECT().
						ExistsByEmail(ctx, gomock.Any()).
						Return(false, nil)
					mockUserRepo.EXPECT().
						Create(ctx, gomock.Any()).
						Return(nil)

					req := &auth.RegisterRequest{
						Email:    "timing@example.com",
						Password: "SecurePass1!",
						Name:     "Timing Test",
						Role:     "staff",
					}

					before := time.Now()
					result, err := useCase.Execute(ctx, req)
					after := time.Now()

					Expect(err).NotTo(HaveOccurred())
					Expect(result.User.CreatedAt).To(BeTemporally(">=", before))
					Expect(result.User.CreatedAt).To(BeTemporally("<=", after))
					Expect(result.User.UpdatedAt).To(BeTemporally(">=", before))
					Expect(result.User.UpdatedAt).To(BeTemporally("<=", after))
				})
			})
		})
	})
})
