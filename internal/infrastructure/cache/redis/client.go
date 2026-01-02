package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Default connection pool configuration
	defaultPoolSize     = 10
	defaultMinIdleConns = 5
	defaultMaxRetries   = 3

	// Default timeout configuration
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
	defaultPingTimeout  = 5 * time.Second
)

// ClientConfig holds the configuration for Redis client.
type ClientConfig struct {
	Host     string
	Port     string
	Password string
	DB       int

	// Connection pool configuration
	PoolSize     int
	MinIdleConns int
	MaxRetries   int

	// Timeout configuration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Client wraps the Redis client with additional functionality.
type Client struct {
	client *redis.Client
	config *ClientConfig
}

// NewClient creates a new Redis client with connection pooling.
// The client will automatically reconnect on connection failures.
// Config values fallback to defaults if not provided (zero values).
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	// Use config values with fallback to defaults
	poolSize := cfg.PoolSize
	if poolSize == 0 {
		poolSize = defaultPoolSize
	}

	minIdleConns := cfg.MinIdleConns
	if minIdleConns == 0 {
		minIdleConns = defaultMinIdleConns
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = defaultDialTimeout
	}

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = defaultReadTimeout
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = defaultWriteTimeout
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		MaxRetries:   maxRetries,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: rdb,
		config: cfg,
	}, nil
}

// GetClient returns the underlying Redis client.
// This is useful for advanced operations not covered by the repository interfaces.
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Ping checks if the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close gracefully closes the Redis connection.
// Should be called when shutting down the application.
func (c *Client) Close() error {
	return c.client.Close()
}

// Stats returns connection pool statistics for monitoring.
func (c *Client) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// Get retrieves a value from Redis by key.
// Returns (value, nil) on success, ("", redis.Nil) if key doesn't exist, or ("", error) on failure.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set stores a value in Redis with the specified TTL.
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del deletes one or more keys from Redis.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if one or more keys exist in Redis.
// Returns the number of existing keys.
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// MGet retrieves multiple values from Redis by keys.
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.client.MGet(ctx, keys...).Result()
}

// Pipeline returns a new pipeline for batching commands.
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}
