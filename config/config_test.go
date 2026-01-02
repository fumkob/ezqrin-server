package config_test

import (
	"os"
	"testing"

	"github.com/fumkob/ezqrin-server/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	var originalEnv map[string]string

	BeforeEach(func() {
		// Reset viper to ensure clean state
		viper.Reset()

		// Save original environment variables
		originalEnv = make(map[string]string)
		envVars := []string{
			"SERVER_PORT", "SERVER_ENV",
			"SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT", "SERVER_IDLE_TIMEOUT",
			"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE",
			"DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME", "DB_MAX_CONN_IDLE_TIME",
			"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
			"JWT_SECRET", "JWT_ACCESS_TOKEN_EXPIRY", "JWT_REFRESH_TOKEN_EXPIRY_WEB", "JWT_REFRESH_TOKEN_EXPIRY_MOBILE",
			"LOG_LEVEL", "LOG_FORMAT",
			"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS", "CORS_ALLOW_CREDENTIALS",
		}
		for _, key := range envVars {
			originalEnv[key] = os.Getenv(key)
			_ = os.Unsetenv(key)
		}
	})

	AfterEach(func() {
		// Restore original environment variables
		for key, value := range originalEnv {
			if value != "" {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		}

		// Reset viper again for good measure
		viper.Reset()
	})

	Describe("Load", func() {
		Context("with all required environment variables set", func() {
			BeforeEach(func() {
				_ = os.Setenv("DB_USER", "testuser")
				_ = os.Setenv("DB_PASSWORD", "testpass")
				_ = os.Setenv("DB_NAME", "testdb")
				_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")
			})

			It("should load configuration successfully", func() {
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg).ToNot(BeNil())
			})

			It("should use default values from YAML files", func() {
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())

				// Values from default.yaml
				Expect(cfg.Server.Port).To(Equal(8080))
				Expect(cfg.Server.Environment).To(Equal("development")) // From development.yaml (default env)
				Expect(cfg.Database.Host).To(Equal("postgres"))         // From development.yaml (DevContainer)
				Expect(cfg.Database.Port).To(Equal(5432))
				Expect(cfg.Database.SSLMode).To(Equal("disable"))
				Expect(cfg.Redis.Host).To(Equal("redis")) // From development.yaml (DevContainer)
				Expect(cfg.Redis.Port).To(Equal(6379))
				Expect(cfg.Logging.Level).To(Equal("debug")) // From development.yaml
				Expect(cfg.Logging.Format).To(Equal("text")) // From development.yaml
			})
		})

		Context("with custom values", func() {
			BeforeEach(func() {
				_ = os.Setenv("SERVER_PORT", "9000")
				_ = os.Setenv("SERVER_ENV", "production")
				_ = os.Setenv("DB_HOST", "db.example.com")
				_ = os.Setenv("DB_PORT", "5433")
				_ = os.Setenv("DB_USER", "produser")
				_ = os.Setenv("DB_PASSWORD", "prodpass")
				_ = os.Setenv("DB_NAME", "proddb")
				_ = os.Setenv("DB_SSL_MODE", "require")
				_ = os.Setenv("DB_MAX_CONNS", "50")
				_ = os.Setenv("DB_MIN_CONNS", "10")
				_ = os.Setenv("DB_MAX_CONN_LIFETIME", "10m")
				_ = os.Setenv("DB_MAX_CONN_IDLE_TIME", "5m")
				_ = os.Setenv("REDIS_HOST", "redis.example.com")
				_ = os.Setenv("REDIS_PORT", "6380")
				_ = os.Setenv("REDIS_PASSWORD", "redispass")
				_ = os.Setenv("REDIS_DB", "1")
				_ = os.Setenv("JWT_SECRET", "production-secret-key-very-long-and-secure-string-here")
				_ = os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "30m")
				_ = os.Setenv("JWT_REFRESH_TOKEN_EXPIRY_WEB", "336h")
				_ = os.Setenv("JWT_REFRESH_TOKEN_EXPIRY_MOBILE", "4320h")
				_ = os.Setenv("LOG_LEVEL", "warn")
				_ = os.Setenv("LOG_FORMAT", "text")
			})

			It("should load all custom values correctly", func() {
				cfg, err := config.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(cfg.Server.Port).To(Equal(9000))
				Expect(cfg.Server.Environment).To(Equal("production"))
				Expect(cfg.Database.Host).To(Equal("db.example.com"))
				Expect(cfg.Database.Port).To(Equal(5433))
				Expect(cfg.Database.User).To(Equal("produser"))
				Expect(cfg.Database.Password).To(Equal("prodpass"))
				Expect(cfg.Database.Name).To(Equal("proddb"))
				Expect(cfg.Database.SSLMode).To(Equal("require"))
				Expect(cfg.Database.MaxConns).To(Equal(50))
				Expect(cfg.Database.MinConns).To(Equal(10))
				Expect(cfg.Redis.Host).To(Equal("redis.example.com"))
				Expect(cfg.Redis.Port).To(Equal(6380))
				Expect(cfg.Redis.Password).To(Equal("redispass"))
				Expect(cfg.Redis.DB).To(Equal(1))
				Expect(cfg.Logging.Level).To(Equal("warn"))
				Expect(cfg.Logging.Format).To(Equal("text"))
			})
		})

		Context("with missing required environment variables", func() {
			When("DB_USER is missing", func() {
				BeforeEach(func() {
					_ = os.Setenv("DB_PASSWORD", "testpass")
					_ = os.Setenv("DB_NAME", "testdb")
					_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("database user is required"))
				})
			})

			When("DB_PASSWORD is missing", func() {
				BeforeEach(func() {
					_ = os.Setenv("DB_USER", "testuser")
					_ = os.Setenv("DB_NAME", "testdb")
					_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("database password is required"))
				})
			})

			When("DB_NAME is missing", func() {
				BeforeEach(func() {
					_ = os.Setenv("DB_USER", "testuser")
					_ = os.Setenv("DB_PASSWORD", "testpass")
					_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("database name is required"))
				})
			})

			When("JWT_SECRET is missing", func() {
				BeforeEach(func() {
					_ = os.Setenv("DB_USER", "testuser")
					_ = os.Setenv("DB_PASSWORD", "testpass")
					_ = os.Setenv("DB_NAME", "testdb")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("jwt secret is required"))
				})
			})
		})

		Context("with invalid values", func() {
			BeforeEach(func() {
				_ = os.Setenv("DB_USER", "testuser")
				_ = os.Setenv("DB_PASSWORD", "testpass")
				_ = os.Setenv("DB_NAME", "testdb")
				_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")
			})

			When("SERVER_PORT is invalid", func() {
				BeforeEach(func() {
					_ = os.Setenv("SERVER_PORT", "invalid")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("server port must be between 1 and 65535"))
				})
			})

			When("DB_PORT is invalid", func() {
				BeforeEach(func() {
					_ = os.Setenv("DB_PORT", "not-a-number")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("database port must be between 1 and 65535"))
				})
			})

			When("JWT_ACCESS_TOKEN_EXPIRY is invalid", func() {
				BeforeEach(func() {
					_ = os.Setenv("JWT_ACCESS_TOKEN_EXPIRY", "invalid-duration")
				})

				It("should return an error", func() {
					_, err := config.Load()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("jwt access token expiry must be positive"))
				})
			})
		})
	})

	Describe("Validate", func() {
		var cfg *config.Config

		BeforeEach(func() {
			_ = os.Setenv("DB_USER", "testuser")
			_ = os.Setenv("DB_PASSWORD", "testpass")
			_ = os.Setenv("DB_NAME", "testdb")
			_ = os.Setenv("JWT_SECRET", "test-secret-key-with-at-least-32-characters-long")

			var err error
			cfg, err = config.Load()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with valid configuration", func() {
			It("should validate successfully", func() {
				err := cfg.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid server port", func() {
			It("should return validation error for port 0", func() {
				cfg.Server.Port = 0
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("server port must be between 1 and 65535"))
			})

			It("should return validation error for port > 65535", func() {
				cfg.Server.Port = 70000
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("server port must be between 1 and 65535"))
			})
		})

		Context("with invalid server environment", func() {
			It("should return validation error", func() {
				cfg.Server.Environment = "invalid"
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("server environment must be"))
			})
		})

		Context("with invalid JWT secret", func() {
			It("should return validation error for short secret", func() {
				cfg.JWT.Secret = "short"
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jwt secret must be at least 32 characters"))
			})

			It("should return validation error for empty secret", func() {
				cfg.JWT.Secret = ""
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jwt secret is required"))
			})
		})

		Context("with invalid log level", func() {
			It("should return validation error", func() {
				cfg.Logging.Level = "invalid"
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("log level must be one of"))
			})
		})

		Context("with invalid log format", func() {
			It("should return validation error", func() {
				cfg.Logging.Format = "invalid"
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("log format must be"))
			})
		})
	})

	Describe("Helper Methods", func() {
		var cfg *config.Config

		BeforeEach(func() {
			_ = os.Setenv("SERVER_ENV", "production")
			_ = os.Setenv("DB_HOST", "db.example.com")
			_ = os.Setenv("DB_PORT", "5433")
			_ = os.Setenv("DB_USER", "produser")
			_ = os.Setenv("DB_PASSWORD", "prodpass")
			_ = os.Setenv("DB_NAME", "proddb")
			_ = os.Setenv("DB_SSL_MODE", "require")
			_ = os.Setenv("REDIS_HOST", "redis.example.com")
			_ = os.Setenv("REDIS_PORT", "6380")
			_ = os.Setenv("JWT_SECRET", "production-secret-key-very-long-and-secure-string-here")

			var err error
			cfg, err = config.Load()
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("GetDatabaseDSN", func() {
			It("should return correct PostgreSQL connection string", func() {
				dsn := cfg.GetDatabaseDSN()
				Expect(
					dsn,
				).To(Equal("host=db.example.com port=5433 user=produser password=prodpass dbname=proddb sslmode=require"))
			})
		})

		Describe("GetRedisAddr", func() {
			It("should return correct Redis address", func() {
				addr := cfg.GetRedisAddr()
				Expect(addr).To(Equal("redis.example.com:6380"))
			})
		})

		Describe("IsProduction", func() {
			When("environment is production", func() {
				It("should return true", func() {
					Expect(cfg.IsProduction()).To(BeTrue())
				})
			})

			When("environment is development", func() {
				BeforeEach(func() {
					cfg.Server.Environment = "development"
				})

				It("should return false", func() {
					Expect(cfg.IsProduction()).To(BeFalse())
				})
			})
		})
	})
})
