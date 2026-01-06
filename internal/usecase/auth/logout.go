package auth

import (
	"context"
	"time"

	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/pkg/crypto"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"go.uber.org/zap"
)

// LogoutUseCase handles user logout
type LogoutUseCase struct {
	blacklistRepo repository.TokenBlacklistRepository
	jwtSecret     string
	logger        *logger.Logger
}

// NewLogoutUseCase creates a new LogoutUseCase
func NewLogoutUseCase(
	blacklistRepo repository.TokenBlacklistRepository,
	jwtSecret string,
	logger *logger.Logger,
) *LogoutUseCase {
	return &LogoutUseCase{
		blacklistRepo: blacklistRepo,
		jwtSecret:     jwtSecret,
		logger:        logger,
	}
}

// LogoutRequest represents the input for user logout
type LogoutRequest struct {
	AccessToken  string
	RefreshToken string
}

// LogoutResponse represents the logout response
type LogoutResponse struct {
	Message string
}

// Execute executes the user logout use case
// This is a best-effort operation - even invalid tokens should succeed
func (u *LogoutUseCase) Execute(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	// Blacklist access token if provided
	if req.AccessToken != "" {
		if err := u.blacklistToken(ctx, req.AccessToken); err != nil {
			// Log but don't fail - best effort
			u.logger.WithContext(ctx).Warn("failed to blacklist access token", zap.Error(err))
		}
	}

	// Blacklist refresh token if provided
	if req.RefreshToken != "" {
		if err := u.blacklistToken(ctx, req.RefreshToken); err != nil {
			// Log but don't fail - best effort
			u.logger.WithContext(ctx).Warn("failed to blacklist refresh token", zap.Error(err))
		}
	}

	u.logger.WithContext(ctx).Info("user logged out successfully")

	return &LogoutResponse{
		Message: "Successfully logged out",
	}, nil
}

// blacklistToken blacklists a token with appropriate TTL
func (u *LogoutUseCase) blacklistToken(ctx context.Context, token string) error {
	// Parse token to get expiry time
	claims, err := crypto.ParseToken(token, u.jwtSecret)
	if err != nil {
		// For logout, we allow expired tokens - no need to blacklist
		if err == crypto.ErrExpiredToken {
			return nil
		}
		// For invalid tokens, skip blacklisting
		return err
	}

	// Calculate TTL as time until token expires
	expiryTime := claims.ExpiresAt.Time
	ttl := time.Until(expiryTime)

	// If token is already expired, no need to blacklist
	if ttl <= 0 {
		return nil
	}

	// Add token to blacklist
	return u.blacklistRepo.AddToBlacklist(ctx, token, ttl)
}
