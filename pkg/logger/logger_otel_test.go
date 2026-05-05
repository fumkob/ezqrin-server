package logger_test

import (
	"github.com/fumkob/ezqrin-server/pkg/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Logger", func() {
	When("using WithOTelCore", func() {
		Context("with an observer core", func() {
			It("should tee log output to the additional core", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "console",
					Environment: "development",
				}
				log, err := logger.New(cfg)
				Expect(err).NotTo(HaveOccurred())

				observedCore, logs := observer.New(zapcore.InfoLevel)
				newLog := log.WithOTelCore(observedCore)

				newLog.Info("test message")

				Expect(logs.Len()).To(Equal(1))
				Expect(logs.All()[0].Message).To(Equal("test message"))
			})

			It("should not affect the original logger", func() {
				cfg := logger.Config{
					Level:       "info",
					Format:      "console",
					Environment: "development",
				}
				log, err := logger.New(cfg)
				Expect(err).NotTo(HaveOccurred())

				observedCore, logs := observer.New(zapcore.InfoLevel)
				_ = log.WithOTelCore(observedCore)

				log.Info("original logger message")

				Expect(logs.Len()).To(Equal(0))
			})

			It("should forward messages at the appropriate level", func() {
				cfg := logger.Config{
					Level:       "debug",
					Format:      "console",
					Environment: "development",
				}
				log, err := logger.New(cfg)
				Expect(err).NotTo(HaveOccurred())

				observedCore, logs := observer.New(zapcore.WarnLevel)
				newLog := log.WithOTelCore(observedCore)

				newLog.Info("info message")
				newLog.Warn("warn message")
				newLog.Error("error message")

				// Only warn and error should be captured by the observer core
				Expect(logs.Len()).To(Equal(2))
				Expect(logs.All()[0].Message).To(Equal("warn message"))
				Expect(logs.All()[1].Message).To(Equal("error message"))
			})
		})
	})
})
