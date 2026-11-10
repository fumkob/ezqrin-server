# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.3] - 2026-11-10

### Changed
- Upgraded Go runtime from 1.26.2 to 1.27.3
- Updated all OpenTelemetry packages in lockstep (otel v1.47.0, sdk/log v1.47.0 — now stable, otelgin v0.72.0, otelzap v0.21.0)
- Bumped direct dependencies to their latest versions, including `kin-openapi`, `oapi-codegen/runtime`, `pgx/v5`, `go-redis`, `golang-migrate`, `ginkgo`/`gomega`, `x/crypto`, and `google.golang.org/api`
- Telemetry output changes from the upgraded instrumentation (dashboards or queries relying on the old names may need updating):
  - PostgreSQL spans (`otelpgx` v0.13.0) are now named by operation (e.g. `SELECT`) instead of the full SQL text, which remains in the `db.query.text` attribute; attributes follow current DB semantic conventions (`db.system.name`) and the operation duration metric is now recorded in seconds
  - Errors logged via `zap.Error` are exported to the OTel log pipeline as the record's error instead of a plain `error` attribute (stdout JSON logs are unchanged)
  - Metrics now apply the SDK default cardinality limit of 2000 attribute sets per instrument
- `go-redis` v9.23.0 uses a separate connection pool for pipelines, so the client may open up to twice as many Redis connections under load
- Upgraded development tooling: `golangci-lint` v2.12.1 → v2.15.0 (required to lint modules targeting Go 1.27)

## [0.2.2] - 2026-05-06

### Fixed
- Apply `sort` and `order` query parameters on `GET /events` to the database query — previously they were silently ignored and results were always sorted by `created_at DESC`. Sortable columns are restricted via a whitelist (`created_at`, `updated_at`, `name`, `start_date`, `end_date`, `status`) to guard against SQL injection. (#49)
- Preserve backticks in GitHub Release notes generated from CHANGELOG entries by passing the body via an environment variable in the release workflow. (#79)

## [0.2.1] - 2026-05-06

### Changed
- Upgraded Go runtime from 1.26.1 to 1.26.2
- Updated all OpenTelemetry packages in lockstep (otel v1.43.0, sdk/log v0.19.0, otelhttp v0.68.0, otelzap v0.18.0, otelgin v0.68.0)
- Bumped direct dependencies to their latest versions, including `kin-openapi`, `validator`, `jwt/v5`, `pgx/v5`, `ginkgo`/`gomega`, `go-redis`, `zap`, `x/crypto`, `google.golang.org/api`, and `grpc`
- Upgraded development tooling: `golangci-lint` v2.11.3 → v2.12.1 and `oapi-codegen` v2.5.1 → v2.7.0
- Regenerated API code with `oapi-codegen` v2.7.0 — adds `Valid()` methods on enum types, `omitempty` JSON tags on optional fields, and a typed `BearerAuthScopes` context key constant
- Adjusted handler routing code to accommodate the typed `BearerAuthScopes` context key

## [0.2.0] - 2026-05-05

### Added
- OpenTelemetry observability across HTTP, PostgreSQL, and Redis layers — traces, metrics, and structured logs exported via OTLP gRPC, with `OTEL_*` environment variables (and `OTEL_ENABLED=false` to opt out) (#76)
- Local telemetry stack (`docker-compose.telemetry.yaml`) bundling OTel Collector, Jaeger, Prometheus, Loki, and Grafana, runnable via `make telemetry-up` / `make telemetry-down` (#76)

### Changed
- Upgraded Go runtime from 1.25.5 to 1.26.1 (#67)
- Eliminated cross-layer code duplication and standardized error handling (#73)
- Removed package-level globals in favor of dependency injection, generated usecase mocks, and fixed transaction context propagation in repositories (#74)

## [0.1.0] - 2026-03-14

Initial beta release of ezQRin Server — a Go-based backend API for QR code-based event check-in management.

### Added

#### Authentication & Users
- JWT-based authentication API with role-based access control (admin / staff)
- User management with soft delete and PII anonymization
- Mobile-aware refresh token expiry via User-Agent detection

#### Events
- Full CRUD API for event management (`GET/POST/PUT/DELETE /api/v1/events`)
- Timezone-aware event datetime handling
- Participant count and check-in count included in event responses

#### Participants & QR Codes
- Participant registration with automatic QR code generation
- QR code format using HMAC-SHA256 signing for tamper-proof tokens
- QR distribution URL for hosting server integration
- Bulk participant registration
- CSV import (supports UTF-8 BOM and Japanese ○△× status symbols)
- CSV export (`GET /api/v1/events/{id}/participants/export`)

#### Check-in
- QR code scan and manual check-in support
- Employee ID-based check-in
- Check-in statistics with per-status breakdown

#### QR Code Email Distribution
- Bulk QR code email delivery to participants (`POST /api/v1/events/{id}/qrcodes/send`)
- HTML email template with embedded QR code image
- Plain-text email fallback to bypass corporate HTML filtering
- Apple Wallet pass URL included in notification emails

#### Infrastructure
- Clean Architecture project structure (domain / repository / usecase / handler)
- OpenAPI-first development with `oapi-codegen` code generation
- PostgreSQL with `golang-migrate` migration management
- Redis cache with connection pooling and health checks
- Structured logging with `zap`
- DevContainer with Air hot-reload and Delve debugger support
- Production Dockerfile supporting Docker and Podman
- GitHub Actions CI pipeline (lint + vet + test in parallel)
- Ginkgo/Gomega BDD-style integration and E2E test suite

[0.2.3]: https://github.com/fumkob/ezqrin-server/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/fumkob/ezqrin-server/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/fumkob/ezqrin-server/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/fumkob/ezqrin-server/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/fumkob/ezqrin-server/releases/tag/v0.1.0
