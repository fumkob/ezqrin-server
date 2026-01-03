package redis

import (
	goredis "github.com/redis/go-redis/v9"
)

// newTestClient creates a Client with the provided redis client for testing.
// This function is only available during tests and can access unexported fields.
// It allows injecting mock redis clients for unit testing.
func newTestClient(client *goredis.Client) *Client {
	return &Client{
		client: client,
		config: &ClientConfig{}, // Empty config for tests
	}
}
