// go:build integration
//go:build integration
// +build integration

package redis_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis Integration Tests", func() {
	var (
		client        *redis.Client
		cacheRepo     *redis.CacheRepository
		blacklistRepo *redis.TokenBlacklistRepository
		ctx           context.Context
		redisHost     string
		redisPort     string
		redisPassword string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Get Redis connection details from environment variables
		redisHost = os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}

		redisPort = os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379"
		}

		redisPassword = os.Getenv("REDIS_PASSWORD")

		// Create Redis client
		cfg := &redis.ClientConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: redisPassword,
			DB:       1, // Use DB 1 for tests to avoid conflicts
		}

		var err error
		client, err = redis.NewClient(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(client).ToNot(BeNil())

		// Create repositories
		cacheRepo = redis.NewCacheRepository(client)
		blacklistRepo = redis.NewTokenBlacklistRepository(client)

		// Clean up any existing test data
		client.GetClient().FlushDB(ctx)
	})

	AfterEach(func() {
		if client != nil {
			// Clean up test data
			client.GetClient().FlushDB(ctx)
			client.Close()
		}
	})

	Describe("CacheRepository Integration", func() {
		When("performing cache operations", func() {
			Context("with Set and Get", func() {
				It("should store and retrieve values correctly", func() {
					key := "integration:test:key"
					value := "test-value"
					ttl := 10 * time.Second

					err := cacheRepo.Set(ctx, key, value, ttl)
					Expect(err).ToNot(HaveOccurred())

					result, err := cacheRepo.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(Equal(value))
				})
			})

			Context("with TTL expiration", func() {
				It("should expire keys after TTL", func() {
					key := "integration:test:expiring"
					value := "expiring-value"
					ttl := 1 * time.Second

					err := cacheRepo.Set(ctx, key, value, ttl)
					Expect(err).ToNot(HaveOccurred())

					// Verify key exists
					exists, err := cacheRepo.Exists(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					// Wait for expiration
					time.Sleep(2 * time.Second)

					// Verify key no longer exists
					exists, err = cacheRepo.Exists(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})

			Context("with Delete operation", func() {
				It("should remove keys successfully", func() {
					key := "integration:test:delete"
					value := "delete-me"

					err := cacheRepo.Set(ctx, key, value, 0)
					Expect(err).ToNot(HaveOccurred())

					err = cacheRepo.Delete(ctx, key)
					Expect(err).ToNot(HaveOccurred())

					result, err := cacheRepo.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeEmpty())
				})
			})

			Context("with batch operations (MGet/MSet)", func() {
				It("should handle multiple keys efficiently", func() {
					items := map[string]string{
						"integration:batch:key1": "value1",
						"integration:batch:key2": "value2",
						"integration:batch:key3": "value3",
					}
					ttl := 10 * time.Second

					err := cacheRepo.MSet(ctx, items, ttl)
					Expect(err).ToNot(HaveOccurred())

					keys := []string{
						"integration:batch:key1",
						"integration:batch:key2",
						"integration:batch:key3",
					}

					results, err := cacheRepo.MGet(ctx, keys)
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(3))
					Expect(results["integration:batch:key1"]).To(Equal("value1"))
					Expect(results["integration:batch:key2"]).To(Equal("value2"))
					Expect(results["integration:batch:key3"]).To(Equal("value3"))
				})
			})

			Context("with latency requirements", func() {
				It("should perform operations within 5ms", func() {
					key := "integration:test:latency"
					value := "fast-value"

					start := time.Now()
					err := cacheRepo.Set(ctx, key, value, 10*time.Second)
					setDuration := time.Since(start)

					Expect(err).ToNot(HaveOccurred())
					Expect(setDuration).To(BeNumerically("<", 5*time.Millisecond))

					start = time.Now()
					_, err = cacheRepo.Get(ctx, key)
					getDuration := time.Since(start)

					Expect(err).ToNot(HaveOccurred())
					Expect(getDuration).To(BeNumerically("<", 5*time.Millisecond))
				})
			})
		})

		When("checking connectivity", func() {
			Context("with Ping operation", func() {
				It("should verify Redis is reachable", func() {
					err := cacheRepo.Ping(ctx)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("TokenBlacklistRepository Integration", func() {
		When("managing token blacklist", func() {
			Context("with AddToBlacklist and IsBlacklisted", func() {
				It("should blacklist tokens correctly", func() {
					token := "integration.test.jwt.token"
					ttl := 15 * time.Minute

					err := blacklistRepo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())

					isBlacklisted, err := blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())
				})
			})

			Context("with non-blacklisted token", func() {
				It("should return false for valid tokens", func() {
					token := "valid.jwt.token"

					isBlacklisted, err := blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeFalse())
				})
			})

			Context("with token expiry", func() {
				It("should automatically remove expired tokens", func() {
					token := "expiring.jwt.token"
					ttl := 1 * time.Second

					err := blacklistRepo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())

					// Verify blacklisted
					isBlacklisted, err := blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())

					// Wait for expiry
					time.Sleep(2 * time.Second)

					// Verify no longer blacklisted
					isBlacklisted, err = blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeFalse())
				})
			})

			Context("with different TTL durations", func() {
				It("should handle short access token TTL", func() {
					token := "access.token.short"
					ttl := 15 * time.Minute

					err := blacklistRepo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())

					isBlacklisted, err := blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())
				})

				It("should handle long refresh token TTL (mobile)", func() {
					token := "refresh.token.mobile"
					ttl := 90 * 24 * time.Hour // 90 days

					err := blacklistRepo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())

					isBlacklisted, err := blacklistRepo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())
				})
			})
		})
	})

	Describe("Connection reliability", func() {
		When("handling connection issues", func() {
			Context("with invalid credentials", func() {
				It("should fail to connect gracefully", func() {
					invalidCfg := &redis.ClientConfig{
						Host:     redisHost,
						Port:     redisPort,
						Password: "wrong-password",
						DB:       0,
					}

					_, err := redis.NewClient(invalidCfg)
					if redisPassword != "" {
						Expect(err).To(HaveOccurred())
					}
				})
			})

			Context("with connection pool", func() {
				It("should maintain multiple connections", func() {
					stats := client.Stats()
					Expect(stats).ToNot(BeNil())

					// Perform multiple operations to utilize pool
					for i := 0; i < 20; i++ {
						key := fmt.Sprintf("pool:test:%d", i)
						err := cacheRepo.Set(ctx, key, "value", 1*time.Minute)
						Expect(err).ToNot(HaveOccurred())
					}

					// Verify pool statistics
					stats = client.Stats()
					Expect(stats.TotalConns).To(BeNumerically(">", 0))
				})
			})
		})
	})
})
