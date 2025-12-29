# ezQRin API Documentation

Complete API reference for the ezQRin event management and QR code distribution system.

---

## API Specification (OpenAPI)

**Single Source of Truth:**

The ezQRin API is defined using **OpenAPI 3.0+ specification** as the **Single Source of Truth
(SSOT)** for all API contracts. This specification serves as the authoritative source from which
server code, client SDKs, and documentation are generated.

**OpenAPI Specification Files:**

The specification is organized as modular YAML files for improved maintainability:

```
api/
├── openapi.yaml              # Main entry point (aggregator)
├── schemas/                  # Reusable schemas (entities, enums, responses)
├── components/               # Reusable components (security, responses, parameters)
└── paths/                    # API endpoint definitions (by module)
```

**For detailed structure and usage**, see [`api/README.md`](../../api/README.md)

**Code Generation:**

We use `oapi-codegen` to automatically generate:

- **Type-safe DTOs** - Request/Response models with validation
- **Gin Server Interfaces** - HTTP handlers with type safety
- **API Client** - Client libraries for testing and integration
- **Validation Schemas** - Input validation logic

**Benefits of API-First Approach:**

- ✅ **Consistency:** Documentation and implementation always in sync
- ✅ **Type Safety:** Compile-time guarantees for API contracts
- ✅ **Rapid Development:** Reduced boilerplate through code generation
- ✅ **Client SDKs:** Automatic generation for frontend/mobile teams

For detailed information on API-first development workflow, see
[Architecture Documentation](../architecture/overview.md#api-first-development-with-openapi).

---

## Quick Navigation

### Core Workflows

**🎫 Event Registration Flow**

```
Events API → Participants API → QR Code API → Check-in API
```

[Create Event](./events.md) → [Register Participant](./participants.md) →
[Send QR Code](./qrcode.md) → [Record Check-in](./checkin.md)

**👥 Participant Self-Service Flow (Token-Based)**

```
Email Link (Token) → Download QR Code → Add to Wallet → Check Status
```

[QR Code Retrieval](./qrcode.md#participant-portal-access) →
[Wallet Pass](./qrcode.md#get-wallet-pass) →
[Check-in Status](./qrcode.md#participant-portal-access)

**⚙️ Organizer Management Flow**

```
Login → Create Event → Manage Participants → Send Communications → Monitor Check-ins
```

[Login](./authentication.md#login) → [Create Event](./events.md#create-event) →
[Manage Participants](./participants.md) → [Send QR Codes](./qrcode.md#send-qr-codes-via-email) →
[View Check-ins](./checkin.md#list-checkins)

---

## API Documentation Map

### 1. **Authentication** - `authentication.md`

Organizer/Staff authentication and authorization

- [Register User](./authentication.md#register-user)
- [Login](./authentication.md#login)
- [Refresh Token](./authentication.md#refresh-token)
- [Logout](./authentication.md#logout)
- User roles (Admin, Organizer, Staff)
- JWT token management

**Related:** [Participant Token Access](./qrcode.md#participant-portal-access) (token-based
participant access), [Users](./users.md) (user account management)

---

### 2. **Users** - `users.md`

User account management and lifecycle

- [List Users](./users.md#list-users) - Get all users (Admin only)
- [Get User](./users.md#get-user) - Retrieve user details (Self or Admin)
- [Update User](./users.md#update-user) - Modify user profile (Self or Admin)
- [Delete User Account](./users.md#delete-user) - PII anonymization and soft delete
- User deletion validation and constraints
- Event ownership preservation
- Data protection compliance

**Related:** [Authentication](./authentication.md) (user registration and login),
[Events](./events.md) (event ownership), [Deletion Logs](./deletion_logs.md) (audit trail)

---

### 3. **Events** - `events.md`

Event lifecycle management

- Create, read, update, delete events
- Event details and settings
- Event status management
- Staff assignment
- Event metadata and configuration
- Enhanced deletion with validation rules

**Related:** [Participants](./participants.md) (register participants for events),
[QR Codes](./qrcode.md) (send QR codes to participants), [Deletion Logs](./deletion_logs.md)
(deletion audit trail)

---

### 4. **Participants** - `participants.md`

Participant registration and management

- Register individual or bulk participants
- Retrieve participant information
- Update participant status and metadata
- Automatic QR code generation
- Alternative email routing (qr_email field)
- Enhanced deletion with payment protection

**Related:** [Events](./events.md) (participants belong to events), [QR Codes](./qrcode.md) (each
participant gets QR code), [Check-in](./checkin.md) (participants check in),
[Deletion Logs](./deletion_logs.md) (deletion audit trail)

---

### 5. **QR Codes** - `qrcode.md`

QR code generation, distribution, and wallet pass integration

- Send QR codes via email
- Async job management for large batches
- QR code formats and specifications
- Wallet pass integration (Apple/Google)
- **Participant self-service access (token-based)** - Direct links for participant access
- Secure token management
- Email template customization
- Security and privacy considerations

**Related:** [Participants](./participants.md) (QR codes assigned to participants),
[Participant Portal Access](./qrcode.md#participant-portal-access) (token-based participant access),
[Check-in](./checkin.md) (QR codes scanned at check-in)

---

### 6. **Check-in** - `checkin.md`

Event check-in management

- Record participant check-ins
- QR code/barcode scanning
- Check-in status queries
- Staff check-in authority
- Bulk check-in operations

**Related:** [QR Codes](./qrcode.md) (QR codes scanned for check-in),
[Participants](./participants.md) (check-in records for participants),
[Deletion Logs](./deletion_logs.md) (check-in deletion audit)

---

### 7. **Deletion Audit Logs** - `deletion_logs.md`

Comprehensive deletion audit trail for compliance and troubleshooting

- [List Deletion Logs](./deletion_logs.md#list-deletion-logs) - Query deletion history
- Entity snapshots and cascade effects tracking
- Deletion type filtering (hard, soft, anonymize)
- 3-year retention policy
- Admin-only access

**Related:** [Users](./users.md) (user deletion), [Events](./events.md) (event deletion),
[Participants](./participants.md) (participant deletion), [Security](../architecture/security.md)
(audit logging)

---

### 8. **Rate Limiting** - `rate_limits.md`

API rate limiting strategy and thresholds

- Authentication limits (login, registration)
- Email operation limits
- Data retrieval limits
- Check-in operation limits
- Job polling limits
- Burst handling and rate limit headers
- Exemptions for trusted partners

**Referenced by:** [Authentication](./authentication.md), [QR Codes](./qrcode.md),
[Check-in](./checkin.md)

---

### 9. **Testing** - `.../testing.md`

API testing and sandbox mode

- Sandbox environment setup
- Test event creation
- Mock email responses
- QR code validation testing
- Check-in simulation
- Rate limit testing
- Webhook testing
- Testing best practices and checklist

**Related:** All API documents (can be tested in sandbox mode with `X-Test-Mode: true`)

---

### 10. **Schemas** - `schemas.md`

Common data models and response formats

- Standard response envelope
- Error response format
- Pagination structure
- Common object definitions
- HTTP status codes

**Referenced by:** All API documents

---

## Document Cross-References

### By Feature

**Email Management**

- [Send QR Codes](./qrcode.md#send-qr-codes-via-email) - QR Code API
- [Email Templates](./qrcode.md#email-templates) - QR Code API
- [Email Template Customization](./qrcode.md#email-template-customization) - QR Code API
- [Rate Limits: Email Operations](./rate_limits.md#email-operations) - Rate Limits

**QR Code Management**

- [QR Code Generation](./participants.md) - Automatic at participant registration (Participants API)
- [QR Code Distribution](./qrcode.md#send-qr-codes-via-email) - QR Code API
- [QR Code Format & Specs](./qrcode.md#qr-code-specifications) - QR Code API
- [QR Code Retrieval](./qrcode.md#get-individual-qr-code) - Organizer/Staff access (QR Code API)
- [Participant QR Code Access (Token-Based)](./qrcode.md#qr-code-retrieval-by-participant) - Direct
  links for participants (QR Code API)
- [QR Code in Testing](../testing.md#test-qr-codes) - Testing Guide

**Wallet Pass Management**

- [Wallet Pass Integration](./qrcode.md#wallet-pass-integration) - QR Code API
- [Token Management](./qrcode.md#wallet-pass-token-management) - QR Code API (includes participant
  access)
- [Wallet Pass Endpoints](./qrcode.md#get-wallet-pass) - QR Code API
- [Participant Wallet Access (Token-Based)](./qrcode.md#get-wallet-pass) - Direct links for
  participants

**Check-in Operations**

- [Record Check-in](./checkin.md#record-checkin) - Check-in API
- [Check-in Status](./checkin.md#get-checkin-status) - Check-in API
- [Participant Check-in Status (Token-Based)](./qrcode.md#check-in-status) - Direct links for
  participants (QR Code API)
- [Check-in Rate Limits](./rate_limits.md#check-in-operations) - Rate Limits
- [Check-in Testing](../testing.md#check-in-simulation) - Testing Guide

**Authentication & Authorization**

- [User Registration](./authentication.md#register-user) - Authentication API (organizers/staff)
- [User Login](./authentication.md#login) - Authentication API (organizers/staff)
- [Token Refresh](./authentication.md#refresh-token) - Authentication API (organizers/staff)
- [Participant Token Access](./qrcode.md#participant-portal-access) - Token-based participant access
  via secure URLs (QR Code API)
- [Authorization Model](./authentication.md#authorization) - Authentication API
- [Auth Rate Limits](./rate_limits.md#authentication--security-operations) - Rate Limits

**Async Operations**

- [Email Job Queueing](./qrcode.md#send-qr-codes-via-email) - QR Code API
- [Job Status Tracking](./qrcode.md#get-email-job-status) - QR Code API
- [Job Polling Rate Limits](./rate_limits.md#jobasync-operations) - Rate Limits

---

## Architecture & Design Docs

**System Architecture:** See [System Architecture Overview](../architecture/overview.md)

- Architectural patterns
- Technology stack
- Directory structure
- Data flow
- Design patterns

**Database Design:** See [Database Design](../architecture/database.md)

- Complete schema reference
- Relationships and constraints
- Indexing strategy
- Sample queries

**Security Design:** See [Security Design](../architecture/security.md)

- Authentication flow
- Authorization model
- Data protection
- Security considerations

---

## Key Concepts

### Participant Flow (Token-Based)

1. **Registration**: Participant registered for event via [Participants API](./participants.md)
   - Automatic QR code generation
   - Secure access token generated (90-day validity)
   - Optional qr_email field for alternative delivery
2. **Notification**: QR code sent via [QR Code API](./qrcode.md)
   - Email delivery with direct secure links (sync <100 or async ≥100 participants)
   - Links include access tokens for:
     - Direct QR code retrieval: `/participants/:pid/qrcode?token=...`
     - Wallet pass download: `/participants/:pid/wallet?token=...`
     - Check-in status: `/participants/:pid/checkin-status?token=...`
   - Wallet pass option (Apple/Google)
3. **Self-Service**: Participant accesses resources via secure email links
   - No authentication required (token validates access)
   - Download QR code (PNG/SVG/JSON formats)
   - Add to Apple/Google Wallet
   - Check check-in status
   - See [QR Code API - Participant Portal Access](./qrcode.md#participant-portal-access) for
     details
4. **Check-in**: Staff records check-in via [Check-in API](./checkin.md)
   - Scan QR code or manually enter ID
   - System records check-in timestamp
   - Token becomes invalid (one-time use optional)

### Token Types

**JWT Access Tokens** (15 minutes)

- Used for: Organizer/staff API authentication
- Issued by: [Login endpoint](./authentication.md#login)
- Stored in: Authorization header

**Participant Access Tokens** (90 days)

- Used for: Secure participant access to QR codes, wallet passes, and check-in status
- Issued by: System during participant registration
- Stored in: URL query parameter
- Secure management: See [Token Management](./qrcode.md#wallet-pass-token-management)
- Format: `wallet_{event_id}_{participant_id}_{random_token}_{timestamp_hash}`

---

## Common Tasks

### Send QR Codes to Participants

```
1. Create event [Events API](./events.md#create-event)
2. Register participants [Participants API](./participants.md#add-participant)
3. Send QR codes [QR Code API](./qrcode.md#send-qr-codes-via-email)
4. Monitor job [QR Code API](./qrcode.md#get-email-job-status) (if ≥100 participants)
5. Verify email content [Testing API](../testing.md#mock-email-responses) (sandbox mode)
```

### Manage Event Check-in

```
1. Setup event staff [Events API](./events.md#assign-staff)
2. At event: Record check-ins [Check-in API](./checkin.md#record-checkin)
3. Query check-in status [Check-in API](./checkin.md#get-checkin-status)
4. Generate reports [Events API](./events.md#get-event-statistics)
```

### Customize Email Templates

```
1. Create custom template [QR Code API](./qrcode.md#email-template-customization)
2. Set variables [QR Code API](./qrcode.md#available-template-variables)
3. Preview template [QR Code API](./qrcode.md#preview-template)
4. Use in email send [QR Code API](./qrcode.md#send-qr-codes-via-email)
```

### Test Before Production

```
1. Enable sandbox mode [Testing API](../testing.md#sandbox-environment)
2. Create test event [Testing API](../testing.md#test-event-creation)
3. Send test emails [Testing API](../testing.md#mock-email-responses)
4. Simulate check-ins [Testing API](../testing.md#check-in-simulation)
5. Cleanup test data [Testing API](../testing.md#cleanup--teardown)
```

---

## Rate Limiting Guide

**See:** [Rate Limiting Strategy](./rate_limits.md)

Quick limits by operation type:

| Operation         | Limit | Window             |
| ----------------- | ----- | ------------------ |
| Login attempts    | 5     | 15 minutes per IP  |
| Email sending     | 100   | 1 minute per event |
| QR code retrieval | 50    | 1 minute per event |
| Check-in          | 50    | 1 minute per event |
| Job polling       | 10    | 1 minute per job   |

---

## Error Handling

**See:** [Schemas: Error Responses](./schemas.md#error-response-format)

All errors follow standard format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

**Common error codes:**

- `AUTH_INVALID_CREDENTIALS` - Login failed
- `RATE_LIMIT_EXCEEDED` - Rate limit exceeded
- `EVENT_NOT_FOUND` - Event doesn't exist
- `PARTICIPANT_NOT_FOUND` - Participant doesn't exist
- `ALREADY_CHECKED_IN` - Participant already checked in

---

## Support & Resources

**Questions about:**

- **API Usage**: Refer to specific endpoint documentation
- **Architecture & Design**: See [Architecture Overview](../architecture/overview.md)
- **Database Schema**: See [Database Design](../architecture/database.md)
- **Security**: See [Security Design](../architecture/security.md)
- **Testing**: See [Testing Guide](../testing.md)
- **Rate Limits**: See [Rate Limiting Strategy](./rate_limits.md)

**Contact:** support@ezqrin.com
