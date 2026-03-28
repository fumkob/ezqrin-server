# Observability Design

## Overview

ezQRin Server adopts OpenTelemetry (OTel) to provide comprehensive observability across the three
telemetry signals: Traces, Metrics, and Logs. This document defines the design and intended
behavior for implementing observability in the system. It serves as the authoritative specification
for all observability-related design and implementation work.

In local development, Jaeger, Prometheus, and Grafana are used for visualization. In production,
the backend is fully replaceable via OTel Collector configuration with no application code changes
required.

---

## Goals & Non-Goals

### Goals

- Visualize request flows end-to-end with distributed tracing
- Measure performance and error rates through metrics
- Automatically inject Trace ID and Span ID into log output for log-trace correlation
- Collect and aggregate structured logs via OTel Logs SDK and Loki for log-trace correlation
- Enable telemetry visualization in local environments via Jaeger, Prometheus, and Grafana
- Support exporter switching through environment variables
- Allow complete telemetry disablement via `OTEL_ENABLED=false`

### Non-Goals

- Production infrastructure provisioning (Cloud Trace, Datadog, etc.)
- Production log backend provisioning (Cloud Logging, Datadog Logs, etc.) — achievable via Collector config swap with no application changes
- Custom Grafana dashboard creation
- Instrumentation of external service calls such as email sending and QR code generation

---

## Architecture

### Signal Flow

```
┌─────────────────────────────────────────────────────────┐
│  ezQRin Server (Go)                                     │
│                                                         │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────────┐  │
│  │ otelgin │  │ otelpgx  │  │redisotel│  │  OTel Log    │  │
│  │ MW      │  │ tracer   │  │ hook    │  │  Bridge (Zap)│  │
│  └────┬────┘  └────┬─────┘  └────┬────┘  └──────┬───────┘  │
│       │            │             │               │          │
│  ┌────▼────────────▼─────────────▼───────────────▼────┐     │
│  │     OTel SDK (TracerProvider, MeterProvider,        │     │
│  │     LoggerProvider)                                 │     │
│  └────────────────────────┬────────────────────────────┘     │
│                           │ OTLP (gRPC)                      │
└───────────────────────────┼──────────────────────────────────┘
                            ▼
                  ┌─────────────────┐
                  │  OTel Collector │
                  │  (Gateway)      │
                  └──┬──────┬───┬───┘
                     │      │   │
           ┌─────────▼┐  ┌──▼──────────┐  ┌──▼──┐
           │  Jaeger  │  │ Prometheus  │  │Loki │
           │ (traces) │  │ (metrics)   │  │(logs)│
           └────┬─────┘  └──┬──────────┘  └──┬──┘
                │           │                 │
           ┌────▼───────────▼─────────────────▼───┐
           │              Grafana                  │
           │  (unified visualization)              │
           └───────────────────────────────────────┘
```

### Design Principles

- The application exports telemetry using a single protocol: OTLP over gRPC
- The OTel Collector receives all signals and routes traces to Jaeger and metrics to Prometheus
- Zap logs are bridged to OTel Logs SDK via a Log Bridge, enabling structured log export over OTLP alongside traces and metrics
- Grafana integrates Jaeger, Prometheus, and Loki as data sources for unified visualization across all three signals
- In production, Loki can be replaced with Cloud Logging, Datadog Logs, or other backends by changing only the Collector exporter configuration
- Switching backends requires only Collector configuration changes; no application code changes

---

## Technology Stack

| Component                  | Technology                                       | Purpose                                    |
| -------------------------- | ------------------------------------------------ | ------------------------------------------ |
| Tracing SDK                | go.opentelemetry.io/otel/sdk                     | TracerProvider, MeterProvider              |
| OTLP Exporter              | otlptrace/otlptracegrpc, otlpmetric/otlpmetricgrpc | Telemetry data export                    |
| Gin Instrumentation        | otelgin                                          | Automatic HTTP request instrumentation     |
| PostgreSQL Instrumentation | exaring/otelpgx                                  | Automatic DB query instrumentation         |
| Redis Instrumentation      | redis/go-redis/extra/redisotel/v9                | Automatic Redis command instrumentation    |
| Collector                  | otel/opentelemetry-collector-contrib             | Telemetry reception and routing            |
| OTel Logs SDK              | go.opentelemetry.io/otel/sdk/log                 | LoggerProvider configuration               |
| OTLP Log Exporter          | otlplog/otlploggrpc                              | Log data export over OTLP                  |
| Zap OTel Bridge            | go.opentelemetry.io/contrib/bridges/otelzap      | Bridge Zap logs to OTel Logs SDK           |
| Log Backend                | grafana/loki                                      | Log aggregation and storage                |
| Trace Backend              | jaegertracing/all-in-one                         | Trace storage and visualization            |
| Metrics Backend            | prom/prometheus                                  | Metrics collection and storage             |
| Dashboard                  | grafana/grafana                                  | Unified visualization (traces, metrics, logs) |

---

## Instrumentation Scope

### HTTP Layer (Gin Middleware)

Automatic instrumentation via `otelgin.Middleware()`.

- Span name: `HTTP {method} {route}`
- Attributes: `http.method`, `http.route`, `http.status_code`
- Context propagation: W3C TraceContext (`traceparent`, `tracestate`)
- Automatic metrics: `http.server.request.duration`, `http.server.active_requests`

**Middleware order:**

```
RequestID → OTelGin → Logging → Recovery → CORS → Auth
```

### Database Layer (PostgreSQL)

Automatic instrumentation via `otelpgx`.

- Tracer option added at `pgxpool` creation time
- Span: `db.query` (per query execution)
- Attributes: `db.system=postgresql`, `db.statement`, `db.operation`

### Cache Layer (Redis)

Automatic instrumentation via `redisotel.InstrumentClient()`.

- Span: `redis.command` (per command execution)
- Attributes: `db.system=redis`, `db.operation`

### Log Pipeline

Structured logs are exported to Loki via OTel Logs SDK for centralized aggregation and trace
correlation.

#### Zap to OTel Bridge

- Zap logger is wrapped with `otelzap.NewHandler()` to bridge log records to the OTel Logs SDK
- All Zap log output is simultaneously sent to stdout (for local debugging) and to OTel
  LoggerProvider (for export)
- Log records automatically include `trace_id` and `span_id` from the active span in context

#### Log Attributes

- `trace_id`: Trace correlation identifier (auto-injected from span context)
- `span_id`: Span correlation identifier (auto-injected from span context)
- `request_id`: Request identifier (preserved from existing middleware)
- `level`: Log level (info, warn, error, etc.)
- `caller`: Source file and line number

#### Behavior

- When a span exists in context: trace_id and span_id are automatically attached
- When no span is present: logging proceeds without error, correlation fields are omitted
- stdout output is always maintained regardless of OTel export status
- When `OTEL_ENABLED=false`: only stdout output, no OTel export

---

## Configuration

### Environment Variables

| Variable                      | Default           | Description                              |
| ----------------------------- | ----------------- | ---------------------------------------- |
| OTEL_ENABLED                  | true              | Enable or disable all telemetry          |
| OTEL_SERVICE_NAME             | ezqrin-server     | Service identifier name                  |
| OTEL_EXPORTER_OTLP_ENDPOINT   | localhost:4317    | OTel Collector gRPC endpoint             |
| OTEL_EXPORTER_OTLP_INSECURE   | true              | Disable TLS (for local development)      |
| OTEL_TRACES_SAMPLER           | always_on         | Sampling strategy                        |
| OTEL_TRACES_SAMPLER_ARG       | 1.0               | Sampling ratio                           |
| OTEL_LOG_LEVEL                | info              | OTel SDK internal log level              |
| OTEL_LOGS_EXPORTER                | otlp              | Log exporter type (otlp or none)         |

All variable names follow the official OTel environment variable naming convention (`OTEL_*`).

All three signals (traces, metrics, logs) share the same `OTEL_EXPORTER_OTLP_ENDPOINT`. Per-signal
endpoint override is possible via `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`,
`OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`, and `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` following the OTel
environment variable specification, but this is not required for typical usage.

### Configuration Struct

```go
type TelemetryConfig struct {
    Enabled          bool
    ServiceName      string
    OTLPEndpoint     string
    OTLPInsecure     bool
    TracesSampler    string
    TracesSamplerArg float64
}
```

---

## Local Development Infrastructure

The telemetry stack is defined in `docker-compose.telemetry.yaml`, separate from the existing
DevContainer compose setup to keep the development baseline unaffected.

### Services

| Service        | Image                                    | Ports                                        | Purpose                           |
| -------------- | ---------------------------------------- | -------------------------------------------- | --------------------------------- |
| otel-collector | otel/opentelemetry-collector-contrib     | 4317 (gRPC), 4318 (HTTP), 8889 (metrics)     | Telemetry reception and routing   |
| jaeger         | jaegertracing/all-in-one                 | 16686 (UI)                                   | Trace visualization               |
| prometheus     | prom/prometheus                          | 9090                                         | Metrics collection                |
| grafana        | grafana/grafana                          | 3000                                         | Unified dashboard                 |
| loki           | grafana/loki                             | 3100                                         | Log aggregation and storage       |

### OTel Collector Configuration (`otel-collector-config.yaml`)

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

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: otel-collector
    scrape_interval: 15s
    static_configs:
      - targets: ["otel-collector:8889"]
```

### Grafana

Jaeger, Prometheus, and Loki are configured as data sources via Grafana provisioning, so they are
available automatically on startup without manual configuration.

### Makefile Commands

```bash
make telemetry-up    # Start the telemetry stack
make telemetry-down  # Stop the telemetry stack
```

---

## Application Integration Points

### Existing File Changes

| File                                               | Change                                                                          |
| -------------------------------------------------- | ------------------------------------------------------------------------------- |
| cmd/api/main.go                                    | Add telemetry initialization after `initializeInfrastructure()`; call `Shutdown()` on exit |
| internal/infrastructure/database/postgres.go       | Add `otelpgx` Tracer option to `pgxpool` creation                               |
| internal/infrastructure/cache/redis/client.go      | Add `redisotel.InstrumentClient()` call after client creation                   |
| internal/interface/api/router.go                   | Add `otelgin` middleware immediately after `RequestID()`                        |
| pkg/logger/logger.go                               | Integrate `otelzap.NewHandler()` to bridge Zap output to OTel Logs SDK; retain `WithContext()` for `trace_id`/`span_id` field injection |
| config/config.go                                   | Add `TelemetryConfig` with environment variable reading                         |
| .env.example                                       | Add `OTEL_*` environment variable template entries                              |

The `pkg/logger` package does not depend on the `telemetry` package directly. The otelzap bridge
connection (passing the LoggerProvider to `otelzap.NewHandler()`) happens in `cmd/api/main.go`
during initialization, keeping the dependency direction clean.

### New Package

```
internal/infrastructure/telemetry/
├── telemetry.go    # Integrated provider initialization and shutdown
├── tracer.go       # TracerProvider configuration
├── meter.go        # MeterProvider configuration
├── logger.go       # LoggerProvider configuration and otelzap bridge setup
└── config.go       # Telemetry configuration struct
```

### Dependency Direction

The existing dependency direction is preserved unchanged:

```
handler → usecase → repository → domain
```

The `telemetry` package is placed in the `infrastructure` layer. The `domain` and `usecase` layers
are not affected.

---

## Testing Requirements

### Test Environment

All unit tests and integration tests disable telemetry via `OTEL_ENABLED=false`. The application
must behave identically whether telemetry is enabled or disabled.

### Test Cases

| Target        | Test                                                                          |
| ------------- | ----------------------------------------------------------------------------- |
| telemetry.go  | Provider initialization and shutdown complete without error                    |
| telemetry.go  | `OTEL_ENABLED=false` results in NoopProvider being configured                  |
| config.go     | Configuration values are correctly read from environment variables             |
| router.go     | Requests are processed correctly after otelgin middleware is added             |
| logger.go     | `trace_id` and `span_id` are present in logs when a span exists in the context |
| logger.go     | Logging does not error when no span is present in the context                  |

The `telemetry` package uses `tracetest.SpanRecorder` to verify span generation in unit tests.

### E2E Verification (Manual)

```bash
# 1. Start the telemetry stack
make telemetry-up

# 2. Send a request to the API
curl -X GET http://localhost:8080/api/v1/health

# 3. Verify traces in Jaeger UI
open http://localhost:16686

# 4. Verify metrics in Grafana
open http://localhost:3000
```

---

## Future Extensions

- Log aggregation and visualization via Loki
- Instrumentation of external service calls (email sending, QR code generation)
- Production backend support (Cloud Trace, Datadog, etc.)
- Custom Grafana dashboards
- Alert rule configuration

---

## Related Documentation

- [System Architecture Overview](./overview.md)
- [Security Design](./security.md)
- [Configuration Reference](../deployment/environment.md)
- [Docker Setup](../deployment/docker.md)
