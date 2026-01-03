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

var _ = Describe("CacheRepository", func() {
	var (
		mockClient *goredis.Client
		mock       redismock.ClientMock
		client     *Client
		repo       *CacheRepository
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient, mock = redismock.NewClientMock()
		client = newTestClient(mockClient)
		repo = NewCacheRepository(client)
	})

	AfterEach(func() {
		mock.ClearExpect()
	})

	Describe("Get", func() {
		When("retrieving an existing key", func() {
			Context("with valid key", func() {
				It("should return the value", func() {
					key := "test-key"
					expectedValue := "test-value"

					mock.ExpectGet(key).SetVal(expectedValue)

					value, err := repo.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(expectedValue))
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("retrieving a non-existent key", func() {
			Context("with key that doesn't exist", func() {
				It("should return empty string without error", func() {
					key := "non-existent"

					mock.ExpectGet(key).RedisNil()

					value, err := repo.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(BeEmpty())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("Redis returns an error", func() {
			Context("with connection failure", func() {
				It("should return the error", func() {
					key := "test-key"
					expectedErr := errors.New("connection error")

					mock.ExpectGet(key).SetErr(expectedErr)

					value, err := repo.Get(ctx, key)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection error"))
					Expect(value).To(BeEmpty())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Set", func() {
		When("setting a key-value pair", func() {
			Context("with TTL", func() {
				It("should set the value with expiration", func() {
					key := "test-key"
					value := "test-value"
					ttl := 5 * time.Minute

					mock.ExpectSet(key, value, ttl).SetVal("OK")

					err := repo.Set(ctx, key, value, ttl)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("without TTL (ttl = 0)", func() {
				It("should set the value without expiration", func() {
					key := "test-key"
					value := "test-value"
					ttl := time.Duration(0)

					mock.ExpectSet(key, value, ttl).SetVal("OK")

					err := repo.Set(ctx, key, value, ttl)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("Redis returns an error", func() {
			Context("with write failure", func() {
				It("should return the error", func() {
					key := "test-key"
					value := "test-value"
					ttl := 5 * time.Minute
					expectedErr := errors.New("write error")

					mock.ExpectSet(key, value, ttl).SetErr(expectedErr)

					err := repo.Set(ctx, key, value, ttl)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("write error"))
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Delete", func() {
		When("deleting an existing key", func() {
			Context("with valid key", func() {
				It("should delete the key successfully", func() {
					key := "test-key"

					mock.ExpectDel(key).SetVal(1)

					err := repo.Delete(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})

		When("deleting a non-existent key", func() {
			Context("with key that doesn't exist", func() {
				It("should not return an error", func() {
					key := "non-existent"

					mock.ExpectDel(key).SetVal(0)

					err := repo.Delete(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Exists", func() {
		When("checking key existence", func() {
			Context("with existing key", func() {
				It("should return true", func() {
					key := "test-key"

					mock.ExpectExists(key).SetVal(1)

					exists, err := repo.Exists(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with non-existent key", func() {
				It("should return false", func() {
					key := "non-existent"

					mock.ExpectExists(key).SetVal(0)

					exists, err := repo.Exists(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("MGet", func() {
		When("retrieving multiple keys", func() {
			Context("with all keys existing", func() {
				It("should return all values", func() {
					keys := []string{"key1", "key2", "key3"}
					values := []interface{}{"value1", "value2", "value3"}

					mock.ExpectMGet(keys...).SetVal(values)

					result, err := repo.MGet(ctx, keys)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(HaveLen(3))
					Expect(result["key1"]).To(Equal("value1"))
					Expect(result["key2"]).To(Equal("value2"))
					Expect(result["key3"]).To(Equal("value3"))
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with some keys missing", func() {
				It("should return only existing values", func() {
					keys := []string{"key1", "key2", "key3"}
					values := []interface{}{"value1", nil, "value3"}

					mock.ExpectMGet(keys...).SetVal(values)

					result, err := repo.MGet(ctx, keys)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(HaveLen(2))
					Expect(result["key1"]).To(Equal("value1"))
					Expect(result["key3"]).To(Equal("value3"))
					Expect(result["key2"]).To(BeEmpty()) // nil values not included
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with empty keys slice", func() {
				It("should return empty map", func() {
					keys := []string{}

					result, err := repo.MGet(ctx, keys)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeEmpty())
				})
			})
		})
	})

	Describe("MSet", func() {
		When("setting multiple key-value pairs", func() {
			Context("with valid items", func() {
				It("should set all values with TTL", func() {
					items := map[string]string{
						"key1": "value1",
					}
					ttl := 5 * time.Minute

					// Set up pipeline expectations
					mock.MatchExpectationsInOrder(false)
					for key, value := range items {
						mock.ExpectSet(key, value, ttl).SetVal("OK")
					}

					err := repo.MSet(ctx, items, ttl)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with empty items map", func() {
				It("should not perform any Redis operations", func() {
					items := map[string]string{}

					err := repo.MSet(ctx, items, 5*time.Minute)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Ping", func() {
		When("checking Redis connectivity", func() {
			Context("with healthy connection", func() {
				It("should return no error", func() {
					mock.ExpectPing().SetVal("PONG")

					err := repo.Ping(ctx)
					Expect(err).ToNot(HaveOccurred())
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})

			Context("with connection failure", func() {
				It("should return an error", func() {
					expectedErr := errors.New("connection refused")

					mock.ExpectPing().SetErr(expectedErr)

					err := repo.Ping(ctx)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection refused"))
					Expect(mock.ExpectationsWereMet()).ToNot(HaveOccurred())
				})
			})
		})
	})
})
