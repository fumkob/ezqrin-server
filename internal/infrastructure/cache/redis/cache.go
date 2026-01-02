package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheRepository implements the domain cache repository interface using Redis.
type CacheRepository struct {
	client *Client
}

// NewCacheRepository creates a new Redis-based cache repository.
func NewCacheRepository(client *Client) *CacheRepository {
	return &CacheRepository{
		client: client,
	}
}

// Get retrieves a value from cache by key.
// Returns empty string if key doesn't exist.
func (r *CacheRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key)
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in cache with the specified TTL.
// If ttl is 0, the key will not expire.
func (r *CacheRepository) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl)
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a key from cache.
// No error is returned if key doesn't exist.
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in cache.
func (r *CacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return count > 0, nil
}

// MGet retrieves multiple values from cache by keys.
// Returns a map where missing keys are not included.
func (r *CacheRepository) MGet(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	values, err := r.client.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("failed to mget keys: %w", err)
	}

	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if values[i] != nil {
			if val, ok := values[i].(string); ok {
				result[key] = val
			}
		}
	}

	return result, nil
}

// MSet stores multiple key-value pairs in cache with the same TTL.
func (r *CacheRepository) MSet(ctx context.Context, items map[string]string, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Use pipeline for atomic execution
	pipe := r.client.Pipeline()

	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mset keys: %w", err)
	}

	return nil
}

// Ping checks if the cache service is reachable.
func (r *CacheRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx)
}
