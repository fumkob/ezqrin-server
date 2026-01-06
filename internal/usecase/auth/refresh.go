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

// RefreshTokenUseCase handles refresh token rotation
type RefreshTokenUseCase struct {
	userRepo      repository.UserRepository
	blacklistRepo repository.TokenBlacklistRepository
	jwtSecret     string
	logger        *logger.Logger
}

// NewRefreshTokenUseCase creates a new RefreshTokenUseCase
func NewRefreshTokenUseCase(
	userRepo repository.UserRepository,
	blacklistRepo repository.TokenBlacklistRepository,
	jwtSecret string,
	logger *logger.Logger,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		userRepo:      userRepo,
		blacklistRepo: blacklistRepo,
		jwtSecret:     jwtSecret,
		logger:        logger,
	}
}

// RefreshRequest represents the input for token refresh
type RefreshRequest struct {
	RefreshToken string
}

// Execute executes the refresh token use case
func (u *RefreshTokenUseCase) Execute(ctx context.Context, req *RefreshRequest) (*AuthResponse, error) {
	// Validate input
	if err := validator.ValidateRequired(req.RefreshToken, "refresh_token"); err != nil {
		return nil, apperrors.Validation(err.Error())
	}

	// Validate and parse token
	claims, err := u.validateToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Get and validate user
	user, err := u.validateUser(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := u.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	// Blacklist old refresh token (best effort)
	u.blacklistOldToken(ctx, req.RefreshToken, claims)

	u.logger.WithContext(ctx).Info(fmt.Sprintf("refresh token rotated for user: %s", user.ID))

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
		User:         user,
	}, nil
}

// validateToken validates the refresh token and returns claims
func (u *RefreshTokenUseCase) validateToken(ctx context.Context, token string) (*crypto.Claims, error) {
	claims, err := crypto.ParseToken(token, u.jwtSecret)
	if err != nil {
		if err == crypto.ErrExpiredToken {
			return nil, apperrors.Unauthorized("refresh token has expired")
		}
		u.logger.WithContext(ctx).Warn("invalid refresh token", zap.Error(err))
		return nil, apperrors.Unauthorized("invalid refresh token")
	}

	// Verify token type
	if claims.TokenType != crypto.TokenTypeRefresh {
		u.logger.WithContext(ctx).Warn("attempted to refresh with non-refresh token")
		return nil, apperrors.Unauthorized("invalid token type")
	}

	// Check if token is blacklisted
	isBlacklisted, err := u.blacklistRepo.IsBlacklisted(ctx, token)
	if err != nil {
		u.logger.WithContext(ctx).Error("failed to check token blacklist", zap.Error(err))
		return nil, apperrors.Internal("failed to validate token")
	}
	if isBlacklisted {
		u.logger.WithContext(ctx).Warn(fmt.Sprintf("attempted use of blacklisted token for user: %s", claims.UserID))
		return nil, apperrors.Unauthorized("token has been revoked")
	}

	return claims, nil
}

// validateUser finds and validates the user
func (u *RefreshTokenUseCase) validateUser(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		u.logger.WithContext(ctx).Warn(fmt.Sprintf("user not found for refresh token: %s", userID), zap.Error(err))
		return nil, apperrors.Unauthorized("user not found")
	}

	if user.IsDeleted() {
		u.logger.WithContext(ctx).Warn(fmt.Sprintf("refresh attempt for deleted user: %s", user.ID))
		return nil, apperrors.Unauthorized("user not found")
	}

	return user, nil
}

// generateTokens generates new access and refresh tokens
func (u *RefreshTokenUseCase) generateTokens(ctx context.Context, user *entity.User) (string, string, error) {
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

// blacklistOldToken blacklists the old refresh token (best effort)
func (u *RefreshTokenUseCase) blacklistOldToken(ctx context.Context, token string, claims *crypto.Claims) {
	expiryTime := claims.ExpiresAt.Time
	ttl := time.Until(expiryTime)
	if ttl <= 0 {
		return
	}

	if err := u.blacklistRepo.AddToBlacklist(ctx, token, ttl); err != nil {
		// Log error but don't fail the request - token rotation is more important
		u.logger.WithContext(ctx).Warn("failed to blacklist old refresh token", zap.Error(err))
	}
}

// ParseTokenForLogout parses a token without validating expiry (for logout)
func ParseTokenForLogout(tokenString, secret string) (*uuid.UUID, time.Duration, error) {
	if tokenString == "" {
		return nil, 0, nil
	}

	claims, err := crypto.ParseToken(tokenString, secret)
	if err != nil {
		// For logout, we allow expired tokens
		if err == crypto.ErrExpiredToken {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	// Calculate TTL
	expiryTime := claims.ExpiresAt.Time
	ttl := time.Until(expiryTime)
	if ttl <= 0 {
		return nil, 0, nil
	}

	return &claims.UserID, ttl, nil
}
