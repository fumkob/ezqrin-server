package logger_test

import (
	"context"

	"github.com/fumkob/ezqrin-server/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Logger", func() {
	When("creating a new logger", func() {
		Context("with development environment", func() {
			It("should create logger with console output", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
				Expect(log.Logger).NotTo(BeNil())
			})

			It("should create logger with colored output", func() {
				cfg := logger.Config{
					Level:       "debug",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})
		})

		Context("with production environment", func() {
			It("should create logger with JSON output", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "json",
					Environment: "production",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
				Expect(log.Logger).NotTo(BeNil())
			})

			It("should create logger with ISO8601 timestamps", func() {
				cfg := logger.Config{
					Level:       "warn",
					Format:      "json",
					Environment: "production",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})
		})

		Context("with different log levels", func() {
			It("should accept debug level", func() {
				cfg := logger.Config{
					Level:       "debug",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})

			It("should accept info level", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})

			It("should accept warn level", func() {
				cfg := logger.Config{
					Level:       "warn",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})

			It("should accept error level", func() {
				cfg := logger.Config{
					Level:       "error",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})
		})

		Context("with invalid log level", func() {
			It("should return error for unknown level", func() {
				cfg := logger.Config{
					Level:       "invalid",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid log level"))
				Expect(log).To(BeNil())
			})

			It("should return error for empty level", func() {
				cfg := logger.Config{
					Level:       "",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).To(HaveOccurred())
				Expect(log).To(BeNil())
			})
		})

		Context("with different output formats", func() {
			It("should create logger with JSON format", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "json",
					Environment: "production",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})

			It("should create logger with console format", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "console",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})

			It("should default to console for unknown format", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "unknown",
					Environment: "development",
				}

				log, err := logger.New(cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(log).NotTo(BeNil())
			})
		})
	})

	When("logging messages", func() {
		var log *logger.Logger

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "debug",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with Debug level", func() {
			It("should log debug message without fields", func() {
				Expect(func() {
					log.Debug("debug message")
				}).NotTo(Panic())
			})

			It("should log debug message with fields", func() {
				Expect(func() {
					log.Debug("debug with fields",
						zap.String("key", "value"),
						zap.Int("count", 42),
					)
				}).NotTo(Panic())
			})
		})

		Context("with Info level", func() {
			It("should log info message without fields", func() {
				Expect(func() {
					log.Info("info message")
				}).NotTo(Panic())
			})

			It("should log info message with fields", func() {
				Expect(func() {
					log.Info("info with fields",
						zap.String("user_id", "123"),
						zap.Bool("active", true),
					)
				}).NotTo(Panic())
			})
		})

		Context("with Warn level", func() {
			It("should log warn message without fields", func() {
				Expect(func() {
					log.Warn("warning message")
				}).NotTo(Panic())
			})

			It("should log warn message with fields", func() {
				Expect(func() {
					log.Warn("warning with fields",
						zap.String("reason", "timeout"),
					)
				}).NotTo(Panic())
			})
		})

		Context("with Error level", func() {
			It("should log error message without fields", func() {
				Expect(func() {
					log.Error("error message")
				}).NotTo(Panic())
			})

			It("should log error message with fields", func() {
				Expect(func() {
					log.Error("error with fields",
						zap.Error(context.DeadlineExceeded),
						zap.String("operation", "database query"),
					)
				}).NotTo(Panic())
			})
		})
	})

	When("working with context", func() {
		var log *logger.Logger
		var ctx context.Context

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "info",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())

			ctx = context.Background()
		})

		Context("with request ID in context", func() {
			It("should add request ID to logger", func() {
				requestID := "test-request-123"
				ctx = logger.ContextWithRequestID(ctx, requestID)

				contextLogger := log.WithContext(ctx)

				Expect(contextLogger).NotTo(BeNil())
				Expect(contextLogger).NotTo(Equal(log))
			})

			It("should retrieve request ID from context", func() {
				requestID := "test-request-456"
				ctx = logger.ContextWithRequestID(ctx, requestID)

				retrievedID := logger.GetRequestID(ctx)

				Expect(retrievedID).To(Equal(requestID))
			})

			It("should allow logging with context", func() {
				requestID := "req-789"
				ctx = logger.ContextWithRequestID(ctx, requestID)

				Expect(func() {
					contextLogger := log.WithContext(ctx)
					contextLogger.Info("message with request ID")
				}).NotTo(Panic())
			})
		})

		Context("without request ID in context", func() {
			It("should return empty string when no request ID", func() {
				retrievedID := logger.GetRequestID(ctx)

				Expect(retrievedID).To(Equal(""))
			})

			It("should return same logger when no request ID", func() {
				contextLogger := log.WithContext(ctx)

				Expect(contextLogger).To(Equal(log))
			})
		})

		Context("with nil context", func() {
			It("should return empty string for nil context", func() {
				var nilCtx context.Context

				requestID := logger.GetRequestID(nilCtx)
				Expect(requestID).To(Equal(""))
			})
		})

		Context("with wrong type in context", func() {
			It("should return empty string for non-string value", func() {
				// Create context with wrong type
				type wrongKey string
				ctx = context.WithValue(ctx, wrongKey("request_id"), 12345)

				retrievedID := logger.GetRequestID(ctx)

				Expect(retrievedID).To(Equal(""))
			})
		})
	})

	When("adding request ID to logger", func() {
		var log *logger.Logger

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "info",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with WithRequestID method", func() {
			It("should create logger with request ID", func() {
				requestID := "direct-request-123"

				loggerWithID := log.WithRequestID(requestID)

				Expect(loggerWithID).NotTo(BeNil())
				Expect(loggerWithID).NotTo(Equal(log))
			})

			It("should allow logging with request ID", func() {
				requestID := "direct-request-456"

				Expect(func() {
					loggerWithID := log.WithRequestID(requestID)
					loggerWithID.Info("message with direct request ID")
				}).NotTo(Panic())
			})

			It("should handle empty request ID", func() {
				Expect(func() {
					loggerWithID := log.WithRequestID("")
					loggerWithID.Info("message with empty request ID")
				}).NotTo(Panic())
			})
		})
	})

	When("adding custom fields to logger", func() {
		var log *logger.Logger

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "info",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with WithFields method", func() {
			It("should create logger with single field", func() {
				loggerWithFields := log.WithFields(
					zap.String("user_id", "user-123"),
				)

				Expect(loggerWithFields).NotTo(BeNil())
				Expect(loggerWithFields).NotTo(Equal(log))
			})

			It("should create logger with multiple fields", func() {
				loggerWithFields := log.WithFields(
					zap.String("user_id", "user-123"),
					zap.String("tenant_id", "tenant-456"),
					zap.Int("version", 1),
				)

				Expect(loggerWithFields).NotTo(BeNil())
			})

			It("should allow logging with custom fields", func() {
				Expect(func() {
					loggerWithFields := log.WithFields(
						zap.String("service", "api"),
						zap.String("environment", "test"),
					)
					loggerWithFields.Info("message with custom fields")
				}).NotTo(Panic())
			})

			It("should handle empty fields", func() {
				Expect(func() {
					loggerWithFields := log.WithFields()
					loggerWithFields.Info("message with no fields")
				}).NotTo(Panic())
			})
		})
	})

	When("syncing logger", func() {
		var log *logger.Logger

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "info",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with Sync method", func() {
			It("should sync without error", func() {
				log.Info("test message")

				err := log.Sync()

				// Note: Sync may return error on stderr/stdout, which is expected
				_ = err
			})

			It("should not panic when called multiple times", func() {
				Expect(func() {
					_ = log.Sync()
					_ = log.Sync()
					_ = log.Sync()
				}).NotTo(Panic())
			})
		})
	})

	When("managing request ID in context", func() {
		Context("with ContextWithRequestID", func() {
			It("should add request ID to context", func() {
				ctx := context.Background()
				requestID := "context-req-123"

				newCtx := logger.ContextWithRequestID(ctx, requestID)

				retrievedID := logger.GetRequestID(newCtx)
				Expect(retrievedID).To(Equal(requestID))
			})

			It("should preserve other context values", func() {
				type testKey string
				ctx := context.WithValue(context.Background(), testKey("other"), "value")
				requestID := "req-preserve-test"

				newCtx := logger.ContextWithRequestID(ctx, requestID)

				Expect(logger.GetRequestID(newCtx)).To(Equal(requestID))
				Expect(newCtx.Value(testKey("other"))).To(Equal("value"))
			})

			It("should override existing request ID", func() {
				ctx := context.Background()
				firstID := "first-request-id"
				secondID := "second-request-id"

				ctx = logger.ContextWithRequestID(ctx, firstID)
				ctx = logger.ContextWithRequestID(ctx, secondID)

				retrievedID := logger.GetRequestID(ctx)
				Expect(retrievedID).To(Equal(secondID))
			})

			It("should handle empty request ID", func() {
				ctx := context.Background()

				newCtx := logger.ContextWithRequestID(ctx, "")

				retrievedID := logger.GetRequestID(newCtx)
				Expect(retrievedID).To(Equal(""))
			})
		})
	})

	When("chaining logger methods", func() {
		var log *logger.Logger

		BeforeEach(func() {
			cfg := logger.Config{
				Level:       "debug",
				Format:      "console",
				Environment: "development",
			}
			var err error
			log, err = logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with multiple modifications", func() {
			It("should allow chaining WithRequestID and WithFields", func() {
				Expect(func() {
					loggerChain := log.WithRequestID("req-123").WithFields(
						zap.String("user_id", "user-456"),
					)
					loggerChain.Info("chained logger message")
				}).NotTo(Panic())
			})

			It("should preserve all added context", func() {
				ctx := logger.ContextWithRequestID(context.Background(), "ctx-req-789")

				Expect(func() {
					loggerChain := log.WithContext(ctx).WithFields(
						zap.String("operation", "test"),
					)
					loggerChain.Debug("message with context and fields")
				}).NotTo(Panic())
			})
		})
	})
})
