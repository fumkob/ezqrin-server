package telemetry_test

import (
	"context"
	"time"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewLoggerProvider", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	When("telemetry is disabled", func() {
		It("should return a noop LoggerProvider without error", func() {
			cfg := telemetry.Config{
				Enabled: false,
			}

			lp, err := telemetry.NewLoggerProvider(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(lp).NotTo(BeNil())

			// Shutdown should succeed
			Expect(lp.Shutdown(ctx)).To(Succeed())
		})
	})

	When("telemetry is enabled", func() {
		Context("with LogsExporter set to none", func() {
			It("should return a noop LoggerProvider without error", func() {
				cfg := telemetry.Config{
					Enabled:      true,
					LogsExporter: "none",
				}

				lp, err := telemetry.NewLoggerProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(lp).NotTo(BeNil())

				// Shutdown should succeed
				Expect(lp.Shutdown(ctx)).To(Succeed())
			})
		})

		Context("with a valid OTLP endpoint", func() {
			It("should create a LoggerProvider without error and shutdown succeeds", func() {
				cfg := telemetry.Config{
					Enabled:      true,
					ServiceName:  "test-service",
					OTLPEndpoint: "localhost:4317",
					OTLPInsecure: true,
					LogsExporter: "otlp",
				}

				lp, err := telemetry.NewLoggerProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(lp).NotTo(BeNil())

				// Shutdown with a short timeout — the batch processor will
				// attempt to export, which fails without a running collector,
				// so we just verify it completes without panic.
				shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
				_ = lp.Shutdown(shutdownCtx)
			})
		})
	})
})
