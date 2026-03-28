package telemetry_test

import (
	"os"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("NewConfigFromEnv", func() {
		// Save and restore all OTEL env vars around each test
		var savedEnv map[string]string
		envKeys := []string{
			"OTEL_ENABLED",
			"OTEL_SERVICE_NAME",
			"OTEL_EXPORTER_OTLP_ENDPOINT",
			"OTEL_EXPORTER_OTLP_INSECURE",
			"OTEL_TRACES_SAMPLER",
			"OTEL_TRACES_SAMPLER_ARG",
			"OTEL_LOGS_EXPORTER",
		}

		BeforeEach(func() {
			savedEnv = make(map[string]string)
			for _, key := range envKeys {
				savedEnv[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
		})

		AfterEach(func() {
			for _, key := range envKeys {
				if v, ok := savedEnv[key]; ok && v != "" {
					os.Setenv(key, v)
				} else {
					os.Unsetenv(key)
				}
			}
		})

		When("no environment variables are set", func() {
			It("should return default values", func() {
				cfg := telemetry.NewConfigFromEnv()

				Expect(cfg.Enabled).To(BeTrue())
				Expect(cfg.ServiceName).To(Equal("ezqrin-server"))
				Expect(cfg.OTLPEndpoint).To(Equal("localhost:4317"))
				Expect(cfg.OTLPInsecure).To(BeTrue())
				Expect(cfg.TracesSampler).To(Equal("always_on"))
				Expect(cfg.TracesSamplerArg).To(Equal(1.0))
				Expect(cfg.LogsExporter).To(Equal("otlp"))
			})
		})

		When("OTEL_ENABLED is set to false", func() {
			BeforeEach(func() {
				os.Setenv("OTEL_ENABLED", "false")
			})

			It("should disable telemetry", func() {
				cfg := telemetry.NewConfigFromEnv()

				Expect(cfg.Enabled).To(BeFalse())
			})
		})

		When("custom environment variables are set", func() {
			BeforeEach(func() {
				os.Setenv("OTEL_SERVICE_NAME", "custom-service")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")
				os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
				os.Setenv("OTEL_TRACES_SAMPLER", "parentbased_traceidratio")
				os.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")
				os.Setenv("OTEL_LOGS_EXPORTER", "console")
			})

			It("should use the custom values", func() {
				cfg := telemetry.NewConfigFromEnv()

				Expect(cfg.ServiceName).To(Equal("custom-service"))
				Expect(cfg.OTLPEndpoint).To(Equal("otel-collector:4317"))
				Expect(cfg.OTLPInsecure).To(BeFalse())
				Expect(cfg.TracesSampler).To(Equal("parentbased_traceidratio"))
				Expect(cfg.TracesSamplerArg).To(Equal(0.5))
				Expect(cfg.LogsExporter).To(Equal("console"))
			})
		})

		When("OTEL_ENABLED has an invalid value", func() {
			BeforeEach(func() {
				os.Setenv("OTEL_ENABLED", "not-a-bool")
			})

			It("should fall back to default (true)", func() {
				cfg := telemetry.NewConfigFromEnv()

				Expect(cfg.Enabled).To(BeTrue())
			})
		})

		When("OTEL_TRACES_SAMPLER_ARG has an invalid value", func() {
			BeforeEach(func() {
				os.Setenv("OTEL_TRACES_SAMPLER_ARG", "not-a-number")
			})

			It("should fall back to default (1.0)", func() {
				cfg := telemetry.NewConfigFromEnv()

				Expect(cfg.TracesSamplerArg).To(Equal(1.0))
			})
		})
	})
})
