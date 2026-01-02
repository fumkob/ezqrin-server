# Deletion Audit Logs API

## Overview

The Deletion Audit Logs API provides access to comprehensive deletion history for compliance,
security auditing, and troubleshooting. All deletion operations (users, events, participants,
check-ins) are automatically logged with detailed snapshots and metadata.

**Access:** Admin only

**Retention Period:** 3 years

---

## Endpoints

### List Deletion Logs

Retrieve a paginated, filterable list of deletion audit records.

**Endpoint:** `GET /api/v1/deletion-logs`

**Authentication:** Required (Admin only)

**Query Parameters:**

| Parameter     | Type    | Required | Description                                                                     |
| ------------- | ------- | -------- | ------------------------------------------------------------------------------- |
| page          | integer | No       | Page number (default: 1)                                                        |
| per_page      | integer | No       | Items per page (default: 20, max: 100)                                          |
| entity_type   | string  | No       | Filter by entity: `user`, `event`, `participant`, `checkin`, `staff_assignment` |
| entity_id     | UUID    | No       | Filter by specific entity ID                                                    |
| deleted_by    | UUID    | No       | Filter by user who performed deletion                                           |
| deletion_type | string  | No       | Filter by type: `hard`, `soft`, `anonymize`                                     |
| from_date     | string  | No       | ISO 8601 datetime - Start of date range                                         |
| to_date       | string  | No       | ISO 8601 datetime - End of date range                                           |
| sort          | string  | No       | Sort field: `deleted_at`, `entity_type` (default: deleted_at)                   |
| order         | string  | No       | Sort order: `asc`, `desc` (default: desc)                                       |

**Response:** `200 OK`

```json
{
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "entity_type": "event",
      "entity_id": "550e8400-e29b-41d4-a716-446655440000",
      "entity_snapshot": {
        "name": "Tech Conference 2025",
        "start_date": "2025-12-15T09:00:00Z",
        "status": "cancelled",
        "organizer": {
          "id": "660e8400-e29b-41d4-a716-446655440000",
          "name": "John Doe",
          "email": "john@example.com"
        }
      },
      "deleted_by": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "name": "John Doe",
        "email": "john@example.com",
        "role": "organizer"
      },
      "deleted_at": "2025-11-08T15:30:00Z",
      "deletion_type": "hard",
      "deletion_reason": "Event cancelled due to venue unavailability",
      "cascade_effects": {
        "participants_deleted": 150,
        "paid_participants_deleted": 87,
        "checkins_deleted": 45,
        "staff_assignments_removed": 3
      },
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)..."
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "entity_type": "user",
      "entity_id": "660e8400-e29b-41d4-a716-446655440000",
      "entity_snapshot": {
        "name": "Jane Smith",
        "email": "jane@example.com",
        "role": "organizer",
        "created_at": "2024-06-15T10:00:00Z"
      },
      "deleted_by": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "name": "Jane Smith (self-deletion)",
        "role": "organizer"
      },
      "deleted_at": "2025-11-08T14:00:00Z",
      "deletion_type": "anonymize",
      "deletion_reason": "User requested account deletion",
      "cascade_effects": {
        "events_preserved": 5,
        "events_anonymized": 5,
        "staff_assignments_removed": 2
      },
      "ip_address": "203.0.113.50",
      "user_agent": "Chrome/119.0.0.0"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 2,
    "total_pages": 1
  }
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Admin access required
- `400 Bad Request` - Invalid query parameters

---

## Deletion Log Record Structure

### Entity Snapshot

Each deletion log includes a snapshot of the entity before deletion:

**User Snapshot:**

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "role": "organizer",
  "created_at": "2024-01-15T10:00:00Z",
  "events_count": 5
}
```

**Event Snapshot:**

```json
{
  "name": "Tech Conference 2025",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T18:00:00Z",
  "location": "San Francisco Convention Center",
  "status": "cancelled",
  "participants_count": 150,
  "organizer": {
    "id": "...",
    "name": "John Doe"
  }
}
```

**Participant Snapshot:**

```json
{
  "name": "Alice Johnson",
  "email": "alice@example.com",
  "status": "confirmed",
  "payment_status": "unpaid",
  "event": {
    "id": "...",
    "name": "Tech Conference 2025"
  }
}
```

**Check-in Snapshot:**

```json
{
  "participant": {
    "name": "Bob Williams",
    "email": "bob@example.com"
  },
  "event": {
    "name": "Tech Conference 2025"
  },
  "checked_in_at": "2025-12-15T09:15:00Z",
  "checkin_method": "qrcode"
}
```

---

## CASCADE Effects Tracking

Deletion logs record the impact of cascading deletions:

### Event Deletion

```json
{
  "cascade_effects": {
    "participants_deleted": 150,
    "paid_participants_deleted": 87,
    "unpaid_participants_deleted": 63,
    "checkins_deleted": 45,
    "staff_assignments_removed": 3,
    "qr_codes_invalidated": 150
  }
}
```

### User Deletion (Anonymization)

```json
{
  "cascade_effects": {
    "events_preserved": 5,
    "events_anonymized": 5,
    "staff_assignments_removed": 2,
    "pii_fields_anonymized": ["name", "email", "password_hash"]
  }
}
```

### Participant Deletion

```json
{
  "cascade_effects": {
    "checkins_deleted": 1,
    "qr_code_invalidated": true
  }
}
```

---

## Filtering Examples

### By Entity Type

Get all event deletions:

```bash
GET /api/v1/deletion-logs?entity_type=event&per_page=50
```

### By Date Range

Get deletions in the last 30 days:

```bash
GET /api/v1/deletion-logs?from_date=2025-10-08T00:00:00Z&to_date=2025-11-08T23:59:59Z
```

### By User

Get all deletions performed by a specific user:

```bash
GET /api/v1/deletion-logs?deleted_by=660e8400-e29b-41d4-a716-446655440000
```

### By Deletion Type

Get all anonymization operations:

```bash
GET /api/v1/deletion-logs?deletion_type=anonymize
```

### By Specific Entity

Get deletion log for a specific event:

```bash
GET /api/v1/deletion-logs?entity_type=event&entity_id=550e8400-e29b-41d4-a716-446655440000
```

---

## Deletion Types

| Type        | Description                               | Entities                                           |
| ----------- | ----------------------------------------- | -------------------------------------------------- |
| `hard`      | Permanent physical deletion from database | Events, Participants, Check-ins, Staff Assignments |
| `soft`      | Logical deletion with deleted_at flag     | (Reserved for future use)                          |
| `anonymize` | Soft delete with PII anonymization        | Users                                              |

---

## Use Cases

### Compliance Audits

Track all data deletions for GDPR, data protection, and privacy compliance:

```bash
GET /api/v1/deletion-logs?entity_type=user&deletion_type=anonymize&from_date=2025-01-01T00:00:00Z
```

### Security Investigations

Investigate suspicious deletion activities:

```bash
GET /api/v1/deletion-logs?deleted_by=[suspect_user_id]&sort=deleted_at&order=desc
```

### Troubleshooting

Understand why specific data is missing:

```bash
GET /api/v1/deletion-logs?entity_id=[missing_entity_id]
```

### Financial Reconciliation

Track deletions involving paid participants:

```bash
GET /api/v1/deletion-logs?entity_type=event
# Filter results where cascade_effects.paid_participants_deleted > 0
```

---

## Data Retention Policy

**Retention Period:** 3 years from deletion date

**Automatic Cleanup:**

Deletion logs older than 3 years are automatically removed:

```sql
-- Scheduled job runs monthly
DELETE FROM deletion_audit_log
WHERE deleted_at < NOW() - INTERVAL '3 years';
```

**Export Before Cleanup:**

Logs can be exported before automatic cleanup:

```bash
GET /api/v1/deletion-logs?from_date=2022-01-01&to_date=2022-12-31&per_page=1000
# Export to CSV or JSON for archival
```

---

## Security Considerations

### Access Control

- **Admin Only:** Only users with `role='admin'` can access deletion logs
- **No Staff Access:** Staff users cannot view deletion logs
- **No Organizer Access:** Event organizers cannot view deletion logs

### Sensitive Data Handling

**Included in Logs:**

- Entity snapshots (names, emails, event details)
- Deletion reasons
- IP addresses and user agents
- Cascade effects (counts only)

**Excluded from Logs:**

- Password hashes (never logged)
- Full credit card numbers (if applicable)
- JWT tokens or session IDs
- QR code values (only invalidation status)

### Audit Trail Integrity

- Deletion logs are **append-only** (cannot be modified or deleted manually)
- Logs include cryptographic hashes for tamper detection (future enhancement)
- System administrators cannot bypass logging

---

## Error Codes

| Code                     | Message               | Description                        |
| ------------------------ | --------------------- | ---------------------------------- |
| `DELETION_LOG_FORBIDDEN` | Admin access required | Only admins can view deletion logs |
| `INVALID_DATE_RANGE`     | Invalid date range    | from_date must be before to_date   |
| `INVALID_ENTITY_TYPE`    | Invalid entity type   | Must be valid entity type          |
| `INVALID_DELETION_TYPE`  | Invalid deletion type | Must be: hard, soft, or anonymize  |

---

## Example Responses

### Successful Query

**Request:**

```bash
GET /api/v1/deletion-logs?entity_type=user&per_page=5
Authorization: Bearer [admin_access_token]
```

**Response:** `200 OK`

```json
{
  "data": [
    {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "entity_type": "user",
      "entity_id": "660e8400-e29b-41d4-a716-446655440000",
      "entity_snapshot": {
        "name": "John Doe",
        "email": "john@example.com",
        "role": "organizer"
      },
      "deleted_by": {
        "id": "admin-user-id",
        "name": "System Admin",
        "role": "admin"
      },
      "deleted_at": "2025-11-08T15:30:00Z",
      "deletion_type": "anonymize",
      "deletion_reason": "User requested GDPR data deletion",
      "cascade_effects": {
        "events_preserved": 3,
        "staff_assignments_removed": 1
      }
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 5,
    "total": 1,
    "total_pages": 1
  }
}
```

### Forbidden Access

**Request:**

```bash
GET /api/v1/deletion-logs
Authorization: Bearer [organizer_access_token]
```

**Response:** `403 Forbidden`

```json
{
  "type": "https://api.ezqrin.com/problems/forbidden",
  "title": "Forbidden",
  "status": 403,
  "detail": "Admin access required to view deletion logs",
  "instance": "/api/v1/deletion-logs",
  "code": "DELETION_LOG_FORBIDDEN"
}
```

---

## Related Documentation

- [Users API](./users.md)
- [Events API](./events.md)
- [Participants API](./participants.md)
- [Security Design](../architecture/security.md)
- [Database Design](../architecture/database.md)
