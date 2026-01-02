# ezQRin Documentation

Welcome to the ezQRin API documentation. This repository contains comprehensive guides for using and
deploying the ezQRin event check-in system.

## Overview

ezQRin is a modern event management and check-in system that uses QR codes to streamline participant
registration and attendance tracking. Built with Go and PostgreSQL, it provides a robust REST API
for managing events, participants, and check-ins.

## Documentation Structure

### 📚 API Documentation

- [API Index](./api/README.md) - Navigation hub and quick reference
- [Authentication](./api/authentication.md) - User authentication and authorization
- [Users](./api/users.md) - User account management, profile updates, and deletion with PII
  anonymization
- [Events](./api/events.md) - Event management endpoints
- [Participants](./api/participants.md) - Participant registration and management
- [Check-in](./api/checkin.md) - Check-in operations and history
- [QR Codes](./api/qrcode.md) - QR code generation, wallet integration, and participant self-service
  access
- [Deletion Audit Logs](./api/deletion_logs.md) - Deletion audit trail and compliance tracking
  (Admin-only)
- [Request/Response Schemas](./api/schemas.md) - Common data structures
- [Rate Limiting](./api/rate_limits.md) - API rate limiting strategy and thresholds
- [Error Codes](./api/error_codes.md) - Complete error code reference with solutions
- [Internationalization](./api/internationalization.md) - Multi-language support (i18n)

### 🏗️ Architecture

- [System Overview](./architecture/overview.md) - High-level architecture and design patterns
- [Database Design](./architecture/database.md) - Database schema and relationships
- [Security](./architecture/security.md) - Authentication, authorization, and security measures

### 🧪 Testing & Development

- [Testing Guide](./testing.md) - API testing, sandbox mode, and quality assurance

### 🚀 Deployment

- [Docker Setup](./deployment/docker.md) - DevContainer and docker-compose configuration
- [Configuration Reference](./deployment/environment.md) - Hierarchical YAML configuration and environment variables

## Quick Start

### Base URL

```
http://localhost:8080/api/v1
```

### Authentication

Most API endpoints require JWT authentication. Include the access token in the Authorization header:

```
Authorization: Bearer <access_token>
```

### Standard Response Format

**Single Entity:**
```json
{
  "id": "evt_123",
  "name": "Tech Conference 2025",
  "status": "active"
}
```

**Collection with Pagination:**
```json
{
  "data": [
    {"id": "evt_123", "name": "Event 1"},
    {"id": "evt_456", "name": "Event 2"}
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

**Empty Success (DELETE, UPDATE):**
```
204 No Content
(empty body)
```

### Error Response Format

All errors follow [RFC 9457 Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html):

```json
{
  "type": "https://api.ezqrin.com/problems/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "The requested event was not found",
  "instance": "/api/v1/events/123",
  "code": "NOT_FOUND"
}
```

**Validation Errors:**
```json
{
  "type": "https://api.ezqrin.com/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "One or more validation errors occurred",
  "instance": "/api/v1/events",
  "code": "VALIDATION_ERROR",
  "errors": [
    {"field": "email", "message": "Invalid email format"},
    {"field": "start_date", "message": "Must be in the future"}
  ]
}
```

## API Versioning

The API uses URL versioning with the format `/api/v{version}/`. The current version is `v1`.

## Rate Limiting

API requests are rate-limited to ensure service quality. See
[Rate Limiting Strategy](./api/rate_limits.md) for comprehensive details.

**Quick Reference:**

- **Login attempts:** 5 per 15 minutes per IP
- **Email sending:** 100 emails per minute per event
- **Check-in:** 50 check-ins per minute per event
- **QR code retrieval:** 50 requests per minute per event

For full details including burst handling, exemptions, and all operation types, see
[Rate Limiting Strategy](./api/rate_limits.md).

## License

This project is licensed under the MIT License - see the LICENSE file for details.
