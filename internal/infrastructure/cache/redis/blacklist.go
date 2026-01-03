package redis

import (
	"context"
	"fmt"
	"time"
)

const (
	// BlacklistKeyPrefix is the prefix for blacklisted token keys.
	BlacklistKeyPrefix = "blacklist:token:"
)

// TokenBlacklistRepository implements the domain token blacklist repository using Redis.
type TokenBlacklistRepository struct {
	client *Client
}

// NewTokenBlacklistRepository creates a new Redis-based token blacklist repository.
func NewTokenBlacklistRepository(client *Client) *TokenBlacklistRepository {
	return &TokenBlacklistRepository{
		client: client,
	}
}

// AddToBlacklist adds a token to the blacklist with TTL matching token expiry.
// The token will be automatically removed from blacklist after TTL expires.
func (r *TokenBlacklistRepository) AddToBlacklist(ctx context.Context, token string, ttl time.Duration) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	key := r.makeKey(token)

	// Store a placeholder value (we only care about key existence)
	err := r.client.Set(ctx, key, "1", ttl)
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

// IsBlacklisted checks if a token is in the blacklist.
func (r *TokenBlacklistRepository) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token cannot be empty")
	}

	key := r.makeKey(token)

	exists, err := r.client.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist status: %w", err)
	}

	return exists > 0, nil
}

// makeKey creates a Redis key for a blacklisted token.
func (r *TokenBlacklistRepository) makeKey(token string) string {
	return BlacklistKeyPrefix + token
}
