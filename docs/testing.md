# API Testing & Sandbox Mode Guide

## Overview

The ezQRin API provides a comprehensive testing environment for developers and event organizers to
validate workflows without affecting production data or sending real emails.

This guide covers testing strategies across different environments and the Sandbox Mode feature for
safe API testing.

**Key Concepts:**

- **Environments**: Development, Staging, and Production environments with different testing
  capabilities
- **Sandbox Mode**: A test mode activated via `X-Test-Mode: true` header that provides mock
  responses and isolated test data
- **Test Organization**: Dedicated organization for testing, separate from production data

---

## Environment-Specific Testing Strategies

ezQRin supports three deployment environments, each with different testing capabilities and
restrictions.

### Environment Overview

| Environment     | Sandbox Mode | Purpose                               | Data Isolation           |
| --------------- | ------------ | ------------------------------------- | ------------------------ |
| **Development** | ✅ Enabled   | Local development and feature testing | Isolated test database   |
| **Staging**     | ✅ Enabled   | Pre-production integration testing    | Isolated test database   |
| **Production**  | ❌ Disabled  | Live production environment           | Production database only |

### Development Environment

**Configuration:** `ENV=development`

**Testing Capabilities:**

- ✅ Sandbox Mode (`X-Test-Mode: true`) fully enabled
- ✅ All test endpoints available (`/api/v1/test/*`)
- ✅ Mock email capture and inspection
- ✅ Test data generation and cleanup
- ✅ Relaxed rate limits (10x normal limits)
- ✅ Debug endpoints enabled
- ✅ Verbose logging for troubleshooting

**Use Cases:**

- Feature development and testing
- API integration testing
- Workflow validation
- Error scenario testing
- Load testing with simulated data

**Data Management:**

- Test data stored in isolated development database
- Automatic cleanup after 30 days
- Manual cleanup via test endpoints
- No impact on production data

### Staging Environment

**Configuration:** `ENV=staging`

**Testing Capabilities:**

- ✅ Sandbox Mode (`X-Test-Mode: true`) fully enabled
- ✅ All test endpoints available (`/api/v1/test/*`)
- ✅ Mock email capture and inspection
- ✅ Test data generation and cleanup
- ✅ Relaxed rate limits (10x normal limits)
- ✅ Production-like configuration for realistic testing
- ✅ Integration testing with external services (optional)

**Use Cases:**

- Pre-production validation
- End-to-end workflow testing
- Integration testing with external systems
- Performance testing under production-like conditions
- User acceptance testing (UAT)

**Data Management:**

- Test data stored in isolated staging database
- Automatic cleanup after 30 days
- Manual cleanup via test endpoints
- No impact on production data

**Note:** Staging environment should mirror production configuration as closely as possible to
ensure accurate testing results.

### Production Environment

**Configuration:** `ENV=production`

**Testing Capabilities:**

- ❌ Sandbox Mode (`X-Test-Mode: true`) **disabled** or ignored
- ❌ Test endpoints (`/api/v1/test/*`) **not available**
- ✅ Standard API endpoints only
- ✅ Real email sending (no mock capture)
- ✅ Production rate limits enforced
- ✅ Strict security and logging

**Restrictions:**

- `X-Test-Mode: true` header is ignored or returns an error
- Test endpoints return `404 Not Found` or `403 Forbidden`
- All operations use production database
- Real emails are sent (no test queue)
- No test data generation or cleanup endpoints

**Important:** Production environment is for live operations only. All testing should be completed
in Development or Staging environments before deploying to Production.

---

## Sandbox Mode

Sandbox Mode is a testing feature that allows you to test API functionality without affecting
production data or sending real emails. It is activated using the `X-Test-Mode: true` header.

### Availability by Environment

| Environment     | Sandbox Mode Available | Notes                                    |
| --------------- | ---------------------- | ---------------------------------------- |
| **Development** | ✅ Yes                 | Fully enabled with all features          |
| **Staging**     | ✅ Yes                 | Fully enabled with all features          |
| **Production**  | ❌ No                  | Disabled for security and data integrity |

**Important:** In Production environment, the `X-Test-Mode: true` header is ignored or will return
an error. All testing must be completed in Development or Staging environments.

### Activation

Use `X-Test-Mode: true` header to enable sandbox mode for any request:

```
POST /api/v1/events
X-Test-Mode: true
Content-Type: application/json

{
  "name": "Test Event",
  ...
}
```

**Behavior in Sandbox Mode:**

- ✅ All API endpoints work normally
- ✅ Data stored in test database (isolated from production)
- ✅ No emails sent (captured in test queue)
- ✅ QR codes generated but marked as test
- ✅ Wallet passes created but unsigned
- ✅ Rate limits relaxed (10x higher limits)
- ❌ No real Stripe/payment processing
- ❌ No production email sending
- ❌ No webhooks to external systems

**Environment-Specific Behavior:**

- **Development & Staging:** All sandbox features are available
- **Production:** Sandbox mode is disabled; requests with `X-Test-Mode: true` will be rejected or
  ignored

### Test Data Retention

**Automatic Cleanup:**

- Sandbox data expires after 30 days
- Can be manually deleted via cleanup endpoint
- No impact on production data

**Cleanup Endpoint:**

```
DELETE /api/v1/test/cleanup
X-Test-Mode: true
```

**Response:**

```json
{
  "success": true,
  "data": {
    "events_deleted": 42,
    "participants_deleted": 2156,
    "storage_freed_mb": 125
  },
  "message": "Test data cleaned up successfully"
}
```

---

## Test Event Creation

> **Note:** Test event creation endpoints are only available in Development and Staging
> environments. In Production, these endpoints return `404 Not Found` or `403 Forbidden`.

### Quick Test Event

Create a pre-configured test event with sample data:

```
POST /api/v1/test/events
X-Test-Mode: true
Content-Type: application/json

{
  "name": "My Test Conference",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T17:00:00Z",
  "include_test_participants": true,
  "participant_count": 10
}
```

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "event_id": "test_550e8400-e29b-41d4-a716-446655440000",
    "event_name": "My Test Conference",
    "organizer": {
      "email": "test@example.com"
    },
    "participants": [
      {
        "id": "test_770e8400-e29b-41d4-a716-446655440000",
        "name": "Test Participant 1",
        "email": "test1+evt@example.com",
        "qr_code": "evt_test_550e8400_prt_test_770e8400_abc123def456"
      },
      ...
    ],
    "credentials": {
      "access_token": "test_token_...",
      "refresh_token": "test_refresh_..."
    },
    "note": "This is a TEST event. All data will be deleted after 30 days."
  }
}
```

### Realistic Test Data

```
POST /api/v1/test/events
X-Test-Mode: true
Content-Type: application/json

{
  "name": "Tech Conference 2025",
  "location": "San Francisco Convention Center",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T17:00:00Z",
  "description": "Annual technology conference",
  "participant_count": 5000,
  "data_set": "realistic"
}
```

**Generates:**

- 5,000 test participants with realistic names/emails
- Varied check-in times throughout the day
- Multiple staff members
- Realistic metadata (job titles, companies, etc.)

---

## Mock Email Responses

> **Note:** Mock email features are only available in Development and Staging environments when
> using `X-Test-Mode: true`. In Production, emails are sent normally and cannot be captured in a
> test queue.

### Email Delivery Simulation

Instead of sending real emails, test mode captures them in the email queue:

**Send Test Emails:**

```
POST /api/v1/events/:event_id/qrcodes/send
X-Test-Mode: true
Content-Type: application/json

{
  "participant_ids": ["test_770e8400-e29b-41d4-a716-446655440000"],
  "send_to_all": false,
  "include_wallet_pass": true
}
```

**Test Response:**

```json
{
  "success": true,
  "data": {
    "sent_count": 1,
    "failed_count": 0,
    "email_queue_id": "queue_test_123456"
  },
  "message": "Emails queued for mock delivery"
}
```

### Retrieve Test Email Content

Access the email content without it being sent:

```
GET /api/v1/test/emails/:email_id
X-Test-Mode: true
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "email_test_123456",
    "to": "test1+evt@example.com",
    "subject": "Your QR Code for Tech Conference 2025",
    "body_preview": "Hello Test Participant 1,\n\nYour event is scheduled for...",
    "attachments": [
      {
        "filename": "event_ticket.pkpass",
        "size_bytes": 2048,
        "type": "application/vnd.apple.pkpass"
      }
    ],
    "rendered_html": "Full HTML email content...",
    "created_at": "2025-11-08T10:00:00Z"
  }
}
```

### Simulate Email Events

Test webhook payloads for email delivery:

```
POST /api/v1/test/emails/:email_id/simulate-delivery
X-Test-Mode: true
Content-Type: application/json

{
  "event_type": "delivered",
  "timestamp": "2025-11-08T10:05:00Z",
  "metadata": {
    "provider": "sendgrid",
    "response_code": 250
  }
}
```

**Supported Events:**

- `queued` - Email in sending queue
- `sent` - Email successfully sent
- `delivered` - Confirmed delivery
- `bounce` - Hard bounce (invalid email)
- `complaint` - Spam complaint received
- `open` - Recipient opened email
- `click` - Recipient clicked link

---

## Test QR Codes

> **Note:** Test QR code features (watermarks, test prefixes) are only available in Development and
> Staging environments when using `X-Test-Mode: true`. In Production, all QR codes are
> production-grade and cannot be marked as test.

### Generating Test QR Codes

All QR codes in sandbox mode include visual indicators:

```
GET /api/v1/participants/:pid/qrcode?format=png&size=512
X-Test-Mode: true
```

**Test QR Code Features:**

- Clear "TEST" watermark in red
- Readable by check-in scanners (watermark doesn't affect scanning)
- Contains `test_` prefix in token
- Expires after 30 days (test mode cleanup)

### QR Code Validation Testing

Test how check-in handles different QR code states:

```
POST /api/v1/test/qrcodes/validate
X-Test-Mode: true
Content-Type: application/json

{
  "qr_token": "evt_test_550e8400_prt_test_770e8400_abc123def456",
  "scenario": "normal"
}
```

**Scenarios:**

| Scenario             | Behavior         | Use Case                |
| -------------------- | ---------------- | ----------------------- |
| `normal`             | Valid, checkable | Baseline test           |
| `expired`            | Expired token    | Test error handling     |
| `invalid`            | Malformed token  | Test validation         |
| `already_checked_in` | Already used     | Test duplicate check-in |
| `event_cancelled`    | Event cancelled  | Test event state        |
| `tampered`           | Modified token   | Test security           |

**Response:**

```json
{
  "success": true,
  "data": {
    "scenario": "normal",
    "is_valid": true,
    "participant_id": "test_770e8400-e29b-41d4-a716-446655440000",
    "event_id": "test_550e8400-e29b-41d4-a716-446655440000",
    "can_check_in": true,
    "details": "QR code is valid and can be checked in"
  }
}
```

---

## Check-in Simulation

> **Note:** Check-in simulation features are only available in Development and Staging environments
> when using `X-Test-Mode: true`. In Production, all check-ins are recorded normally in the
> production database.

### Mock Check-in Recording

Record check-ins without going through the full queue:

```
POST /api/v1/events/:event_id/checkins
X-Test-Mode: true
Content-Type: application/json

{
  "qr_code": "evt_test_550e8400_prt_test_770e8400_abc123def456"
}
```

### Batch Check-in Simulation

Simulate large-scale check-in operations:

```
POST /api/v1/test/events/:event_id/simulate-checkins
X-Test-Mode: true
Content-Type: application/json

{
  "count": 100,
  "duration_seconds": 600,
  "scenario": "rush_hour"
}
```

**Scenarios:**

- `steady` - Uniform distribution over duration
- `rush_hour` - Concentrated at start
- `gradual` - Slow start, accelerating
- `realistic` - Random distribution with peaks

---

## Rate Limit Testing

> **Note:** Relaxed rate limits in Sandbox Mode are only available in Development and Staging
> environments. In Production, standard rate limits are always enforced regardless of headers.

### Relaxed Limits in Sandbox

Sandbox mode increases all rate limits by 10x:

| Operation         | Normal Limit | Sandbox Limit |
| ----------------- | ------------ | ------------- |
| Email sending     | 100/min      | 1,000/min     |
| QR code retrieval | 50/min       | 500/min       |
| Login attempts    | 5/15min      | 50/15min      |
| API requests      | Standard     | 10x standard  |

### Rate Limit Simulation

Test rate limit behavior:

```
POST /api/v1/test/rate-limits/simulate
X-Test-Mode: true
Content-Type: application/json

{
  "endpoint": "/qrcodes/send",
  "requests_per_second": 50,
  "duration_seconds": 10
}
```

---

## Webhook Testing

> **Note:** Webhook testing features are only available in Development and Staging environments when
> using `X-Test-Mode: true`. In Production, webhooks are sent to configured endpoints and cannot be
> captured in a test queue.

### Mock Webhook Events

Capture and test webhook payloads in sandbox mode:

```
GET /api/v1/test/webhooks
X-Test-Mode: true
```

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "wh_test_123456",
      "event_type": "email.delivered",
      "timestamp": "2025-11-08T10:05:00Z",
      "payload": { ... },
      "delivery_attempts": 1,
      "next_retry": null,
      "status": "success"
    }
  ]
}
```

### Replay Webhook

Resend a test webhook payload:

```
POST /api/v1/test/webhooks/:webhook_id/replay
X-Test-Mode: true
```

---

## Test Organization Setup

> **Note:** Test organization creation is only available in Development and Staging environments. In
> Production, this endpoint returns `404 Not Found` or `403 Forbidden`.

### Create Test Organization

```
POST /api/v1/test/organizations
Content-Type: application/json

{
  "name": "QA Test Organization",
  "email": "qa@example.com"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "org_id": "test_org_123456",
    "name": "QA Test Organization",
    "api_key": "test_sk_...",
    "webhook_url": "http://localhost:3000/webhooks",
    "note": "This is a TEST organization. All data will be deleted after 30 days."
  }
}
```

---

## Best Practices for Testing

### Environment Selection

**Choose the right environment for your testing needs:**

- **Development:** Use for feature development, unit testing, and rapid iteration
- **Staging:** Use for integration testing, UAT, and pre-production validation
- **Production:** No testing - production is for live operations only

**Always test in Development or Staging before deploying to Production.**

### 1. Workflow Testing

**Test the complete user journey:**

```javascript
// 1. Create test event with participants
const event = await createTestEvent()

// 2. Send QR codes via email
await sendQRCodes(event.id, { testMode: true })

// 3. Retrieve email to verify content
const email = await getTestEmail(event.id)
console.assert(email.subject.includes(event.name))

// 4. Validate QR code
const qrValidation = await validateQRCode(event.participants[0].qr_code)
console.assert(qrValidation.is_valid === true)

// 5. Simulate check-in
const checkin = await recordCheckin(event.id, event.participants[0].qr_code)
console.assert(checkin.success === true)
```

### 2. Error Scenario Testing

**Test edge cases and error conditions:**

```javascript
// Test invalid QR code
const invalidQR = await validateQRCode("invalid_token")
console.assert(invalidQR.error !== null)

// Test expired event
const expiredEvent = await recordCheckin(pastEvent.id, qrCode)
console.assert(expiredEvent.error.code === "EVENT_EXPIRED")

// Test duplicate check-in
await recordCheckin(event.id, qrCode) // First check-in
const duplicate = await recordCheckin(event.id, qrCode) // Should fail
console.assert(duplicate.error.code === "ALREADY_CHECKED_IN")
```

### 3. Load Testing

**Simulate high-volume operations:**

```javascript
// Simulate 1000 check-ins over 10 minutes
await simulateBatchCheckins(event.id, {
  count: 1000,
  duration_seconds: 600,
  scenario: "realistic"
})

// Monitor performance
const performance = await getSimulationMetrics()
console.log(`Average response time: ${performance.avg_latency_ms}ms`)
console.log(`P99 response time: ${performance.p99_latency_ms}ms`)
```

### 4. Data Validation

**Verify data consistency:**

```javascript
// Verify QR code appears in all retrieval methods
const directQR = await getQRCode(participant.id)
const portalQR = await participantPortal.getQRCode(event.id)
const emailQR = await getTestEmail(participant.id)

console.assert(directQR.qr_code === portalQR.qr_code)
console.assert(emailQR.body.includes(directQR.qr_code))
```

---

## Testing Checklist

### Pre-Launch Testing

- [ ] All API endpoints respond correctly
- [ ] Email templates render properly
- [ ] QR codes scan successfully
- [ ] Wallet passes download correctly
- [ ] Check-in flow works end-to-end
- [ ] Error messages are clear and helpful
- [ ] Rate limiting works as expected
- [ ] Webhook events are triggered correctly
- [ ] Data validation catches invalid inputs
- [ ] Sensitive data is not exposed in responses
- [ ] Large batch operations handle correctly
- [ ] Cleanup removes test data properly

### Integration Testing

- [ ] Email provider integration (SendGrid, etc.)
- [ ] Payment provider integration
- [ ] Webhook delivery to external systems
- [ ] Database transaction handling
- [ ] Cache invalidation on updates
- [ ] Session/token management
- [ ] CORS and security headers

---

## Cleanup & Teardown

### Manual Cleanup

```
DELETE /api/v1/test/organizations/:org_id
X-Test-Mode: true
```

### Automatic Cleanup

- Test data automatically deleted after 30 days
- Test email queue cleared weekly
- Test webhooks archived after 7 days

---

## Troubleshooting

### Test Mode Not Working

**If `X-Test-Mode: true` is not working:**

1. **Check Environment:**

   ```javascript
   // Verify you're not in Production
   // Sandbox Mode is disabled in Production
   ```

2. **Verify Header:**

   ```javascript
   // Ensure header is correctly set
   const response = await fetch("/api/v1/test/events", {
     headers: { "X-Test-Mode": "true" }
   })
   console.assert(response.status === 200)
   ```

3. **Check Environment Variable:**
   - Development: `ENV=development`
   - Staging: `ENV=staging`
   - Production: `ENV=production` (Sandbox Mode disabled)

### Test Data Not Appearing

```javascript
// Verify test mode is enabled and environment is correct
const response = await fetch("/api/v1/test/events", {
  headers: { "X-Test-Mode": "true" }
})
console.assert(response.status === 200)
```

**If test endpoints return 404 or 403:**

- You may be in Production environment where test endpoints are disabled
- Switch to Development or Staging environment

### Emails Not Being Captured

```javascript
// Check email queue (only works in Development/Staging with X-Test-Mode)
const emails = await fetch("/api/v1/test/emails", {
  headers: { "X-Test-Mode": "true" }
})
console.assert(emails.data.length > 0)
```

**If emails are being sent instead of captured:**

- Verify `X-Test-Mode: true` header is present
- Check that you're not in Production environment
- In Production, emails are always sent normally

---

## Related Documentation

- [Environment Configuration](../deployment/environment.md) - Environment variables and
  configuration for Development/Staging/Production
- [QR Code API](./qrcode.md) - QR code specifications and participant self-service access
- [Check-in API](./checkin.md) - Check-in endpoint documentation
- [Docker Deployment](../deployment/docker.md) - Docker setup for different environments

---
