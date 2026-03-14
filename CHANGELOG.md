# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/fumkob/ezqrin-server/releases/tag/v0.1.0
