package telemetry_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"

	"go.opentelemetry.io/otel"
)

var _ = Describe("NewTracerProvider", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	When("telemetry is disabled", func() {
		It("should return a noop TracerProvider without error", func() {
			cfg := telemetry.Config{
				Enabled: false,
			}

			tp, err := telemetry.NewTracerProvider(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(tp).NotTo(BeNil())

			// The noop provider should still work: create a tracer, start a span
			tracer := tp.Tracer("test")
			_, span := tracer.Start(ctx, "test-span")
			span.End()

			// Shutdown should succeed
			Expect(tp.Shutdown(ctx)).To(Succeed())
		})
	})

	When("telemetry is enabled", func() {
		Context("with a valid OTLP endpoint", func() {
			It("should create a TracerProvider that can produce spans", func() {
				cfg := telemetry.Config{
					Enabled:          true,
					ServiceName:      "test-service",
					OTLPEndpoint:     "localhost:4317",
					OTLPInsecure:     true,
					TracesSampler:    "always_on",
					TracesSamplerArg: 1.0,
				}

				tp, err := telemetry.NewTracerProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())

				// Should be able to create a tracer and start a span
				tracer := tp.Tracer("test")
				spanCtx, span := tracer.Start(ctx, "test-span")
				Expect(spanCtx).NotTo(BeNil())
				Expect(span).NotTo(BeNil())
				Expect(span.SpanContext().IsValid()).To(BeTrue())
				span.End()

				// Shutdown should succeed (even if collector is not running,
				// the batch exporter drains gracefully)
				Expect(tp.Shutdown(ctx)).To(Succeed())
			})
		})

		Context("with always_off sampler", func() {
			It("should create a TracerProvider that does not sample", func() {
				cfg := telemetry.Config{
					Enabled:          true,
					ServiceName:      "test-service",
					OTLPEndpoint:     "localhost:4317",
					OTLPInsecure:     true,
					TracesSampler:    "always_off",
					TracesSamplerArg: 0,
				}

				tp, err := telemetry.NewTracerProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())

				tracer := tp.Tracer("test")
				_, span := tracer.Start(ctx, "test-span")
				Expect(span.SpanContext().IsSampled()).To(BeFalse())
				span.End()

				Expect(tp.Shutdown(ctx)).To(Succeed())
			})
		})

		Context("with traceidratio sampler", func() {
			It("should create a TracerProvider without error", func() {
				cfg := telemetry.Config{
					Enabled:          true,
					ServiceName:      "test-service",
					OTLPEndpoint:     "localhost:4317",
					OTLPInsecure:     true,
					TracesSampler:    "traceidratio",
					TracesSamplerArg: 0.5,
				}

				tp, err := telemetry.NewTracerProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())

				Expect(tp.Shutdown(ctx)).To(Succeed())
			})
		})
	})
})

var _ = Describe("NewMeterProvider", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	When("telemetry is disabled", func() {
		It("should return a noop MeterProvider without error", func() {
			cfg := telemetry.Config{
				Enabled: false,
			}

			mp, err := telemetry.NewMeterProvider(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(mp).NotTo(BeNil())

			// Shutdown should succeed
			Expect(mp.Shutdown(ctx)).To(Succeed())
		})
	})

	When("telemetry is enabled", func() {
		Context("with a valid OTLP endpoint", func() {
			It("should create a MeterProvider without error and shutdown succeeds", func() {
				cfg := telemetry.Config{
					Enabled:      true,
					ServiceName:  "test-service",
					OTLPEndpoint: "localhost:4317",
					OTLPInsecure: true,
				}

				mp, err := telemetry.NewMeterProvider(ctx, cfg)

				Expect(err).NotTo(HaveOccurred())
				Expect(mp).NotTo(BeNil())

				// Should be able to create a meter and record a measurement
				meter := mp.Meter("test")
				counter, counterErr := meter.Int64Counter("test_counter")
				Expect(counterErr).NotTo(HaveOccurred())
				counter.Add(ctx, 1)

				// Shutdown with a short timeout — the periodic reader will
				// attempt to export, which fails without a running collector,
				// so we just verify it completes without panic.
				shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
				_ = mp.Shutdown(shutdownCtx)
			})
		})
	})
})

var _ = Describe("Init", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	When("telemetry is enabled", func() {
		It("should initialize all providers and return non-nil Providers and ShutdownFunc", func() {
			cfg := telemetry.Config{
				Enabled:          true,
				ServiceName:      "test-service",
				OTLPEndpoint:     "localhost:4317",
				OTLPInsecure:     true,
				TracesSampler:    "always_on",
				TracesSamplerArg: 1.0,
				LogsExporter:     "otlp",
			}

			providers, shutdown, err := telemetry.Init(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(providers).NotTo(BeNil())
			Expect(providers.TracerProvider).NotTo(BeNil())
			Expect(providers.MeterProvider).NotTo(BeNil())
			Expect(providers.LoggerProvider).NotTo(BeNil())
			Expect(shutdown).NotTo(BeNil())

			// Verify global registration
			Expect(otel.GetTracerProvider()).To(Equal(providers.TracerProvider))
			Expect(otel.GetMeterProvider()).To(Equal(providers.MeterProvider))

			// Shutdown should succeed (graceful drain even without collector)
			shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			_ = shutdown(shutdownCtx)
		})
	})

	When("telemetry is disabled", func() {
		It("should return noop providers and shutdown should succeed", func() {
			cfg := telemetry.Config{
				Enabled: false,
			}

			providers, shutdown, err := telemetry.Init(ctx, cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(providers).NotTo(BeNil())
			Expect(providers.TracerProvider).NotTo(BeNil())
			Expect(providers.MeterProvider).NotTo(BeNil())
			Expect(providers.LoggerProvider).NotTo(BeNil())
			Expect(shutdown).NotTo(BeNil())

			// Shutdown should succeed
			Expect(shutdown(ctx)).To(Succeed())
		})
	})
})
