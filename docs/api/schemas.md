# Request & Response Schemas

## Overview

This document defines common data structures, validation rules, and schema specifications used
across the ezQRin API.

---

## Standard Response Format

The ezQRin API uses different response patterns based on the operation:

### Success Responses

#### Single Entity

For operations that return a single resource (GET, POST, PATCH):

```json
{
  "id": "evt_123",
  "name": "Tech Conference 2025",
  "description": "Annual technology conference",
  "status": "active",
  "start_date": "2025-06-15T09:00:00Z",
  "end_date": "2025-06-17T18:00:00Z"
}
```

The resource data is returned directly without any wrapper.

#### Collection with Pagination

For list operations (GET /events, GET /participants):

```json
{
  "data": [
    {
      "id": "evt_123",
      "name": "Event 1",
      "status": "active"
    },
    {
      "id": "evt_456",
      "name": "Event 2",
      "status": "upcoming"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Fields:**

| Field | Type  | Description                            |
| ----- | ----- | -------------------------------------- |
| data  | array | Array of resources                     |
| meta  | object | Pagination metadata (see below) |

#### Empty Success

For operations that don't return data (DELETE, some PATCH operations):

```
HTTP/1.1 204 No Content
(empty body)
```

### Error Responses

All errors follow **RFC 9457 Problem Details for HTTP APIs** format:

```json
{
  "type": "https://api.ezqrin.com/problems/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "The requested event was not found",
  "instance": "/api/v1/events/evt_123",
  "code": "NOT_FOUND"
}
```

**Required Fields:**

| Field    | Type    | Description                                        |
| -------- | ------- | -------------------------------------------------- |
| type     | string  | URI identifying the problem type                   |
| title    | string  | Short, human-readable summary                      |
| status   | integer | HTTP status code                                   |
| detail   | string  | Human-readable explanation of this occurrence      |
| instance | string  | URI identifying the specific occurrence            |

**Extension Fields:**

| Field  | Type   | Description                                    |
| ------ | ------ | ---------------------------------------------- |
| code   | string | Machine-readable error code (backward compatibility) |
| errors | array  | Validation error details (validation errors only)    |

#### Validation Errors

Validation errors include an `errors` array with field-level details:

```json
{
  "type": "https://api.ezqrin.com/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "One or more validation errors occurred",
  "instance": "/api/v1/events",
  "code": "VALIDATION_ERROR",
  "errors": [
    {
      "field": "email",
      "message": "Invalid email format"
    },
    {
      "field": "start_date",
      "message": "Start date must be in the future"
    }
  ]
}
```

**Validation Error Fields:**

| Field   | Type   | Description                      |
| ------- | ------ | -------------------------------- |
| field   | string | Field name that caused the error |
| message | string | Field-specific error message     |

---

## Pagination Schema

Paginated endpoints include metadata about the result set.

### Request Parameters

```
GET /api/v1/resource?page=1&per_page=20&sort=created_at&order=desc
```

**Query Parameters:**

| Parameter | Type    | Default    | Description                       |
| --------- | ------- | ---------- | --------------------------------- |
| page      | integer | 1          | Page number (min: 1)              |
| per_page  | integer | 20         | Items per page (min: 1, max: 100) |
| sort      | string  | created_at | Sort field name                   |
| order     | string  | desc       | Sort order: `asc`, `desc`         |

### Response Meta

```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Meta Fields:**

| Field       | Type    | Description           |
| ----------- | ------- | --------------------- |
| page        | integer | Current page number   |
| per_page    | integer | Items per page        |
| total       | integer | Total number of items |
| total_pages | integer | Total number of pages |

---

## Entity Schemas

### User

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "organizer",
  "created_at": "2025-11-08T10:00:00Z",
  "updated_at": "2025-11-08T10:00:00Z"
}
```

**Fields:**

| Field      | Type     | Constraints                             | Description            |
| ---------- | -------- | --------------------------------------- | ---------------------- |
| id         | UUID     | Read-only                               | User unique identifier |
| email      | string   | 1-255 chars, valid email format, unique | Email address          |
| name       | string   | 1-255 chars                             | Full name              |
| role       | enum     | `admin`, `organizer`, `staff`           | User role              |
| created_at | datetime | Read-only, ISO 8601, context-dependent  | Creation timestamp     |
| updated_at | datetime | Read-only, ISO 8601, context-dependent  | Last update timestamp  |

**Note:** `created_at` and `updated_at` are included in some responses (e.g., register) but not in
others (e.g., login). Clients should handle these fields as optional in authentication responses.

---

### Event

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "organizer_id": "660e8400-e29b-41d4-a716-446655440000",
  "organizer": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "John Doe",
    "email": "john@example.com"
  },
  "name": "Tech Conference 2025",
  "description": "Annual technology conference",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T18:00:00Z",
  "location": "San Francisco Convention Center",
  "timezone": "America/Los_Angeles",
  "status": "published",
  "participant_count": 150,
  "checked_in_count": 0,
  "created_at": "2025-11-08T10:00:00Z",
  "updated_at": "2025-11-08T10:00:00Z"
}
```

**Fields:**

| Field             | Type     | Constraints                                               | Description                          |
| ----------------- | -------- | --------------------------------------------------------- | ------------------------------------ |
| id                | UUID     | Read-only                                                 | Event unique identifier              |
| organizer_id      | UUID     | Read-only, references users(id)                           | Event owner                          |
| organizer         | object   | Read-only, included in detail views only                  | Event owner user details             |
| name              | string   | 1-255 chars, required                                     | Event name                           |
| description       | string   | 0-5000 chars                                              | Event description                    |
| start_date        | datetime | ISO 8601, required                                        | Event start time                     |
| end_date          | datetime | ISO 8601, must be after start_date                        | Event end time                       |
| location          | string   | 0-500 chars                                               | Venue/location                       |
| timezone          | string   | IANA timezone                                             | Event timezone (default: Asia/Tokyo) |
| status            | enum     | `draft`, `published`, `ongoing`, `completed`, `cancelled` | Event status                         |
| participant_count | integer  | Read-only                                                 | Total registered participants        |
| checked_in_count  | integer  | Read-only                                                 | Number of checked-in participants    |
| created_at        | datetime | Read-only, ISO 8601                                       | Creation timestamp                   |
| updated_at        | datetime | Read-only, ISO 8601                                       | Last update timestamp                |

---

### Participant

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440000",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Jane Smith",
  "email": "jane@example.com",
  "qr_email": "jane.work@example.com",
  "employee_id": "EMP001",
  "phone": "+1-555-0123",
  "status": "confirmed",
  "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
  "qr_code_generated_at": "2025-11-08T10:00:00Z",
  "metadata": {},
  "payment_status": "paid",
  "payment_amount": 150.0,
  "payment_date": "2025-11-08T12:30:00Z",
  "checked_in": false,
  "checked_in_at": null,
  "created_at": "2025-11-08T10:00:00Z",
  "updated_at": "2025-11-08T10:00:00Z"
}
```

**Fields:**

| Field                | Type     | Constraints                                       | Description                                |
| -------------------- | -------- | ------------------------------------------------- | ------------------------------------------ |
| id                   | UUID     | Read-only                                         | Participant unique identifier              |
| event_id             | UUID     | Read-only, references events(id)                  | Associated event                           |
| name                 | string   | 1-255 chars, required                             | Participant full name                      |
| email                | string   | Valid email, required, unique per event           | Email address                              |
| qr_email             | string   | Valid email, nullable                             | Alternative email for QR code distribution |
| employee_id          | string   | 0-255 chars, nullable                             | Employee or staff ID                       |
| phone                | string   | E.164 format                                      | Phone number                               |
| status               | enum     | `tentative`, `confirmed`, `cancelled`, `declined` | Participation status                       |
| qr_code              | string   | Read-only, unique                                 | QR code token                              |
| qr_code_generated_at | datetime | Read-only, ISO 8601                               | QR generation time                         |
| metadata             | object   | Max 10KB JSON, not included in list responses     | Custom participant data                    |
| payment_status       | enum     | `unpaid`, `paid` (default: unpaid)                | Payment status                             |
| payment_amount       | number   | Decimal (2 places), nullable                      | Payment amount                             |
| payment_date         | datetime | ISO 8601, nullable                                | Payment date/time                          |
| checked_in           | boolean  | Read-only                                         | Check-in status                            |
| checked_in_at        | datetime | Read-only, ISO 8601                               | Check-in timestamp                         |
| created_at           | datetime | Read-only, ISO 8601                               | Creation timestamp                         |
| updated_at           | datetime | Read-only, ISO 8601                               | Last update timestamp                      |

---

### Check-in

```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "participant_id": "770e8400-e29b-41d4-a716-446655440000",
  "checked_in_at": "2025-12-15T09:15:00Z",
  "checked_in_by": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Staff User"
  },
  "checkin_method": "qrcode",
  "device_info": {}
}
```

**Fields:**

| Field          | Type     | Constraints                                   | Description                 |
| -------------- | -------- | --------------------------------------------- | --------------------------- |
| id             | UUID     | Read-only                                     | Check-in unique identifier  |
| event_id       | UUID     | Read-only, references events(id)              | Associated event            |
| participant_id | UUID     | Read-only, references participants(id)        | Checked-in participant      |
| checked_in_at  | datetime | Read-only, ISO 8601, default NOW()            | Check-in timestamp          |
| checked_in_by  | object   | Read-only, contains id and name of staff user | User who performed check-in |
| checkin_method | enum     | `qrcode`, `manual`                            | Check-in method used        |
| device_info    | object   | Max 5KB JSON                                  | Device metadata             |

---

## Validation Rules

### Email Validation

**Format:** RFC 5322 compliant email address

**Rules:**

- Must contain `@` symbol
- Valid domain with TLD
- No leading/trailing whitespace
- Case-insensitive storage (normalized to lowercase)

**Examples:**

- ✅ `user@example.com`
- ✅ `john.doe+events@company.co.uk`
- ❌ `invalid@`
- ❌ `@example.com`
- ❌ `user @example.com`

---

### Phone Validation

**Format:** E.164 international format

**Rules:**

- Must start with `+`
- Country code (1-3 digits)
- Subscriber number (up to 15 digits total)
- No spaces or special characters

**Examples:**

- ✅ `+14155552671`
- ✅ `+442071838750`
- ✅ `+81312345678`
- ❌ `555-0123`
- ❌ `(415) 555-2671`
- ❌ `+1 415 555 2671`

---

### Password Validation

**Requirements:**

- Minimum 8 characters
- Maximum 128 characters
- At least one uppercase letter (A-Z)
- At least one lowercase letter (a-z)
- At least one number (0-9)
- Special characters recommended but not required

**Storage:**

- Hashed using bcrypt (cost factor 12)
- Never stored in plain text
- Never returned in API responses

---

### UUID Validation

**Format:** RFC 4122 UUID v4

**Rules:**

- 36 characters including hyphens
- Format: `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx`
- Hexadecimal characters (0-9, a-f)
- Case-insensitive

**Examples:**

- ✅ `550e8400-e29b-41d4-a716-446655440000`
- ✅ `6ba7b810-9dad-11d1-80b4-00c04fd430c8`
- ❌ `550e8400-e29b-41d4-a716` (incomplete)
- ❌ `not-a-valid-uuid`

---

### Datetime Validation

**Format:** ISO 8601

**Rules:**

- UTC timezone (Z suffix) or explicit timezone offset
- Format: `YYYY-MM-DDTHH:mm:ssZ` or `YYYY-MM-DDTHH:mm:ss±HH:mm`
- Microseconds optional: `YYYY-MM-DDTHH:mm:ss.ffffffZ`

**Examples:**

- ✅ `2025-12-15T09:00:00Z`
- ✅ `2025-12-15T09:00:00+09:00`
- ✅ `2025-12-15T09:00:00.123456Z`
- ❌ `2025-12-15 09:00:00`
- ❌ `12/15/2025`

---

### Metadata Validation

**Format:** Valid JSON object

**Rules:**

- Must be valid JSON
- Maximum size: 10KB for participants, 5KB for device_info
- Nested objects allowed
- Arrays allowed
- No executable code

**Example:**

```json
{
  "company": "Tech Corp",
  "role": "Engineer",
  "dietary_restrictions": ["vegetarian"],
  "preferences": {
    "session_track": "backend",
    "t_shirt_size": "L"
  }
}
```

---

## HTTP Status Codes

### Success Codes (2xx)

| Code | Name       | Usage                                       |
| ---- | ---------- | ------------------------------------------- |
| 200  | OK         | Successful GET, PUT, DELETE requests        |
| 201  | Created    | Successful POST request creating a resource |
| 204  | No Content | Successful request with no response body    |

### Client Error Codes (4xx)

| Code | Name                 | Usage                                               |
| ---- | -------------------- | --------------------------------------------------- |
| 400  | Bad Request          | Malformed request or invalid parameters             |
| 401  | Unauthorized         | Missing or invalid authentication                   |
| 403  | Forbidden            | Authenticated but not authorized                    |
| 404  | Not Found            | Resource does not exist                             |
| 409  | Conflict             | Resource conflict (duplicate, constraint violation) |
| 422  | Unprocessable Entity | Validation failed                                   |
| 429  | Too Many Requests    | Rate limit exceeded                                 |

### Server Error Codes (5xx)

| Code | Name                  | Usage                        |
| ---- | --------------------- | ---------------------------- |
| 500  | Internal Server Error | Unexpected server error      |
| 503  | Service Unavailable   | Temporary service disruption |

---

## Common Error Codes

### Authentication Errors (1xxx)

| Code                       | HTTP Status | Description                  |
| -------------------------- | ----------- | ---------------------------- |
| `AUTH_INVALID_CREDENTIALS` | 401         | Invalid email or password    |
| `AUTH_INVALID_TOKEN`       | 401         | Invalid or expired JWT token |
| `AUTH_UNAUTHORIZED`        | 401         | Missing authentication       |
| `AUTH_FORBIDDEN`           | 403         | Insufficient permissions     |
| `AUTH_EMAIL_EXISTS`        | 409         | Email already registered     |
| `AUTH_ACCOUNT_LOCKED`      | 403         | Too many failed attempts     |

### Validation Errors (2xxx)

| Code                        | HTTP Status | Description              |
| --------------------------- | ----------- | ------------------------ |
| `VALIDATION_FAILED`         | 422         | General validation error |
| `VALIDATION_INVALID_EMAIL`  | 422         | Invalid email format     |
| `VALIDATION_INVALID_PHONE`  | 422         | Invalid phone format     |
| `VALIDATION_INVALID_UUID`   | 400         | Invalid UUID format      |
| `VALIDATION_REQUIRED_FIELD` | 422         | Required field missing   |
| `VALIDATION_FIELD_TOO_LONG` | 422         | Field exceeds max length |

### Resource Errors (3xxx)

| Code                          | HTTP Status | Description                        |
| ----------------------------- | ----------- | ---------------------------------- |
| `EVENT_NOT_FOUND`             | 404         | Event does not exist               |
| `PARTICIPANT_NOT_FOUND`       | 404         | Participant does not exist         |
| `CHECKIN_NOT_FOUND`           | 404         | Check-in record not found          |
| `PARTICIPANT_DUPLICATE_EMAIL` | 409         | Email already registered for event |
| `CHECKIN_ALREADY_CHECKED_IN`  | 409         | Participant already checked in     |

### System Errors (9xxx)

| Code                  | HTTP Status | Description                     |
| --------------------- | ----------- | ------------------------------- |
| `INTERNAL_ERROR`      | 500         | Unexpected server error         |
| `DATABASE_ERROR`      | 500         | Database operation failed       |
| `SERVICE_UNAVAILABLE` | 503         | Service temporarily unavailable |
| `RATE_LIMIT_EXCEEDED` | 429         | Too many requests               |

---

## Request Headers

### Required Headers

```
Content-Type: application/json
Accept: application/json
```

### Authentication Header

```
Authorization: Bearer <access_token>
```

### Optional Headers

```
X-Request-ID: <client-generated-uuid>
User-Agent: <client-name>/<version>
Accept-Language: en-US,en;q=0.9
```

---

## Response Headers

### Standard Response Headers

```
Content-Type: application/json; charset=utf-8
X-Request-ID: <echo-or-server-generated-uuid>
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1636372800
```

### Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
```
