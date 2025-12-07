# Future Features - QR Code API

> These features are planned for future implementation after MVP validation and core functionality
> testing.

## Wallet Pass Integration (Planned)

### Overview

Support for Apple Wallet (PKPass) and Google Wallet integration, allowing participants to store
event tickets directly in their mobile wallet applications.

### Get Wallet Pass

Retrieve Apple Wallet or Google Wallet pass for a participant.

**Endpoint:** `GET /api/v1/participants/:pid/wallet`

**Authentication:** Not required (uses secure token in URL)

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| pid       | UUID | Participant ID |

**Query Parameters:**

| Parameter | Type   | Required | Description                                                         |
| --------- | ------ | -------- | ------------------------------------------------------------------- |
| token     | string | Yes      | Secure access token (sent via email)                                |
| platform  | string | No       | Wallet platform: `apple`, `google` (auto-detected if not specified) |

**Response (Apple Wallet):** `200 OK`

```
Content-Type: application/vnd.apple.pkpass
Content-Disposition: attachment; filename="event_ticket.pkpass"

[Binary PKPass data]
```

**Response (Google Wallet):** `302 Redirect`

```
Location: https://pay.google.com/gp/v/save/{jwt_token}
```

**Errors:**

- `400 Bad Request` - Invalid or missing token
- `404 Not Found` - Participant not found
- `410 Gone` - Token expired or event completed

---

## Wallet Pass Token Management

### Token Generation

Each wallet pass access requires a unique, cryptographically secure token:

**Format:**

```
wallet_{event_id}_{participant_id}_{random_token}_{timestamp_hash}
```

**Components:**

- `wallet_` prefix identifies this as a wallet pass token
- `event_id`: Short form of event UUID
- `participant_id`: Short form of participant UUID
- `random_token`: 16-character cryptographic random string (alphanumeric)
- `timestamp_hash`: HMAC-SHA256 hash of creation timestamp (first 8 chars)

**Example:**

```
wallet_550e8400_770e8400_k7x9m2pq8vb4c6d1_a3f2b8e9
```

### Token Storage

Wallet pass tokens are stored in the database with the following metadata:

**Database Schema:**

- `wallet_pass_tokens` table structure:
  - `id`: UUID primary key
  - `participant_id`: UUID (foreign key to participants)
  - `token_hash`: HMAC-SHA256 hash of the full token (indexed for fast lookup)
  - `platform`: Enum (apple, google)
  - `created_at`: Timestamp
  - `expires_at`: Timestamp (when token becomes invalid)
  - `accessed_at`: Timestamp (NULL until first use)
  - `used_count`: Integer (tracks access attempts)
  - `invalidated_at`: Timestamp (NULL unless manually invalidated)
  - `reason_invalidated`: String (reason for invalidation, if applicable)

**Storage Security:**

- Only the token hash is stored, never the plaintext token
- Tokens are stored with timing-safe comparison to prevent timing attacks
- Tokens indexed on participant_id and event_id for efficient lookup

### Token Validity

**Expiration Timeline:**

| Scenario                  | Expiration             | Notes                                            |
| ------------------------- | ---------------------- | ------------------------------------------------ |
| After event completion    | Immediate              | Event status change triggers token invalidation  |
| Standard wallet pass      | 90 days                | From creation date                               |
| After check-in completion | Immediate              | Once participant checks in, token expires        |
| Unused for 30 days        | Automatic invalidation | Unused tokens cleaned up to prevent stale access |
| Event cancelled/postponed | Immediate              | All participant tokens invalidated               |

**Expiration Validation:**

- Server validates token against `expires_at` timestamp on every request
- Returns `410 Gone` if token is expired
- Stale tokens cleaned up via daily background job
- `reason_invalidated` field tracks why token became invalid

### Token Rotation & Refresh

**Token Rotation Policy:**

1. **One-time Generation**: Tokens generated once when participant first receives email
2. **Regeneration Triggers**:
   - Explicit user request (participant clicks "Resend wallet pass")
   - Email delivery failure (automatic retry generates new token)
   - Event rescheduling (all tokens regenerated with new expiration)

3. **Rotation Process**:
   - Generate new token following same format
   - Mark old token as `invalidated_at = NOW()` with reason "rotated"
   - Only latest token is valid for a given participant
   - Previous tokens remain in database for audit trail

**Rotation Rate Limiting:**

- Maximum 5 token regenerations per participant per hour
- Prevents abuse of token generation process
- Returns `429 Too Many Requests` if exceeded

### One-Time Use (Optional)

Wallet pass tokens support optional one-time use mode for high-security scenarios:

**Configuration:**

- Set per-event in event settings
- Default: Multi-use tokens (token doesn't expire after single access)
- One-time use mode: Token invalidates after first successful access

**Behavior:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "High-Security Conference",
  "wallet_pass_one_time_use": true,
  "created_at": "2025-11-08T10:00:00Z"
}
```

- After first successful GET request to wallet endpoint:
  - `used_count` increments to 1
  - `accessed_at` timestamp recorded
  - If `one_time_use = true`, token marked as invalidated
  - Subsequent requests return `410 Gone`

**Use Cases:**

- High-value events (VIP access, premium conferences)
- Security-sensitive events (corporate events, government functions)
- Limited distribution (private/invite-only events)

### Security Considerations

**1. Token Transmission:**

- Tokens sent via email with HTTPS links only
- Never included in SMS or other unsecured channels
- Email links include HTTPS protocol (never HTTP)
- Tokens never logged in plain text format

**2. Token Storage:**

- Tokens hashed before storage (HMAC-SHA256)
- Only token hashes compared for validation
- Plaintext tokens never written to logs or error messages
- Tokens encrypted at rest if database supports encryption

**3. Access Control:**

- Token-based authentication (no user credentials required)
- IP address optionally whitelisted per token (configurable)
- Geolocation checks optional (detect unusual access patterns)
- Rate limiting on token validation (10 attempts per minute per token)

**4. Token Leakage Prevention:**

- Tokens rotated immediately if leaked (manual admin action)
- Email delivery logs sanitized (token removed before logging)
- API responses never echo back the token
- Error messages don't reveal whether token exists

**5. Token Invalidation:**

- Tokens invalidated on event completion/cancellation
- Tokens invalidated on participant check-in completion
- Tokens invalidated if participant unregistered from event
- Manual invalidation available to event organizers

### Implementation Notes

**Database Query Pattern:**

```
1. Receive token from URL
2. Hash token using HMAC-SHA256
3. Query: SELECT * FROM wallet_pass_tokens WHERE token_hash = ? AND invalidated_at IS NULL
4. Validate: expires_at > NOW()
5. Check: accessed_at or create timestamp within allowed window
6. If one_time_use: Set invalidated_at = NOW()
7. Update: Set accessed_at = NOW(), used_count += 1
8. Return wallet pass
```

**Error Responses:**

- `400 Bad Request` - Token malformed or missing
- `401 Unauthorized` - Token hash doesn't match any valid token
- `403 Forbidden` - Token valid but rate limited (too many attempts)
- `404 Not Found` - Participant not found
- `410 Gone` - Token expired, invalidated, or already used (one-time use)

### Wallet Pass Error Codes

These error codes will be implemented when wallet pass functionality is added:

#### WALLET_PASS_UNAVAILABLE

- **HTTP Status:** 410 Gone
- **Message:** Wallet pass not available
- **Cause:** Event completed, participant checked in, or pass expired
- **Solution:** No action, wallet pass no longer needed
- **Retry:** No, pass expired or event complete

#### WALLET_TOKEN_INVALID

- **HTTP Status:** 401 Unauthorized
- **Message:** Invalid or expired wallet token
- **Cause:** Wallet token doesn't match or has expired
- **Solution:** Request new wallet pass from participant portal or email
- **Retry:** No, need new token

#### WALLET_GENERATION_FAILED

- **HTTP Status:** 500 Internal Server Error
- **Message:** Failed to generate wallet pass
- **Cause:** Apple PKPass or Google Wallet generation error
- **Solution:** Retry request, contact support if persists
- **Retry:** Yes, with exponential backoff

**Monitoring & Alerts:**

- Alert on: Unusual number of failed token access attempts (>10/minute)
- Alert on: Token accessed from geographically distant locations
- Log: Successful wallet pass downloads with timestamp and platform
- Metric: Track token validity duration and rotation frequency

---

## Wallet Pass Integration

### Apple Wallet (PKPass)

**Features:**

- Automatic updates when event details change
- Lock screen display on event day
- Relevant location-based notifications
- Barcode for scanning

**Pass Contents:**

- Event name and logo
- Participant name
- Event date, time, and location
- QR code for check-in
- Organizer contact information

**Technical Details:**

- Format: PKPass (.pkpass file)
- Signing: PKCS#7 signature with valid certificate
- Distribution: Direct download or email attachment

### Google Wallet

**Features:**

- Real-time event updates
- Location-based reminders
- Integration with Google Calendar
- QR/Barcode scanning

**Pass Contents:**

- Event details (name, date, location)
- Participant information
- QR code for check-in
- Event organizer details

**Technical Details:**

- Format: JWT-based link
- Distribution: Smart Tap link or email

---

## Email Template Enhancements

### Additional Template Variables (Wallet-related)

When wallet integration is implemented, the following variables will be available for email
templates:

**Participant Information:**

```
{wallet_link_apple}    - Apple Wallet download link (token-based)
{wallet_link_google}   - Google Wallet link (token-based)
```

### Updated Request Example (with Wallet Support)

```json
{
  "name": "Event Check-in Information",
  "description": "Email template with direct links for participant access",
  "template_type": "html",
  "subject": "Your {event_name} Check-in Information",
  "body": "<html><body>Hello {participant_name},<br/><br/>You are registered for <strong>{event_name}</strong> on {event_date} at {event_time}.<br/><br/><h3>Check-in</h3><p>Show your QR code at the check-in desk:</p><p><strong>{qr_code_image}</strong></p><p>Or view your QR code: <a href=\"{qr_code_link}\">View QR Code</a></p><p><a href=\"{qr_code_refresh_link}\">Download QR Code</a></p><br/><h3>Digital Tickets</h3><p><a href=\"{wallet_link_apple}\">Add to Apple Wallet</a> | <a href=\"{wallet_link_google}\">Add to Google Wallet</a></p><br/><h3>Check Your Status</h3><p><a href=\"{checkin_status_link}\">Check if you're checked in</a></p><br/><hr/><p>Location: {event_location}</p><p>Questions? Contact us at {organizer_email}</p></body></html>",
  "is_default": false,
  "is_global": false
}
```

---

## Implementation Priorities

### Phase 1: Token Infrastructure

- Create `wallet_pass_tokens` table with security measures
- Implement token generation and validation logic
- Add token storage and hashing mechanisms

### Phase 2: Apple Wallet Integration

- Implement PKPass generation
- Set up certificate signing
- Create wallet download endpoint

### Phase 3: Google Wallet Integration

- Implement JWT token generation
- Set up Google Pay integration
- Create redirect endpoint

### Phase 4: Email Template Enhancement

- Update template variables system
- Add wallet link support
- Test with email templates

### Phase 5: Advanced Features

- One-time use token validation
- Token rotation policies
- Advanced security features (IP whitelisting, geolocation checks)

---

## Related Issues and Dependencies

- Requires: Email sending infrastructure (MVP)
- Requires: Token generation utilities
- Requires: Database schema updates
- Optional: One-time use mode
- Optional: Geolocation validation
