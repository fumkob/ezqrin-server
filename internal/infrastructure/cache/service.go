package cache

// Service defines the complete cache service interface.
// It combines health checking (infrastructure concern) with graceful shutdown capability.
//
// The Service interface provides:
//   - Health checking for monitoring and readiness probes
//   - Graceful connection shutdown
//
// Implementation: redis.Client satisfies this interface.
//
// Note: For repository initialization, use type assertion in the composition root:
//
//	if redisClient, ok := cacheService.(*redis.Client); ok {
//	    cacheRepo := redis.NewCacheRepository(redisClient)
//	    blacklistRepo := redis.NewTokenBlacklistRepository(redisClient)
//	}
type Service interface {
	// HealthChecker provides cache connectivity health status
	HealthChecker // Ping(ctx) error

	// Close gracefully shuts down the cache connection.
	// It waits for pending operations to complete.
	Close() error
}
