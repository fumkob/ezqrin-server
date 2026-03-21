//go:build integration
// +build integration

package database_test

import (
	"context"
	"os"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PostgresDB", func() {
	var (
		ctx context.Context
		log *logger.Logger
		cfg *config.DatabaseConfig
	)

	BeforeEach(func() {
		ctx = context.Background()
		log, _ = logger.New(logger.Config{
			Level:       "info",
			Format:      "console",
			Environment: "development",
		})
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		cfg = &config.DatabaseConfig{
			Host:            dbHost,
			Port:            5432,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            "ezqrin_test",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		}
	})

	When("creating database connection", func() {
		Context("with nil config", func() {
			It("should return validation error", func() {
				db, err := database.NewPostgresDB(ctx, nil, log)

				Expect(db).To(BeNil())
				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsValidation(err)).To(BeTrue())
			})
		})

		Context("with nil logger", func() {
			It("should return validation error", func() {
				db, err := database.NewPostgresDB(ctx, cfg, nil)

				Expect(db).To(BeNil())
				Expect(err).NotTo(BeNil())
				Expect(apperrors.IsValidation(err)).To(BeTrue())
			})
		})

		// Integration test - requires actual PostgreSQL
		Context("with valid config and running database", func() {
			It("should create connection pool successfully", func() {
				db, err := database.NewPostgresDB(ctx, cfg, log)

				Expect(err).To(BeNil())
				Expect(db).NotTo(BeNil())
				Expect(db.GetPool()).NotTo(BeNil())

				// Cleanup
				db.Close()
			})
		})

		Context("with invalid connection string", func() {
			It("should return error for invalid port", func() {
				invalidCfg := *cfg
				invalidCfg.Port = 99999 // Invalid port

				// This will parse successfully but fail on connection
				db, err := database.NewPostgresDB(ctx, &invalidCfg, log)

				// Connection will fail during Ping
				Expect(db).To(BeNil())
				Expect(err).NotTo(BeNil())
			})
		})
	})

	When("accessing connection pool", func() {
		// Integration test
		Describe("with established connection", func() {
			var db *database.PostgresDB

			BeforeEach(func() {
				var err error
				db, err = database.NewPostgresDB(ctx, cfg, log)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				if db != nil {
					db.Close()
				}
			})

			It("should return non-nil pool", func() {
				pool := db.GetPool()

				Expect(pool).NotTo(BeNil())
			})

			It("should successfully ping database", func() {
				err := db.Ping(ctx)

				Expect(err).To(BeNil())
			})
		})
	})

	When("closing database connection", func() {
		Context("with nil pool", func() {
			It("should not panic", func() {
				db := &database.PostgresDB{}

				Expect(func() {
					db.Close()
				}).NotTo(Panic())
			})
		})

		// Integration test
		Context("with active connection", func() {
			It("should close gracefully", func() {
				db, err := database.NewPostgresDB(ctx, cfg, log)
				Expect(err).To(BeNil())

				Expect(func() {
					db.Close()
				}).NotTo(Panic())
			})
		})
	})
})

var _ = Describe("HealthChecker", func() {
	var (
		ctx context.Context
		log *logger.Logger
		cfg *config.DatabaseConfig
	)

	BeforeEach(func() {
		ctx = context.Background()
		log, _ = logger.New(logger.Config{
			Level:       "info",
			Format:      "console",
			Environment: "development",
		})
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		cfg = &config.DatabaseConfig{
			Host:            dbHost,
			Port:            5432,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            "ezqrin_test",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		}
	})

	// Integration tests
	Describe("health check operations", func() {
		var db *database.PostgresDB

		BeforeEach(func() {
			var err error
			db, err = database.NewPostgresDB(ctx, cfg, log)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			if db != nil {
				db.Close()
			}
		})

		When("checking database health", func() {
			Context("with healthy database", func() {
				It("should return healthy status", func() {
					status, err := db.CheckHealth(ctx)

					Expect(err).To(BeNil())
					Expect(status).NotTo(BeNil())
					Expect(status.Healthy).To(BeTrue())
					Expect(status.ResponseTime).To(BeNumerically(">", 0))
					Expect(status.MaxConns).To(Equal(int32(25)))
					Expect(status.Error).To(BeEmpty())
				})

				It("should include timestamp", func() {
					now := time.Now()
					status, err := db.CheckHealth(ctx)

					Expect(err).To(BeNil())
					Expect(status.Timestamp).To(BeTemporally("~", now, time.Second))
				})
			})

			Context("with cancelled context", func() {
				It("should return error", func() {
					cancelCtx, cancel := context.WithCancel(ctx)
					cancel() // Cancel immediately

					status, err := db.CheckHealth(cancelCtx)

					Expect(err).NotTo(BeNil())
					Expect(status).NotTo(BeNil())
					Expect(status.Healthy).To(BeFalse())
				})
			})
		})

		When("checking if database is healthy", func() {
			Context("with healthy database", func() {
				It("should return true", func() {
					healthy := db.IsHealthy(ctx)

					Expect(healthy).To(BeTrue())
				})
			})
		})

		When("getting pool statistics", func() {
			Context("without ping", func() {
				It("should return current stats", func() {
					stats := db.GetPoolStats()

					Expect(stats).NotTo(BeNil())
					Expect(stats.MaxConns).To(Equal(int32(25)))
					Expect(stats.TotalConns).To(BeNumerically(">=", 0))
					Expect(stats.IdleConns).To(BeNumerically(">=", 0))
				})

				It("should include timestamp", func() {
					now := time.Now()
					stats := db.GetPoolStats()

					Expect(stats.Timestamp).To(BeTemporally("~", now, time.Second))
				})
			})
		})

		When("waiting for database to become healthy", func() {
			Context("with already healthy database", func() {
				It("should return immediately", func() {
					start := time.Now()
					err := db.WaitForHealthy(ctx, 100*time.Millisecond)
					elapsed := time.Since(start)

					Expect(err).To(BeNil())
					// Should complete in less than 500ms (not wait for retry)
					Expect(elapsed).To(BeNumerically("<", 500*time.Millisecond))
				})
			})

			Context("with timeout", func() {
				It("should return error when context expires", func() {
					timeoutCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
					defer cancel()

					err := db.WaitForHealthy(timeoutCtx, 500*time.Millisecond)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("Transaction Management", func() {
	var (
		ctx context.Context
		log *logger.Logger
		cfg *config.DatabaseConfig
	)

	BeforeEach(func() {
		ctx = context.Background()
		log, _ = logger.New(logger.Config{
			Level:       "info",
			Format:      "console",
			Environment: "development",
		})
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		cfg = &config.DatabaseConfig{
			Host:            dbHost,
			Port:            5432,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            "ezqrin_test",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		}
	})

	When("managing transactions", func() {
		Context("with GetTx", func() {
			It("should return nil when no transaction in context", func() {
				tx := database.GetTx(ctx)

				Expect(tx).To(BeNil())
			})
		})

		// Integration tests
		Describe("transaction operations", func() {
			var db *database.PostgresDB

			BeforeEach(func() {
				var err error
				db, err = database.NewPostgresDB(ctx, cfg, log)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				if db != nil {
					db.Close()
				}
			})

			Context("with WithTransaction", func() {
				It("should commit on success", func() {
					executed := false
					err := database.WithTransaction(ctx, db.GetPool(), func(txCtx context.Context) error {
						executed = true
						tx := database.GetTx(txCtx)
						Expect(tx).NotTo(BeNil())
						return nil
					})

					Expect(err).To(BeNil())
					Expect(executed).To(BeTrue())
				})

				It("should rollback on error", func() {
					expectedErr := apperrors.Validation("test error")
					err := database.WithTransaction(ctx, db.GetPool(), func(txCtx context.Context) error {
						return expectedErr
					})

					Expect(err).To(Equal(expectedErr))
				})

				It("should rollback on panic", func() {
					Expect(func() {
						_ = database.WithTransaction(ctx, db.GetPool(), func(txCtx context.Context) error {
							panic("test panic")
						})
					}).To(Panic())
				})
			})

			Context("with GetQueryable", func() {
				It("should return pool when no transaction", func() {
					q := database.GetQueryable(ctx, db.GetPool())

					Expect(q).To(Equal(db.GetPool()))
				})

				It("should return transaction when in transaction context", func() {
					err := database.WithTransaction(ctx, db.GetPool(), func(txCtx context.Context) error {
						q := database.GetQueryable(txCtx, db.GetPool())
						tx := database.GetTx(txCtx)

						Expect(q).To(Equal(tx))
						Expect(q).NotTo(Equal(db.GetPool()))
						return nil
					})

					Expect(err).To(BeNil())
				})
			})
		})
	})
})
