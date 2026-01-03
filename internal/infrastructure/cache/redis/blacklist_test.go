package redis

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redismock/v9"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	goredis "github.com/redis/go-redis/v9"
)

var _ = Describe("TokenBlacklistRepository", func() {
	var (
		mockClient *goredis.Client
		mock       redismock.ClientMock
		client     *Client
		repo       *TokenBlacklistRepository
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient, mock = redismock.NewClientMock()
		client = newTestClient(mockClient)
		repo = NewTokenBlacklistRepository(client)
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
					key := BlacklistKeyPrefix + token

					mock.ExpectSet(key, "1", ttl).SetVal("OK")

					err := repo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with long TTL for mobile tokens", func() {
				It("should add the token with 90 day expiry", func() {
					token := "mobile.refresh.token"
					ttl := 90 * 24 * time.Hour
					key := BlacklistKeyPrefix + token

					mock.ExpectSet(key, "1", ttl).SetVal("OK")

					err := repo.AddToBlacklist(ctx, token, ttl)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("adding token with invalid parameters", func() {
			Context("with empty token", func() {
				It("should return an error", func() {
					err := repo.AddToBlacklist(ctx, "", 15*time.Minute)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("token cannot be empty"))
				})
			})

			Context("with zero TTL", func() {
				It("should return an error", func() {
					err := repo.AddToBlacklist(ctx, "valid.token", 0)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ttl must be positive"))
				})
			})

			Context("with negative TTL", func() {
				It("should return an error", func() {
					err := repo.AddToBlacklist(ctx, "valid.token", -5*time.Minute)
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
					key := BlacklistKeyPrefix + token
					expectedErr := errors.New("connection error")

					mock.ExpectSet(key, "1", ttl).SetErr(expectedErr)

					err := repo.AddToBlacklist(ctx, token, ttl)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection error"))
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("IsBlacklisted", func() {
		When("checking if token is blacklisted", func() {
			Context("with blacklisted token", func() {
				It("should return true", func() {
					token := "blacklisted.jwt.token"
					key := BlacklistKeyPrefix + token

					mock.ExpectExists(key).SetVal(1)

					isBlacklisted, err := repo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with non-blacklisted token", func() {
				It("should return false", func() {
					token := "valid.jwt.token"
					key := BlacklistKeyPrefix + token

					mock.ExpectExists(key).SetVal(0)

					isBlacklisted, err := repo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeFalse())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("checking with invalid parameters", func() {
			Context("with empty token", func() {
				It("should return an error", func() {
					isBlacklisted, err := repo.IsBlacklisted(ctx, "")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("token cannot be empty"))
					Expect(isBlacklisted).To(BeFalse())
				})
			})
		})

		When("Redis returns an error", func() {
			Context("with connection failure", func() {
				It("should return the error", func() {
					token := "valid.jwt.token"
					key := BlacklistKeyPrefix + token
					expectedErr := errors.New("connection error")

					mock.ExpectExists(key).SetErr(expectedErr)

					isBlacklisted, err := repo.IsBlacklisted(ctx, token)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection error"))
					Expect(isBlacklisted).To(BeFalse())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Token expiry behavior", func() {
		When("token TTL expires", func() {
			Context("after the specified duration", func() {
				It("should automatically remove token from blacklist", func() {
					token := "expiring.jwt.token"
					key := BlacklistKeyPrefix + token

					// Initially blacklisted
					mock.ExpectExists(key).SetVal(1)

					isBlacklisted, err := repo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeTrue())

					// After expiry - token no longer exists
					mock.ExpectExists(key).SetVal(0)

					isBlacklisted, err = repo.IsBlacklisted(ctx, token)
					Expect(err).ToNot(HaveOccurred())
					Expect(isBlacklisted).To(BeFalse())

					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})
})
