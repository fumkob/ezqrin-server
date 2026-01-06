package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/entity"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/fumkob/ezqrin-server/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// PasswordMinLength is the minimum password length
	PasswordMinLength = 8

	// AccessTokenExpiry is the expiry duration for access tokens (15 minutes)
	AccessTokenExpiry = 15 * time.Minute

	// RefreshTokenExpiry is the expiry duration for refresh tokens (7 days)
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// RegisterUseCase handles user registration
type RegisterUseCase struct {
	userRepo  repository.UserRepository
	jwtSecret string
	logger    *logger.Logger
}

// NewRegisterUseCase creates a new RegisterUseCase
func NewRegisterUseCase(userRepo repository.UserRepository, jwtSecret string, logger *logger.Logger) *RegisterUseCase {
	return &RegisterUseCase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// RegisterRequest represents the input for user registration
type RegisterRequest struct {
	Email    string
	Password string
	Name     string
	Role     string
}

// AuthResponse represents the authentication response with tokens
type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	User         *entity.User
}

// Execute executes the user registration use case
func (u *RegisterUseCase) Execute(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Validate input
	if err := u.validateRequest(req); err != nil {
		return nil, err
	}

	// Create and validate user
	user, err := u.createUser(ctx, req)
	if err != nil {
		return nil, err
	}

	// Generate authentication tokens
	accessToken, refreshToken, err := u.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	u.logger.WithContext(ctx).Info(fmt.Sprintf("user registered successfully: %s", user.ID))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
		User:         user,
	}, nil
}

// createUser creates and persists a new user entity
func (u *RegisterUseCase) createUser(ctx context.Context, req *RegisterRequest) (*entity.User, error) {
	// Check if email already exists
	exists, err := u.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to check email existence", zap.Error(err))
		return nil, apperrors.Internal("failed to check email existence")
	}
	if exists {
		return nil, apperrors.Conflict("email already exists")
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to hash password", zap.Error(err))
		return nil, apperrors.Internal("failed to hash password")
	}

	// Create user entity
	now := time.Now()
	user := &entity.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		Role:         entity.UserRole(req.Role),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Validate user entity
	if err := user.Validate(); err != nil {
		return nil, apperrors.Validation(err.Error())
	}

	// Save user to database
	if err := u.userRepo.Create(ctx, user); err != nil {
		u.logger.WithContext(ctx).Error("failed to create user", zap.Error(err))
		return nil, apperrors.Internal("failed to create user")
	}

	return user, nil
}

// generateTokens generates access and refresh tokens for a user
func (u *RegisterUseCase) generateTokens(ctx context.Context, user *entity.User) (string, string, error) {
	accessToken, err := crypto.GenerateAccessToken(user.ID.String(), string(user.Role), u.jwtSecret, AccessTokenExpiry)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to generate access token", zap.Error(err))
		return "", "", apperrors.Internal("failed to generate access token")
	}

	refreshToken, err := crypto.GenerateRefreshToken(
		user.ID.String(),
		string(user.Role),
		u.jwtSecret,
		RefreshTokenExpiry,
	)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to generate refresh token", zap.Error(err))
		return "", "", apperrors.Internal("failed to generate refresh token")
	}

	return accessToken, refreshToken, nil
}

// validateRequest validates the registration request
func (u *RegisterUseCase) validateRequest(req *RegisterRequest) error {
	// Validate email
	if err := validator.ValidateEmail(req.Email); err != nil {
		return apperrors.Validation(err.Error())
	}

	// Validate password
	if err := validator.ValidateRequired(req.Password, "password"); err != nil {
		return apperrors.Validation(err.Error())
	}
	if err := validator.ValidateMinLength(req.Password, PasswordMinLength, "password"); err != nil {
		return apperrors.Validation(err.Error())
	}

	// Validate name
	if err := validator.ValidateRequired(req.Name, "name"); err != nil {
		return apperrors.Validation(err.Error())
	}
	if err := validator.ValidateMinLength(req.Name, entity.UserNameMinLength, "name"); err != nil {
		return apperrors.Validation(err.Error())
	}
	if err := validator.ValidateMaxLength(req.Name, entity.UserNameMaxLength, "name"); err != nil {
		return apperrors.Validation(err.Error())
	}

	// Validate role
	if err := validator.ValidateRequired(req.Role, "role"); err != nil {
		return apperrors.Validation(err.Error())
	}
	if err := entity.ValidateRole(req.Role); err != nil {
		return apperrors.Validation(err.Error())
	}

	return nil
}
