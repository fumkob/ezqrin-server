package auth

import (
	"context"
	"fmt"

	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/fumkob/ezqrin-server/pkg/validator"
	"go.uber.org/zap"
)

// LoginUseCase handles user login
type LoginUseCase struct {
	userRepo  repository.UserRepository
	jwtSecret string
	logger    *logger.Logger
}

// NewLoginUseCase creates a new LoginUseCase
func NewLoginUseCase(userRepo repository.UserRepository, jwtSecret string, logger *logger.Logger) *LoginUseCase {
	return &LoginUseCase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// LoginRequest represents the input for user login
type LoginRequest struct {
	Email    string
	Password string
}

// Execute executes the user login use case
func (u *LoginUseCase) Execute(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// Validate input
	if err := u.validateRequest(req); err != nil {
		return nil, err
	}

	// Find user by email with password hash
	user, err := u.userRepo.FindByEmailWithPassword(ctx, req.Email)
	if err != nil {
		u.logger.WithContext(ctx).Warn("login attempt with non-existent email", zap.Error(err))
		return nil, apperrors.Unauthorized("invalid credentials")
	}

	// Check if user is deleted
	if user.IsDeleted() {
		u.logger.WithContext(ctx).Warn(fmt.Sprintf("login attempt for deleted user: %s", user.ID))
		return nil, apperrors.Unauthorized("invalid credentials")
	}

	// Compare password with hash
	if err := crypto.ComparePassword(user.PasswordHash, req.Password); err != nil {
		u.logger.WithContext(ctx).Warn(fmt.Sprintf("invalid password attempt for user: %s", user.ID))
		return nil, apperrors.Unauthorized("invalid credentials")
	}

	// Generate access token
	accessToken, err := crypto.GenerateAccessToken(user.ID.String(), string(user.Role), u.jwtSecret, AccessTokenExpiry)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to generate access token", zap.Error(err))
		return nil, apperrors.Internal("failed to generate access token")
	}

	// Generate refresh token
	refreshToken, err := crypto.GenerateRefreshToken(
		user.ID.String(),
		string(user.Role),
		u.jwtSecret,
		RefreshTokenExpiry,
	)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to generate refresh token", zap.Error(err))
		return nil, apperrors.Internal("failed to generate refresh token")
	}

	u.logger.WithContext(ctx).Info(fmt.Sprintf("user logged in successfully: %s", user.ID))

	// Clear password hash before returning
	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
		User:         user,
	}, nil
}

// validateRequest validates the login request
func (u *LoginUseCase) validateRequest(req *LoginRequest) error {
	// Validate email
	if err := validator.ValidateEmail(req.Email); err != nil {
		return apperrors.Validation(err.Error())
	}

	// Validate password
	if err := validator.ValidateRequired(req.Password, "password"); err != nil {
		return apperrors.Validation(err.Error())
	}

	return nil
}
