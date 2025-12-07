# Rate Limiting Strategy

## Overview

Rate limiting protects the ezQRin API from abuse, ensures fair resource allocation, and maintains
service stability. Limits are applied per client, per user, and per resource based on operation
type.

**Key Principles:**

- Lenient limits for legitimate use cases
- Stricter limits for high-risk operations (auth, tokens)
- Resource-based limits for data-intensive operations
- IP-based limits for unauthenticated requests
- User-based limits for authenticated requests

---

## Authentication & Security Operations

These operations have strict limits due to security sensitivity.

### Login Attempts

| Limit                | Window            | Scope                      |
| -------------------- | ----------------- | -------------------------- |
| 5 attempts           | 15 minutes        | Per IP address             |
| After limit exceeded | 15-minute lockout | Account temporarily locked |

**Trigger:** Failed authentication attempt (invalid credentials)

**Response:** `429 Too Many Requests`

**Retry-After Header:** `900` (15 minutes in seconds)

---

### User Registration

| Limit            | Window       | Scope            |
| ---------------- | ------------ | ---------------- |
| 10 registrations | Per hour     | Per IP address   |
| 3 registrations  | Per 24 hours | Per email domain |

**Purpose:** Prevent automated account creation and email enumeration attacks

**Response:** `429 Too Many Requests`

---

### Token Operations

#### Magic Link Validation (Participant Portal)

| Limit                 | Window   | Scope                |
| --------------------- | -------- | -------------------- |
| 5 validation attempts | Per hour | Per IP address       |
| 1 successful use      | N/A      | Per magic link token |

**Purpose:** Prevent brute-force magic link exploitation

**Response:** `429 Too Many Requests` (after 5 failed attempts)

---

## Email Operations

### Send QR Codes via Email

| Limit             | Window       | Scope         | Notes                   |
| ----------------- | ------------ | ------------- | ----------------------- |
| 5 send operations | Per minute   | Per event     | Bulk send operations    |
| 100 emails        | Per minute   | Per event     | Individual email count  |
| 1,000 emails      | Per hour     | Per event     | Cumulative hourly limit |
| 5,000 emails      | Per 24 hours | Per organizer | Organizer-wide limit    |

**Special Handling for Large Batches:**

- Batches < 100 recipients: Synchronous processing, subject to minute-level limits
- Batches ≥ 100 recipients: Queued asynchronously, subject to hourly/daily limits
- Maximum batch size: 5,000 recipients per request

**Example Scenarios:**

- ✅ 50 small batches (2-minute window): 50 × 80 emails = 4,000 emails/hour (within limit)
- ✅ 1 large batch (5,000 emails): Processed asynchronously, counts against daily limit
- ❌ 15 minute-long batches with 100+ emails each: Would exceed 100 emails/minute limit

**Response:** `429 Too Many Requests` with `Retry-After` header

---

## Data Retrieval Operations

These operations have lenient limits to support legitimate use cases.

### Event Management Endpoints

| Operation               | Limit        | Window     | Scope     |
| ----------------------- | ------------ | ---------- | --------- |
| List events             | 100 requests | Per minute | Per user  |
| Get event details       | 100 requests | Per minute | Per user  |
| List participants       | 50 requests  | Per minute | Per event |
| Get participant details | 100 requests | Per minute | Per event |

**Purpose:** Prevent excessive data extraction

**Recommended for UI:** Poll event/participant lists no more than once per 10 seconds

---

### Participant Portal

| Operation         | Limit       | Window     | Scope       |
| ----------------- | ----------- | ---------- | ----------- |
| List my events    | 30 requests | Per minute | Per session |
| Get event details | 30 requests | Per minute | Per session |
| Get QR code       | 20 requests | Per minute | Per session |
| Check-in status   | 20 requests | Per minute | Per session |

**Purpose:** Participant self-service portal usage (not heavy polling)

---

### QR Code Operations

#### Get Individual QR Code

| Limit        | Window     | Scope           |
| ------------ | ---------- | --------------- |
| 50 requests  | Per minute | Per event       |
| 200 requests | Per hour   | Per participant |

**Purpose:** Retrieve QR codes for display/printing

**Special Cases:**

- QR code in different formats (PNG, SVG, JSON): Single request
- Different sizes: Single request
- Caching: Server may cache for 5 minutes to reduce generation overhead

---

## Check-in Operations

### Record Check-in

| Limit            | Window       | Scope          |
| ---------------- | ------------ | -------------- |
| 50 check-ins     | Per minute   | Per event      |
| 200 check-ins    | Per minute   | Per IP address |
| 10,000 check-ins | Per 24 hours | Per event      |

**Purpose:** Support high-volume event check-in (e.g., large conferences)

**Calculation:** 50 check-ins/minute = 3,000 check-ins/hour (supports events with ~3K participants)

**Response:** `429 Too Many Requests` when limit exceeded

---

### Check-in Status Queries

| Limit        | Window     | Scope          |
| ------------ | ---------- | -------------- |
| 100 requests | Per minute | Per event      |
| 50 requests  | Per minute | Per IP address |

**Purpose:** Query check-in status without triggering recording

---

## Job/Async Operations

### Job Status Polling

| Limit           | Window     | Scope    |
| --------------- | ---------- | -------- |
| 10 status polls | Per minute | Per job  |
| 50 status polls | Per minute | Per user |

**Purpose:** Monitor async job progress (email sends, bulk imports)

**Recommendation:** Poll every 5 seconds maximum (12 polls/minute), no more than once per second

---

## API-Wide Limits

### Request Size Limits

| Resource                | Limit            | Notes                |
| ----------------------- | ---------------- | -------------------- |
| Request body            | 10 MB            | Per request          |
| Participant bulk import | 50,000 records   | Per import operation |
| Bulk email send         | 5,000 recipients | Per operation        |

### Connection Limits

| Limit                        | Scope          |
| ---------------------------- | -------------- |
| 100 concurrent connections   | Per IP address |
| 1,000 concurrent connections | Per API server |

---

## Burst Handling

The API uses **sliding window** rate limiting with burst allowance:

**Algorithm:**

- Rate limits apply over rolling time windows
- Burst requests up to 150% of limit allowed if previous window was under-utilized
- Smooth distribution encouraged via progressive backoff

**Example (Email: 100/minute limit):**

- Minute 1: Send 150 emails (burst allowed, 50% over limit)
- Minute 2: Allowed 50 emails only (to average out the burst)
- Result: Fair distribution over 2-minute window

---

## Response Headers

All rate-limited endpoints include headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1701345600
Retry-After: 35
```

**Fields:**

- `X-RateLimit-Limit`: Maximum requests allowed in window
- `X-RateLimit-Remaining`: Requests remaining before limit
- `X-RateLimit-Reset`: Unix timestamp when limit resets
- `Retry-After`: Seconds to wait before retrying (429 responses)

---

## Error Handling

### Rate Limit Exceeded Response

**HTTP Status:** `429 Too Many Requests`

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Maximum 100 requests per minute allowed.",
    "details": {
      "limit": 100,
      "window": "1 minute",
      "scope": "per_event"
    }
  },
  "retry_after_seconds": 45
}
```

**Headers:**

```
HTTP/1.1 429 Too Many Requests
Retry-After: 45
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1701345600
```

---

## Monitoring & Alerts

### Rate Limit Monitoring

**Server tracks:**

- Request count per client/user/resource
- Burst patterns and anomalies
- Repeated rate limit violations

**Alerts triggered on:**

- Single IP exceeding limits more than 10 times/hour (potential attack)
- Unusual burst patterns (200% of normal traffic)
- Repeated failed auth attempts (potential credential stuffing)

### Client Recommendations

**Recommended monitoring:**

- Check `X-RateLimit-Remaining` header
- Implement exponential backoff on 429 responses
- Cache responses when appropriate (e.g., event details)
- Use webhooks for async events instead of polling

**Backoff strategy:**

```
Attempt 1: Immediate
Attempt 2: Wait 2 seconds
Attempt 3: Wait 4 seconds
Attempt 4: Wait 8 seconds
Attempt 5: Wait 16 seconds
Max: Wait 60 seconds
```

---

## Rate Limit Policy Evolution

This document reflects the v1 API rate limits. Limits may be adjusted based on:

- Service capacity and performance
- Abuse patterns
- User feedback and legitimate use cases

**Last Updated:** 2025-11-22

**Change Log:**

- v1.0 (2025-11-22): Initial rate limits defined

---

## Related Documentation

- [Authentication API](./authentication.md) - Auth endpoints and login limits
- [QR Code API](./qrcode.md) - Email sending and participant self-service access
- [Check-in API](./checkin.md) - Check-in recording operations
