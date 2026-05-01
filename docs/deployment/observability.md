# Observability Operations Guide

## Overview

ezQRin Server exports traces, metrics, and logs via OpenTelemetry (OTel). In local development, a
Docker-based stack — Jaeger, Prometheus, Loki, and Grafana — collects and visualizes all three
signals. For design decisions and architecture details, see the
[Observability Design Specification](../architecture/observability.md).

---

## 5-Minute Quick Start

Run these four steps in order to have traces appearing in Jaeger within five minutes.

**Step 1 — Start the telemetry stack** (in one terminal):

```bash
make telemetry-up
```

This starts the OTel Collector, Jaeger, Prometheus, Loki, and Grafana. Wait a few seconds for all
containers to become healthy.

**Step 2 — Start the API server** (in a second terminal):

```bash
air
# or, without hot reload:
make run
```

The API server starts on `http://localhost:8080`. Telemetry is enabled by default in development.

**Step 3 — Send a request to generate a trace**:

```bash
curl http://localhost:8080/health
```

**Step 4 — Open Jaeger and find your trace**:

```bash
open http://localhost:16686
```

In the Jaeger UI: select **Service** `ezqrin-server`, click **Find Traces**. The trace for the
health check request appears in the results. Click on it to expand the span tree.

---

## Service UIs and Ports

| Service                     | URL                        | Purpose                                         |
| --------------------------- | -------------------------- | ----------------------------------------------- |
| Jaeger UI                   | http://localhost:16686      | Distributed trace viewer                        |
| Grafana                     | http://localhost:3000       | Unified dashboard (anonymous Admin auto-login)  |
| Prometheus                  | http://localhost:9090       | Metrics storage and PromQL queries              |
| Loki                        | http://localhost:3100       | Log aggregation (accessed via Grafana Explore)  |
| OTel Collector (gRPC)       | localhost:4317              | OTLP receive endpoint — app sends here          |
| OTel Collector (HTTP)       | localhost:4318              | OTLP over HTTP (alternative protocol)           |
| Prometheus exporter         | localhost:8889              | Collector-exposed metrics scrape endpoint       |

Grafana has anonymous access enabled with the Admin role (`GF_AUTH_ANONYMOUS_ENABLED=true`,
`GF_AUTH_ANONYMOUS_ORG_ROLE=Admin`). No login is required. Jaeger, Prometheus, and Loki are
pre-registered as data sources via `grafana/provisioning/datasources/datasources.yaml`. Prometheus
is the default data source.

---

## Viewing Traces in Jaeger

Open http://localhost:16686.

1. In the **Service** dropdown, select `ezqrin-server`.
2. Optionally filter by **Operation** (e.g., `GET /health` or `db.query`).
3. Click **Find Traces**. Results appear as a timeline of recent traces sorted by time.
4. Click any trace row to open the span tree.

The span tree shows every operation that occurred during the request. The root span is the HTTP
handler (instrumented by otelgin). Child spans represent downstream operations:

- Database queries appear as `db.query` spans with attributes such as `db.system=postgresql`,
  `db.statement`, and `db.operation`.
- Redis commands appear as `redis.command` spans with `db.system=redis` and `db.operation`.

Each span shows its start time, duration, and attributes. Click a span row to expand its attribute
list. Useful attributes to look for:

- `http.method`, `http.route`, `http.status_code` — HTTP handler span
- `db.statement` — the exact SQL query executed
- `db.operation` — the SQL verb (SELECT, INSERT, etc.)
- `trace_id` — the full trace identifier (use this to correlate with logs)

---

## Viewing Logs in Grafana (Loki)

Open http://localhost:3000 and navigate to **Explore** (compass icon in the left sidebar).

Select **Loki** from the data source dropdown at the top.

Enter a query in the log browser. Useful queries:

```
# All logs from ezqrin-server
{service_name="ezqrin-server"}

# Filter to error-level logs only
{service_name="ezqrin-server"} |= "error"

# Filter logs by a specific trace ID (from Jaeger)
{service_name="ezqrin-server"} | json | trace_id="<TRACE_ID>"
```

Replace `<TRACE_ID>` with the trace identifier copied from the Jaeger UI. Because the Zap logger
bridges logs to OTel via `otelzap`, every log line emitted while a span is active includes the
`trace_id` and `span_id` fields automatically.

Click **Run query** (or press Shift+Enter). Log lines appear in reverse chronological order. Click
any line to expand the full JSON payload including `trace_id`, `span_id`, `level`, `caller`, and
`request_id`.

---

## Viewing Metrics in Prometheus or Grafana

### Prometheus UI

Open http://localhost:9090 and use the **Graph** tab to run PromQL queries.

Useful queries:

```promql
# Total HTTP request count
http_server_request_duration_count

# Average HTTP request latency over the last minute
rate(http_server_request_duration_sum[1m]) / rate(http_server_request_duration_count[1m])

# Total database query count
db_client_operation_duration_count
```

### Grafana Explore (Prometheus)

Open http://localhost:3000, navigate to **Explore**, and select **Prometheus** as the data source
(it is the default). Enter the same PromQL queries above. Grafana renders the results as a time
series graph and provides autocomplete for metric names.

---

## Trace-to-Log Correlation

Because every log line includes a `trace_id` field, you can jump between a trace in Jaeger and the
corresponding logs in Loki.

**Manual workflow:**

1. Open a trace in Jaeger and copy the **Trace ID** displayed at the top of the trace view.
2. Open Grafana Explore, select Loki as the data source.
3. Paste the trace ID into a query:
   ```
   {service_name="ezqrin-server"} | json | trace_id="<TRACE_ID>"
   ```
4. The resulting log lines correspond exactly to the operations in that trace.

**Grafana Trace-to-Logs feature:**

Grafana supports direct navigation from a trace panel to related log lines. When viewing a trace
in Grafana's Jaeger panel, a **Logs** button or link appears next to individual spans. Clicking it
opens Loki Explore pre-filtered to the matching `trace_id`. This feature requires no additional
configuration because Loki is already registered as a data source.

---

## Auto-Instrumentation Coverage

The following libraries provide automatic instrumentation with no manual span creation required:

- **otelgin** (HTTP layer) — instruments every Gin request, creating a root span per request with
  `http.method`, `http.route`, and `http.status_code` attributes. Also records HTTP request
  duration metrics automatically (`http_server_request_duration`, `http_server_active_requests`).
- **otelpgx** (PostgreSQL) — instruments every pgxpool query, creating a child span per SQL
  statement with `db.statement`, `db.operation`, and `db.system=postgresql` attributes.
- **redisotel** (Redis) — instruments every go-redis command via `redisotel.InstrumentClient()`,
  creating a child span per command with `db.system=redis` and `db.operation` attributes.
- **otelzap bridge** (Logs) — wraps the Zap logger with `otelzap.NewHandler()` so every log record
  is exported to the OTel LoggerProvider (and then to Loki via the Collector). When a span is
  active in the request context, `trace_id` and `span_id` are injected into the log record
  automatically.

---

## Environment Variable Reference

All telemetry behavior is controlled by `OTEL_*` environment variables. These are read at startup
and override the values in `config/default.yaml`.

| Variable                      | Default          | Type    | Purpose                                                                 |
| ----------------------------- | ---------------- | ------- | ----------------------------------------------------------------------- |
| `OTEL_ENABLED`                | `true`           | Boolean | Master switch. Set to `false` to disable all telemetry (NoopProvider). |
| `OTEL_SERVICE_NAME`           | `ezqrin-server`  | String  | Service identifier shown in Jaeger, Prometheus labels, and Loki logs.  |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | String  | OTel Collector gRPC endpoint the app sends telemetry to.               |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true`           | Boolean | Disables TLS on the gRPC connection. Set to `false` in production.     |
| `OTEL_TRACES_SAMPLER`         | `always_on`      | String  | Sampling strategy. Options: `always_on`, `always_off`, `traceidratio`. |
| `OTEL_TRACES_SAMPLER_ARG`     | `1.0`            | Float   | Sampling ratio when sampler is `traceidratio` (0.0–1.0).               |
| `OTEL_LOGS_EXPORTER`          | `otlp`           | String  | Log export target. Use `otlp` to send logs to Collector, `none` to disable log export while keeping traces and metrics active. |

### Choosing a Sampling Strategy

- `always_on` — every request is traced. Use this in local development.
- `always_off` — no traces are created. Useful for disabling tracing only while keeping metrics
  and logs.
- `traceidratio` — a fraction of requests are sampled. Set `OTEL_TRACES_SAMPLER_ARG` to the
  desired ratio (e.g., `0.1` for 10%). Use this in production-like environments to reduce volume.

**Local development (default):**

```bash
OTEL_ENABLED=true
OTEL_TRACES_SAMPLER=always_on
OTEL_TRACES_SAMPLER_ARG=1.0
OTEL_LOGS_EXPORTER=otlp
```

**Production-like environment (10% sampling):**

```bash
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.internal:4317
OTEL_EXPORTER_OTLP_INSECURE=false
OTEL_TRACES_SAMPLER=traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
OTEL_LOGS_EXPORTER=otlp
```

---

## Disabling Telemetry

**Temporarily disable in local development:**

```bash
OTEL_ENABLED=false air
```

Or set it in your `.env` file:

```bash
OTEL_ENABLED=false
```

The application runs normally with all telemetry replaced by no-op providers. No Collector
connection is attempted.

**During tests** — telemetry is automatically disabled. `config/test.yaml` sets
`telemetry.enabled: false`, so tests never attempt to connect to the Collector and no OTel-related
error logs appear in test output.

**Production without a Collector** — if the OTel Collector is not yet available, set
`OTEL_ENABLED=false` to prevent connection errors, or set `OTEL_LOGS_EXPORTER=none` to disable
only log export while keeping traces and metrics active (if a partial Collector pipeline exists).

---

## Production Deployment Notes (Future)

The application exports all telemetry signals in OTLP format with no backend-specific code. Switching
from the local Jaeger/Prometheus/Loki stack to Cloud Trace, Datadog, Cloud Logging, or any other
OTel-compatible backend requires only changes to the OTel Collector exporter configuration
(`otel-collector-config.yaml`). No application code changes are needed. See the
[Observability Design Specification](../architecture/observability.md) for the full architecture
and future extension plans.

---

## Troubleshooting

### No traces appear in Jaeger

1. Confirm the Collector container is running: `docker ps | grep otel-collector`
2. Confirm `OTEL_ENABLED=true` (the default in development).
3. Confirm `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317` (the default).
4. Check Collector logs for errors:
   ```bash
   docker logs $(docker ps -qf name=otel-collector)
   ```
5. Restart the stack: `make telemetry-down && make telemetry-up`

### Loki or Prometheus not visible in Grafana

The data sources are provisioned automatically from
`grafana/provisioning/datasources/datasources.yaml`. If they do not appear in Grafana Explore:

1. Restart the stack: `make telemetry-down && make telemetry-up`
2. Verify the provisioning file is mounted: `docker inspect <grafana-container> | grep provisioning`
3. Check Grafana logs: `docker logs $(docker ps -qf name=grafana)`

### Logs do not contain `trace_id`

The `trace_id` field is injected automatically when the logger is called with a context that
contains an active span. If `trace_id` is missing from log lines:

- Ensure the request handler passes the request context through to downstream calls.
- Ensure `logger.WithContext(ctx)` (or equivalent) is used before logging within a handler or
  usecase function. The Zap OTel bridge reads the span from the context provided at log call time.

### `make telemetry-up` fails to start

A port conflict is the most common cause. The stack uses ports 4317, 4318, 8889, 16686, 9090,
3100, and 3000. Check which process is holding a port:

```bash
lsof -i :4317
lsof -i :16686
lsof -i :3000
```

Stop the conflicting process or service, then run `make telemetry-up` again.

### Tests are slow or show OTel-related error logs

Tests should always run with telemetry disabled. Verify that `config/test.yaml` is being picked up:

```yaml
# config/test.yaml
telemetry:
  enabled: false
```

If the test environment variable `SERVER_ENV=test` is not set, the test config may not be loaded.
Set `SERVER_ENV=test` or `OTEL_ENABLED=false` explicitly in your test environment.

---

## Related Documentation

- [Observability Design Specification](../architecture/observability.md) — architecture, design
  decisions, instrumentation scope
- [Configuration Reference](./environment.md) — all environment variables
- [Docker Setup Guide](./docker.md) — DevContainer and service configuration
