package repository

import (
	"context"
	"time"
)

// CacheRepository defines the interface for caching operations.
// Implementations should provide distributed caching capabilities with TTL support.
type CacheRepository interface {
	// Get retrieves a value from cache by key.
	// Returns empty string if key doesn't exist.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value in cache with the specified TTL.
	// If ttl is 0, the key will not expire.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Delete removes a key from cache.
	// No error is returned if key doesn't exist.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache.
	Exists(ctx context.Context, key string) (bool, error)

	// MGet retrieves multiple values from cache by keys.
	// Returns a map where missing keys are not included.
	MGet(ctx context.Context, keys []string) (map[string]string, error)

	// MSet stores multiple key-value pairs in cache with the same TTL.
	MSet(ctx context.Context, items map[string]string, ttl time.Duration) error

	// Ping checks if the cache service is reachable.
	Ping(ctx context.Context) error
}

// TokenBlacklistRepository defines the interface for JWT token blacklist operations.
// Used to invalidate tokens before their expiration time.
type TokenBlacklistRepository interface {
	// AddToBlacklist adds a token to the blacklist with TTL matching token expiry.
	// The token will be automatically removed from blacklist after TTL expires.
	AddToBlacklist(ctx context.Context, token string, ttl time.Duration) error

	// IsBlacklisted checks if a token is in the blacklist.
	IsBlacklisted(ctx context.Context, token string) (bool, error)
}
