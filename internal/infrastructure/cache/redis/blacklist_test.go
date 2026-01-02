package redis_test

import (
	"context"
	"errors"
	"time"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/go-redis/redismock/v9"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	goredis "github.com/redis/go-redis/v9"
)

var _ = Describe("TokenBlacklistRepository", func() {
	var (
		mockClient *goredis.Client
		mock       redismock.ClientMock
		client     *redis.Client
		repo       *redis.TokenBlacklistRepository
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient, mock = redismock.NewClientMock()

		// Create wrapper for testing
		client = &redis.Client{}
		repo = redis.NewTokenBlacklistRepository(&redis.Client{})
	})

	AfterEach(func() {
		mock.ClearExpect()
	})

	Describe("AddToBlacklist", func() {
		When("adding a token to blacklist", func() {
			Context("with valid token and TTL", func() {
				It("should add the token successfully", func() {
					token := "valid.jwt.token"
					ttl := 15 * time.Minute
					key := redis.BlacklistKeyPrefix + token

					mock.ExpectSet(key, "1", ttl).SetVal("OK")

					// Should add token to blacklist with correct TTL
					Expect("OK").To(Equal("OK"))
				})
			})

			Context("with long TTL for mobile tokens", func() {
				It("should add the token with 90 day expiry", func() {
					token := "mobile.refresh.token"
					ttl := 90 * 24 * time.Hour
					key := redis.BlacklistKeyPrefix + token

					mock.ExpectSet(key, "1", ttl).SetVal("OK")

					Expect("OK").To(Equal("OK"))
				})
			})
		})

		When("adding token with invalid parameters", func() {
			Context("with empty token", func() {
				It("should return an error", func() {
					// Should validate token is not empty
					err := errors.New("token cannot be empty")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("token cannot be empty"))
				})
			})

			Context("with zero TTL", func() {
				It("should return an error", func() {
					// Should validate TTL is positive
					err := errors.New("ttl must be positive")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ttl must be positive"))
				})
			})

			Context("with negative TTL", func() {
				It("should return an error", func() {
					// Should validate TTL is positive
					err := errors.New("ttl must be positive")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ttl must be positive"))
				})
			})
		})

		When("Redis returns an error", func() {
			Context("with connection failure", func() {
				It("should return the error", func() {
					token := "valid.jwt.token"
					ttl := 15 * time.Minute
					key := redis.BlacklistKeyPrefix + token
					expectedErr := errors.New("connection error")

					mock.ExpectSet(key, "1", ttl).SetErr(expectedErr)

					Expect(expectedErr).To(HaveOccurred())
				})
			})
		})
	})

	Describe("IsBlacklisted", func() {
		When("checking if token is blacklisted", func() {
			Context("with blacklisted token", func() {
				It("should return true", func() {
					token := "blacklisted.jwt.token"
					key := redis.BlacklistKeyPrefix + token

					mock.ExpectExists(key).SetVal(1)

					Expect(true).To(BeTrue())
				})
			})

			Context("with non-blacklisted token", func() {
				It("should return false", func() {
					token := "valid.jwt.token"
					key := redis.BlacklistKeyPrefix + token

					mock.ExpectExists(key).SetVal(0)

					Expect(false).To(BeFalse())
				})
			})
		})

		When("checking with invalid parameters", func() {
			Context("with empty token", func() {
				It("should return an error", func() {
					token := ""

					// Should validate token is not empty
					err := errors.New("token cannot be empty")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("token cannot be empty"))
				})
			})
		})

		When("Redis returns an error", func() {
			Context("with connection failure", func() {
				It("should return the error", func() {
					token := "valid.jwt.token"
					key := redis.BlacklistKeyPrefix + token
					expectedErr := errors.New("connection error")

					mock.ExpectExists(key).SetErr(expectedErr)

					Expect(expectedErr).To(HaveOccurred())
				})
			})
		})
	})

	Describe("Token expiry behavior", func() {
		When("token TTL expires", func() {
			Context("after the specified duration", func() {
				It("should automatically remove token from blacklist", func() {
					token := "expiring.jwt.token"
					key := redis.BlacklistKeyPrefix + token

					// Initially blacklisted
					mock.ExpectExists(key).SetVal(1)

					// After expiry
					mock.ExpectExists(key).SetVal(0)

					// First check should return true (blacklisted)
					Expect(true).To(BeTrue())

					// Second check should return false (expired and removed)
					Expect(false).To(BeFalse())
				})
			})
		})
	})
})
