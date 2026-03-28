# OpenTelemetry Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add OpenTelemetry-based observability (traces, metrics, logs) to ezqrin-server with local development infrastructure (Jaeger, Prometheus, Loki, Grafana).

**Architecture:** A new `internal/infrastructure/telemetry` package initializes TracerProvider, MeterProvider, and LoggerProvider, all exporting via OTLP gRPC to an OTel Collector. Existing infrastructure (pgx, go-redis, Gin) is instrumented via official OTel contrib libraries. Zap logs are bridged to OTel Logs SDK via `otelzap`. The telemetry stack runs in a separate `docker-compose.telemetry.yaml`.

**Tech Stack:** go.opentelemetry.io/otel SDK, otlptracegrpc, otlpmetricgrpc, otlploggrpc, otelgin, exaring/otelpgx, redis/go-redis/extra/redisotel/v9, go.opentelemetry.io/contrib/bridges/otelzap, OTel Collector, Jaeger, Prometheus, Loki, Grafana

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `internal/infrastructure/telemetry/telemetry.go` | Integrated provider initialization (`Init`) and shutdown (`Shutdown`) |
| `internal/infrastructure/telemetry/tracer.go` | TracerProvider configuration with OTLP exporter |
| `internal/infrastructure/telemetry/meter.go` | MeterProvider configuration with OTLP exporter |
| `internal/infrastructure/telemetry/logger.go` | LoggerProvider configuration, OTLP exporter, and otelzap bridge setup |
| `internal/infrastructure/telemetry/config.go` | `TelemetryConfig` struct definition |
| `internal/infrastructure/telemetry/telemetry_test.go` | Tests for provider initialization, shutdown, and noop mode |
| `internal/infrastructure/telemetry/config_test.go` | Tests for config reading |
| `internal/infrastructure/telemetry/logger_test.go` | Tests for log bridge and trace_id/span_id injection |
| `internal/infrastructure/telemetry/suite_test.go` | Ginkgo test suite bootstrap |
| `docker-compose.telemetry.yaml` | Telemetry stack: OTel Collector, Jaeger, Prometheus, Loki, Grafana |
| `otel-collector-config.yaml` | OTel Collector pipeline configuration |
| `prometheus.yaml` | Prometheus scrape config |
| `grafana/provisioning/datasources/datasources.yaml` | Grafana auto-provisioned data sources |

### Modified Files

| File | Change |
|------|--------|
| `config/config.go` | Add `TelemetryConfig` field to `Config`, env var binding, unmarshal, validation |
| `config/default.yaml` | Add `telemetry` section with defaults |
| `config/test.yaml` | Add `telemetry.enabled: false` |
| `cmd/api/main.go` | Initialize telemetry after logger, before infrastructure; call `Shutdown()` on exit; wire otelzap bridge |
| `internal/infrastructure/database/postgres.go` | Add `otelpgx.NewTracer()` option to pgxpool config |
| `internal/infrastructure/cache/redis/client.go` | Add `redisotel.InstrumentClient()` after client creation |
| `internal/interface/api/router.go` | Add `otelgin.Middleware()` after `RequestID()` |
| `pkg/logger/logger.go` | Add `WithOTelCore()` method to attach otelzap bridge core |
| `Makefile` | Add `telemetry-up` and `telemetry-down` targets |

---

## Task 1: TelemetryConfig struct and config integration

**Files:**
- Create: `internal/infrastructure/telemetry/config.go`
- Modify: `config/config.go`
- Modify: `config/default.yaml`
- Modify: `config/test.yaml`
- Create: `internal/infrastructure/telemetry/config_test.go`
- Create: `internal/infrastructure/telemetry/suite_test.go`

- [ ] **Step 1: Create the telemetry config struct**

Create `internal/infrastructure/telemetry/config.go`:

```go
package telemetry

// Config holds OpenTelemetry configuration.
type Config struct {
	Enabled          bool
	ServiceName      string
	OTLPEndpoint     string
	OTLPInsecure     bool
	TracesSampler    string
	TracesSamplerArg float64
	LogsExporter     string
}
```

- [ ] **Step 2: Create the Ginkgo test suite bootstrap**

Create `internal/infrastructure/telemetry/suite_test.go`:

```go
package telemetry_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTelemetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telemetry Suite")
}
```

- [ ] **Step 3: Write failing test for config defaults**

Create `internal/infrastructure/telemetry/config_test.go`:

```go
package telemetry_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
)

var _ = Describe("Config", func() {
	Describe("NewConfigFromEnv", func() {
		When("no environment variables are set", func() {
			It("returns default values", func() {
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
			AfterEach(func() {
				os.Unsetenv("OTEL_ENABLED")
			})

			It("returns Enabled=false", func() {
				cfg := telemetry.NewConfigFromEnv()
				Expect(cfg.Enabled).To(BeFalse())
			})
		})

		When("custom environment variables are set", func() {
			BeforeEach(func() {
				os.Setenv("OTEL_SERVICE_NAME", "my-service")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
				os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
				os.Setenv("OTEL_TRACES_SAMPLER", "traceidratio")
				os.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")
				os.Setenv("OTEL_LOGS_EXPORTER", "none")
			})
			AfterEach(func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
				os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
				os.Unsetenv("OTEL_TRACES_SAMPLER")
				os.Unsetenv("OTEL_TRACES_SAMPLER_ARG")
				os.Unsetenv("OTEL_LOGS_EXPORTER")
			})

			It("reads values from environment", func() {
				cfg := telemetry.NewConfigFromEnv()
				Expect(cfg.ServiceName).To(Equal("my-service"))
				Expect(cfg.OTLPEndpoint).To(Equal("collector:4317"))
				Expect(cfg.OTLPInsecure).To(BeFalse())
				Expect(cfg.TracesSampler).To(Equal("traceidratio"))
				Expect(cfg.TracesSamplerArg).To(Equal(0.5))
				Expect(cfg.LogsExporter).To(Equal("none"))
			})
		})
	})
})
```

- [ ] **Step 4: Run test to verify it fails**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1`
Expected: FAIL — `NewConfigFromEnv` not defined

- [ ] **Step 5: Implement NewConfigFromEnv**

Add to `internal/infrastructure/telemetry/config.go`:

```go
package telemetry

import (
	"os"
	"strconv"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	Enabled          bool
	ServiceName      string
	OTLPEndpoint     string
	OTLPInsecure     bool
	TracesSampler    string
	TracesSamplerArg float64
	LogsExporter     string
}

// NewConfigFromEnv creates a Config by reading OTEL_* environment variables with defaults.
func NewConfigFromEnv() Config {
	cfg := Config{
		Enabled:          true,
		ServiceName:      "ezqrin-server",
		OTLPEndpoint:     "localhost:4317",
		OTLPInsecure:     true,
		TracesSampler:    "always_on",
		TracesSamplerArg: 1.0,
		LogsExporter:     "otlp",
	}

	if v := os.Getenv("OTEL_ENABLED"); v != "" {
		cfg.Enabled, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.ServiceName = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		cfg.OTLPEndpoint = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"); v != "" {
		cfg.OTLPInsecure, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER"); v != "" {
		cfg.TracesSampler = v
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		cfg.TracesSamplerArg, _ = strconv.ParseFloat(v, 64)
	}
	if v := os.Getenv("OTEL_LOGS_EXPORTER"); v != "" {
		cfg.LogsExporter = v
	}

	return cfg
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1`
Expected: PASS — all config tests pass

- [ ] **Step 7: Add TelemetryConfig to application config**

Modify `config/config.go`:

1. Add to `Config` struct:
```go
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Logging   LoggingConfig
	CORS      CORSConfig
	QRCode    QRCodeConfig
	Email     EmailConfig
	Telemetry TelemetryConfig
}

type TelemetryConfig struct {
	Enabled          bool
	ServiceName      string
	OTLPEndpoint     string
	OTLPInsecure     bool
	TracesSampler    string
	TracesSamplerArg float64
	LogsExporter     string
}
```

2. Add environment variable mappings to `envKeyMap`:
```go
// Telemetry (OpenTelemetry)
"OTEL_ENABLED":                 "telemetry.enabled",
"OTEL_SERVICE_NAME":            "telemetry.service_name",
"OTEL_EXPORTER_OTLP_ENDPOINT":  "telemetry.otlp_endpoint",
"OTEL_EXPORTER_OTLP_INSECURE":  "telemetry.otlp_insecure",
"OTEL_TRACES_SAMPLER":          "telemetry.traces_sampler",
"OTEL_TRACES_SAMPLER_ARG":      "telemetry.traces_sampler_arg",
"OTEL_LOGS_EXPORTER":           "telemetry.logs_exporter",
```

3. Add unmarshal logic in `unmarshalConfig`:
```go
cfg.Telemetry.Enabled = v.GetBool("telemetry.enabled")
cfg.Telemetry.ServiceName = v.GetString("telemetry.service_name")
cfg.Telemetry.OTLPEndpoint = v.GetString("telemetry.otlp_endpoint")
cfg.Telemetry.OTLPInsecure = v.GetBool("telemetry.otlp_insecure")
cfg.Telemetry.TracesSampler = v.GetString("telemetry.traces_sampler")
cfg.Telemetry.TracesSamplerArg = v.GetFloat64("telemetry.traces_sampler_arg")
cfg.Telemetry.LogsExporter = v.GetString("telemetry.logs_exporter")
```

- [ ] **Step 8: Add telemetry defaults to YAML configs**

Add to `config/default.yaml`:
```yaml
telemetry:
  enabled: true
  service_name: ezqrin-server
  otlp_endpoint: localhost:4317
  otlp_insecure: true
  traces_sampler: always_on
  traces_sampler_arg: 1.0
  logs_exporter: otlp
```

Add to `config/test.yaml`:
```yaml
telemetry:
  enabled: false
```

- [ ] **Step 9: Run existing config tests**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./config/... -v -count=1`
Expected: PASS — existing tests still pass with new config fields

- [ ] **Step 10: Commit**

```bash
git add internal/infrastructure/telemetry/config.go internal/infrastructure/telemetry/config_test.go internal/infrastructure/telemetry/suite_test.go config/config.go config/default.yaml config/test.yaml
git commit -m "✨ Add TelemetryConfig struct and environment variable integration"
```

---

## Task 2: TracerProvider initialization

**Files:**
- Create: `internal/infrastructure/telemetry/tracer.go`
- Modify: `internal/infrastructure/telemetry/telemetry_test.go`

- [ ] **Step 1: Write failing test for TracerProvider**

Create `internal/infrastructure/telemetry/telemetry_test.go`:

```go
package telemetry_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var _ = Describe("Tracer", func() {
	Describe("NewTracerProvider", func() {
		When("telemetry is enabled", func() {
			It("creates a TracerProvider with the configured service name", func() {
				ctx := context.Background()
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

				// Verify it produces spans
				tracer := tp.Tracer("test")
				_, span := tracer.Start(ctx, "test-span")
				span.End()

				err = tp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("telemetry is disabled", func() {
			It("returns a noop TracerProvider", func() {
				ctx := context.Background()
				cfg := telemetry.Config{Enabled: false}

				tp, err := telemetry.NewTracerProvider(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())

				// Noop provider should still work without error
				tracer := tp.Tracer("test")
				_, span := tracer.Start(ctx, "test-span")
				span.End()

				err = tp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Tracer"`
Expected: FAIL — `NewTracerProvider` not defined

- [ ] **Step 3: Implement TracerProvider**

Create `internal/infrastructure/telemetry/tracer.go`:

```go
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewTracerProvider creates a new TracerProvider.
// When cfg.Enabled is false, it returns a noop provider.
func NewTracerProvider(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		return sdktrace.NewTracerProvider(), nil
	}

	dialOpts := []grpc.DialOption{}
	if cfg.OTLPInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	sampler := resolveSampler(cfg.TracesSampler, cfg.TracesSamplerArg)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return tp, nil
}

// resolveSampler maps sampler name to sdktrace.Sampler.
func resolveSampler(name string, arg float64) sdktrace.Sampler {
	switch name {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(arg)
	default: // "always_on"
		return sdktrace.AlwaysSample()
	}
}
```

- [ ] **Step 4: Run `go mod tidy` to fetch dependencies**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy`

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Tracer"`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/telemetry/tracer.go internal/infrastructure/telemetry/telemetry_test.go go.mod go.sum
git commit -m "✨ Add TracerProvider with OTLP gRPC exporter and sampler configuration"
```

---

## Task 3: MeterProvider initialization

**Files:**
- Create: `internal/infrastructure/telemetry/meter.go`
- Modify: `internal/infrastructure/telemetry/telemetry_test.go`

- [ ] **Step 1: Write failing test for MeterProvider**

Add to `internal/infrastructure/telemetry/telemetry_test.go`:

```go
var _ = Describe("Meter", func() {
	Describe("NewMeterProvider", func() {
		When("telemetry is enabled", func() {
			It("creates a MeterProvider without error", func() {
				ctx := context.Background()
				cfg := telemetry.Config{
					Enabled:      true,
					ServiceName:  "test-service",
					OTLPEndpoint: "localhost:4317",
					OTLPInsecure: true,
				}

				mp, err := telemetry.NewMeterProvider(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(mp).NotTo(BeNil())

				err = mp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("telemetry is disabled", func() {
			It("returns a noop MeterProvider", func() {
				ctx := context.Background()
				cfg := telemetry.Config{Enabled: false}

				mp, err := telemetry.NewMeterProvider(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(mp).NotTo(BeNil())

				err = mp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Meter"`
Expected: FAIL — `NewMeterProvider` not defined

- [ ] **Step 3: Implement MeterProvider**

Create `internal/infrastructure/telemetry/meter.go`:

```go
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewMeterProvider creates a new MeterProvider.
// When cfg.Enabled is false, it returns a noop provider.
func NewMeterProvider(ctx context.Context, cfg Config) (*metric.MeterProvider, error) {
	if !cfg.Enabled {
		return metric.NewMeterProvider(), nil
	}

	dialOpts := []grpc.DialOption{}
	if cfg.OTLPInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	return mp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Meter"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/telemetry/meter.go internal/infrastructure/telemetry/telemetry_test.go go.mod go.sum
git commit -m "✨ Add MeterProvider with OTLP gRPC exporter"
```

---

## Task 4: LoggerProvider and otelzap bridge

**Files:**
- Create: `internal/infrastructure/telemetry/logger.go`
- Modify: `pkg/logger/logger.go`
- Create: `internal/infrastructure/telemetry/logger_test.go`

- [ ] **Step 1: Write failing test for LoggerProvider**

Create `internal/infrastructure/telemetry/logger_test.go`:

```go
package telemetry_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
)

var _ = Describe("Logger", func() {
	Describe("NewLoggerProvider", func() {
		When("telemetry is enabled", func() {
			It("creates a LoggerProvider without error", func() {
				ctx := context.Background()
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

				err = lp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("telemetry is disabled", func() {
			It("returns a noop LoggerProvider", func() {
				ctx := context.Background()
				cfg := telemetry.Config{Enabled: false}

				lp, err := telemetry.NewLoggerProvider(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(lp).NotTo(BeNil())

				err = lp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("logs exporter is none", func() {
			It("returns a noop LoggerProvider", func() {
				ctx := context.Background()
				cfg := telemetry.Config{
					Enabled:      true,
					LogsExporter: "none",
				}

				lp, err := telemetry.NewLoggerProvider(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(lp).NotTo(BeNil())

				err = lp.Shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Logger"`
Expected: FAIL — `NewLoggerProvider` not defined

- [ ] **Step 3: Implement LoggerProvider**

Create `internal/infrastructure/telemetry/logger.go`:

```go
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewLoggerProvider creates a new LoggerProvider.
// When cfg.Enabled is false or LogsExporter is "none", it returns a noop provider.
func NewLoggerProvider(ctx context.Context, cfg Config) (*sdklog.LoggerProvider, error) {
	if !cfg.Enabled || cfg.LogsExporter == "none" {
		return sdklog.NewLoggerProvider(), nil
	}

	dialOpts := []grpc.DialOption{}
	if cfg.OTLPInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlploggrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	return lp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Logger"`
Expected: PASS

- [ ] **Step 5: Write failing test for logger WithOTelCore**

Add test to an existing logger test file or create one. This test verifies that the `pkg/logger` package can accept an additional Zap core for OTel bridging:

```go
// In pkg/logger/logger_test.go or appropriate test file
var _ = Describe("Logger", func() {
	Describe("WithOTelCore", func() {
		It("returns a new logger that includes the additional core", func() {
			cfg := logger.Config{
				Level:       "info",
				Format:      "json",
				Environment: "test",
			}
			log, err := logger.New(cfg)
			Expect(err).NotTo(HaveOccurred())

			// Create a test core (using zaptest observer)
			observedCore, logs := observer.New(zapcore.InfoLevel)
			newLog := log.WithOTelCore(observedCore)
			Expect(newLog).NotTo(BeNil())

			// Log a message — it should appear in the observer
			newLog.Info("test message")
			Expect(logs.Len()).To(Equal(1))
			Expect(logs.All()[0].Message).To(Equal("test message"))
		})
	})
})
```

- [ ] **Step 6: Implement WithOTelCore on Logger**

Add to `pkg/logger/logger.go`:

```go
import (
	"go.uber.org/zap/zapcore"
)

// WithOTelCore returns a new Logger that tees output to the given additional core.
// This is used to bridge Zap logs to OTel Logs SDK without creating a dependency
// from the logger package to the telemetry package.
func (l *Logger) WithOTelCore(core zapcore.Core) *Logger {
	combined := zapcore.NewTee(l.Logger.Core(), core)
	return &Logger{
		Logger: zap.New(combined, zap.AddCaller(), zap.AddCallerSkip(1)),
	}
}
```

- [ ] **Step 7: Run logger tests to verify they pass**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./pkg/logger/... -v -count=1`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/infrastructure/telemetry/logger.go internal/infrastructure/telemetry/logger_test.go pkg/logger/logger.go
git commit -m "✨ Add LoggerProvider with OTLP exporter and WithOTelCore bridge method"
```

---

## Task 5: Integrated Init/Shutdown and global OTel registration

**Files:**
- Create: `internal/infrastructure/telemetry/telemetry.go`
- Modify: `internal/infrastructure/telemetry/telemetry_test.go`

- [ ] **Step 1: Write failing test for Init and Shutdown**

Add to `internal/infrastructure/telemetry/telemetry_test.go`:

```go
var _ = Describe("Telemetry", func() {
	Describe("Init", func() {
		When("telemetry is enabled", func() {
			It("initializes all providers and returns a Shutdown function", func() {
				ctx := context.Background()
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
				Expect(shutdown).NotTo(BeNil())
				Expect(providers).NotTo(BeNil())
				Expect(providers.TracerProvider).NotTo(BeNil())
				Expect(providers.MeterProvider).NotTo(BeNil())
				Expect(providers.LoggerProvider).NotTo(BeNil())

				err = shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("telemetry is disabled", func() {
			It("returns noop providers and a no-op shutdown", func() {
				ctx := context.Background()
				cfg := telemetry.Config{Enabled: false}

				providers, shutdown, err := telemetry.Init(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(shutdown).NotTo(BeNil())
				Expect(providers).NotTo(BeNil())

				err = shutdown(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1 -run "Telemetry"`
Expected: FAIL — `Init` not defined

- [ ] **Step 3: Implement Init and Shutdown**

Create `internal/infrastructure/telemetry/telemetry.go`:

```go
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Providers holds the initialized OTel providers.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// ShutdownFunc shuts down all telemetry providers.
type ShutdownFunc func(ctx context.Context) error

// Init initializes OpenTelemetry providers and registers them globally.
// Returns a Providers struct, a shutdown function, and any initialization error.
func Init(ctx context.Context, cfg Config) (*Providers, ShutdownFunc, error) {
	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init tracer provider: %w", err)
	}

	mp, err := NewMeterProvider(ctx, cfg)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to init meter provider: %w", err)
	}

	lp, err := NewLoggerProvider(ctx, cfg)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to init logger provider: %w", err)
	}

	// Register globally
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	providers := &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
	}

	shutdown := func(ctx context.Context) error {
		return errors.Join(
			tp.Shutdown(ctx),
			mp.Shutdown(ctx),
			lp.Shutdown(ctx),
		)
	}

	return providers, shutdown, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/telemetry/... -v -count=1`
Expected: PASS — all telemetry tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/telemetry/telemetry.go internal/infrastructure/telemetry/telemetry_test.go
git commit -m "✨ Add integrated telemetry Init/Shutdown with global OTel registration"
```

---

## Task 6: Instrument PostgreSQL with otelpgx

**Files:**
- Modify: `internal/infrastructure/database/postgres.go`

- [ ] **Step 1: Add otelpgx tracer to pgxpool config**

In `internal/infrastructure/database/postgres.go`, modify `NewPostgresDB` to add the otelpgx tracer option to the pool config after `pgxpool.ParseConfig`:

```go
import (
	"github.com/exaring/otelpgx"
)

// After poolConfig is created, before pgxpool.NewWithConfig:
poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
```

The single line to add is:
```go
poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
```

This goes after `poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime` and before `pool, err := pgxpool.NewWithConfig(ctx, poolConfig)`.

- [ ] **Step 2: Run `go mod tidy`**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy`

- [ ] **Step 3: Run existing database tests**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/database/... -v -count=1`
Expected: PASS — existing tests pass (otelpgx is noop when no TracerProvider is configured)

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/database/postgres.go go.mod go.sum
git commit -m "✨ Add otelpgx instrumentation to PostgreSQL connection pool"
```

---

## Task 7: Instrument Redis with redisotel

**Files:**
- Modify: `internal/infrastructure/cache/redis/client.go`

- [ ] **Step 1: Add redisotel instrumentation after client creation**

In `internal/infrastructure/cache/redis/client.go`, add `redisotel.InstrumentTracing` and `redisotel.InstrumentMetrics` calls after `redis.NewClient`:

```go
import (
	"github.com/redis/go-redis/extra/redisotel/v9"
)

// After rdb := redis.NewClient(&redis.Options{...}), add:
if err := redisotel.InstrumentTracing(rdb); err != nil {
	return nil, fmt.Errorf("failed to instrument redis tracing: %w", err)
}
if err := redisotel.InstrumentMetrics(rdb); err != nil {
	return nil, fmt.Errorf("failed to instrument redis metrics: %w", err)
}
```

This goes after the `rdb := redis.NewClient(...)` call and before the Ping verification.

- [ ] **Step 2: Run `go mod tidy`**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy`

- [ ] **Step 3: Run existing Redis tests**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/infrastructure/cache/... -v -count=1`
Expected: PASS — existing tests pass (redisotel is noop when no providers are configured)

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/cache/redis/client.go go.mod go.sum
git commit -m "✨ Add redisotel instrumentation to Redis client"
```

---

## Task 8: Add otelgin middleware to router

**Files:**
- Modify: `internal/interface/api/router.go`

- [ ] **Step 1: Add otelgin middleware after RequestID**

In `internal/interface/api/router.go`, add `otelgin.Middleware` immediately after `middleware.RequestID()`:

```go
import (
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Change middleware order to:
router.Use(middleware.RequestID())                        // Generate request ID first
router.Use(otelgin.Middleware(deps.Config.Telemetry.ServiceName)) // OTel tracing
router.Use(middleware.Logging(deps.Logger))               // Log requests with request ID
router.Use(middleware.Recovery(deps.Logger))              // Recover from panics
router.Use(middleware.CORS(&deps.Config.CORS))            // Handle CORS
```

- [ ] **Step 2: Run `go mod tidy`**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go mod tidy`

- [ ] **Step 3: Run router tests**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./internal/interface/api/... -v -count=1`
Expected: PASS — existing handler tests pass with new middleware

- [ ] **Step 4: Commit**

```bash
git add internal/interface/api/router.go go.mod go.sum
git commit -m "✨ Add otelgin middleware for automatic HTTP request tracing"
```

---

## Task 9: Wire telemetry into main.go

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add telemetry initialization to main**

Modify `cmd/api/main.go`:

1. Add import:
```go
"github.com/fumkob/ezqrin-server/internal/infrastructure/telemetry"
"go.opentelemetry.io/contrib/bridges/otelzap"
```

2. Add `telemetryShutdown` field to `app` struct:
```go
type app struct {
	db                 database.Service
	logger             *logger.Logger
	cache              cache.Service
	telemetryShutdown  telemetry.ShutdownFunc
}
```

3. In `main()`, after logger initialization and before `initializeInfrastructure`, add telemetry init:
```go
// Initialize telemetry (after logger, before infrastructure)
telemetryCfg := telemetry.Config{
	Enabled:          cfg.Telemetry.Enabled,
	ServiceName:      cfg.Telemetry.ServiceName,
	OTLPEndpoint:     cfg.Telemetry.OTLPEndpoint,
	OTLPInsecure:     cfg.Telemetry.OTLPInsecure,
	TracesSampler:    cfg.Telemetry.TracesSampler,
	TracesSamplerArg: cfg.Telemetry.TracesSamplerArg,
	LogsExporter:     cfg.Telemetry.LogsExporter,
}
providers, shutdownTelemetry, err := telemetry.Init(ctx, telemetryCfg)
if err != nil {
	appLogger.Fatal("failed to initialize telemetry", zap.Error(err))
}
a.telemetryShutdown = shutdownTelemetry

// Bridge Zap logs to OTel Logs SDK
if cfg.Telemetry.Enabled {
	otelCore := otelzap.NewCore("ezqrin-server", otelzap.WithLoggerProvider(providers.LoggerProvider))
	appLogger = appLogger.WithOTelCore(otelCore)
	a.logger = appLogger
}
```

4. In `cleanup()`, add telemetry shutdown before logger sync:
```go
func (a *app) cleanup() {
	if a.logger != nil {
		a.logger.Info("shutting down application infrastructure")
	}

	if a.db != nil {
		a.db.Close()
	}

	if a.cache != nil {
		a.cache.Close()
	}

	// Shutdown telemetry before logger sync to flush remaining telemetry data
	if a.telemetryShutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := a.telemetryShutdown(ctx); err != nil {
			if a.logger != nil {
				a.logger.Error("telemetry shutdown error", zap.Error(err))
			}
		}
	}

	if a.logger != nil {
		_ = a.logger.Sync()
		a.logger.Info("cleanup completed")
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go build ./cmd/api/...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "✨ Wire telemetry initialization into application startup and shutdown"
```

---

## Task 10: Local development telemetry infrastructure

**Files:**
- Create: `docker-compose.telemetry.yaml`
- Create: `otel-collector-config.yaml`
- Create: `prometheus.yaml`
- Create: `grafana/provisioning/datasources/datasources.yaml`
- Modify: `Makefile`

- [ ] **Step 1: Create OTel Collector configuration**

Create `otel-collector-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 1024

exporters:
  otlphttp/jaeger:
    endpoint: http://jaeger:4318
  prometheus:
    endpoint: 0.0.0.0:8889
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/jaeger]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [loki]
```

- [ ] **Step 2: Create Prometheus configuration**

Create `prometheus.yaml`:

```yaml
scrape_configs:
  - job_name: otel-collector
    scrape_interval: 15s
    static_configs:
      - targets: ["otel-collector:8889"]
```

- [ ] **Step 3: Create Grafana datasource provisioning**

Create `grafana/provisioning/datasources/datasources.yaml`:

```yaml
apiVersion: 1

datasources:
  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
    isDefault: false

  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: false
```

- [ ] **Step 4: Create docker-compose.telemetry.yaml**

Create `docker-compose.telemetry.yaml`:

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "8889:8889"   # Prometheus metrics
    depends_on:
      - jaeger
      - loki

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686" # Jaeger UI

  prometheus:
    image: prom/prometheus:latest
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.retention.time=1h"
    volumes:
      - ./prometheus.yaml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "9090:9090"

  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"

  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    ports:
      - "3000:3000"
    depends_on:
      - jaeger
      - prometheus
      - loki
```

- [ ] **Step 5: Add Makefile targets**

Add to `Makefile`:

```makefile
## Telemetry
telemetry-up: ## Start the telemetry stack (Jaeger, Prometheus, Loki, Grafana)
	docker compose -f docker-compose.telemetry.yaml up -d

telemetry-down: ## Stop the telemetry stack
	docker compose -f docker-compose.telemetry.yaml down
```

- [ ] **Step 6: Commit**

```bash
git add docker-compose.telemetry.yaml otel-collector-config.yaml prometheus.yaml grafana/provisioning/datasources/datasources.yaml Makefile
git commit -m "✨ Add local telemetry infrastructure (OTel Collector, Jaeger, Prometheus, Loki, Grafana)"
```

---

## Task 11: Run full test suite and verify

**Files:** None (verification only)

- [ ] **Step 1: Run all unit tests**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && go test ./... -count=1 -short`
Expected: PASS — all tests pass, including new telemetry tests

- [ ] **Step 2: Run linter**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && make lint`
Expected: PASS — no lint errors

- [ ] **Step 3: Verify build**

Run: `cd /Users/fumkob/Documents/ezqrin/ezqrin-server/main && make build`
Expected: PASS — binary builds successfully

- [ ] **Step 4: Fix any issues found and commit**

If any tests fail or lint errors are found, fix them and commit:
```bash
git add -A
git commit -m "🐛 Fix issues found during observability verification"
```

---

## Dependency Order

```
Task 1 (Config) → Task 2 (Tracer) → Task 3 (Meter) → Task 4 (Logger + bridge)
                                                              ↓
Task 5 (Init/Shutdown) ← ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┘
     ↓
Task 6 (pgx)  ┐
Task 7 (redis) ├── can run in parallel after Task 5
Task 8 (gin)  ┘
     ↓
Task 9 (main.go wiring) ← depends on Tasks 5-8
     ↓
Task 10 (Docker infra) ← can run in parallel with Tasks 6-9
     ↓
Task 11 (Verification) ← depends on all previous tasks
```
