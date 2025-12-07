# Users API

## Overview

The Users API manages user account lifecycle including retrieval, updates, and deletion. This
includes user deletion with PII anonymization to comply with data protection best practices.

**Note:** For user registration and authentication, see
[Authentication API](./authentication.md#register-user).

## Endpoints

### List Users

Retrieve a paginated list of all users in the system.

**Endpoint:** `GET /api/v1/users`

**Authentication:** Required (Admin only)

**Authorization:**

- **Admin**: Can list all users

**Headers:**

```
Authorization: Bearer <access_token>
```

**Query Parameters:**

| Parameter | Type   | Required | Description                                    |
| --------- | ------ | -------- | ---------------------------------------------- |
| page      | int    | No       | Page number (default: 1)                       |
| per_page  | int    | No       | Items per page (default: 20, max: 100)         |
| role      | string | No       | Filter by role: `organizer`, `staff`, `admin`  |
| search    | string | No       | Search by name or email (minimum 3 characters) |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "email": "john@example.com",
        "name": "John Doe",
        "role": "organizer",
        "created_at": "2025-11-01T10:00:00Z"
      },
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "email": "jane@example.com",
        "name": "Jane Smith",
        "role": "admin",
        "created_at": "2025-11-02T14:30:00Z"
      }
    ]
  },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 47,
    "total_pages": 3
  },
  "message": "Users retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required or token invalid
- `403 Forbidden` - Admin role required

---

### Get User

Retrieve details of a specific user.

**Endpoint:** `GET /api/v1/users/:id`

**Authentication:** Required

**Authorization:**

- **Self**: Any authenticated user can retrieve their own information
- **Admin**: Can retrieve any user's information

**Headers:**

```
Authorization: Bearer <access_token>
```

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | User ID     |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "john@example.com",
      "name": "John Doe",
      "role": "organizer",
      "created_at": "2025-11-01T10:00:00Z"
    }
  },
  "message": "User retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required or token invalid
- `403 Forbidden` - Not authorized to view this user
- `404 Not Found` - User not found or deleted

---

### Update User

Update user account information.

**Endpoint:** `PUT /api/v1/users/:id` or `PATCH /api/v1/users/:id`

**Authentication:** Required

**Authorization:**

- **Self**: Any authenticated user can update their own information
- **Admin**: Can update any user's information

**Headers:**

```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | User ID     |

**Request Body:**

```json
{
  "name": "John Updated Doe",
  "email": "john.new@example.com",
  "password": "NewSecurePassword123!"
}
```

**Request Fields:**

| Field    | Type   | Required | Description                                                          |
| -------- | ------ | -------- | -------------------------------------------------------------------- |
| name     | string | No       | Full name (1-255 characters)                                         |
| email    | string | No       | Valid email address (must be unique)                                 |
| password | string | No       | Minimum 8 characters, must include uppercase, lowercase, and numbers |

**Note:** At least one field must be provided for update.

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "john.new@example.com",
      "name": "John Updated Doe",
      "role": "organizer",
      "created_at": "2025-11-01T10:00:00Z",
      "updated_at": "2025-11-08T16:45:00Z"
    }
  },
  "message": "User updated successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or validation failed
- `401 Unauthorized` - Authentication required or token invalid
- `403 Forbidden` - Not authorized to update this user
- `404 Not Found` - User not found or deleted
- `409 Conflict` - Email already in use by another user

---

### Delete User

Delete a user account and anonymize all personal information while preserving event ownership.

**Endpoint:** `DELETE /api/v1/users/:id`

**Authentication:** Required (Admin or Self-deletion)

**Authorization:**

- **Admin**: Can delete any user account
- **User**: Can only delete their own account (self-deletion)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | User ID     |

**Request Body:**

```json
{
  "reason": "User requested account deletion",
  "confirm": true
}
```

**Request Fields:**

| Field   | Type    | Required | Description                                   |
| ------- | ------- | -------- | --------------------------------------------- |
| reason  | string  | Yes      | Reason for deletion (max 500 characters)      |
| confirm | boolean | Yes      | Confirmation flag (must be `true` to proceed) |

**Response:** `200 OK`

```json
{
  "success": true,
  "message": "User account deleted and anonymized successfully",
  "data": {
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "anonymized": true,
    "deletion_type": "soft_delete_with_anonymization",
    "events_count": 5,
    "events_status": "preserved with anonymized owner",
    "staff_assignments_removed": 3,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or missing confirmation
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to delete this user
- `404 Not Found` - User not found
- `409 Conflict` - User has active events (see below)

---

### User Deletion Process

**Step 1: Validation**

The system validates that the user can be deleted:

1. User must exist and not already be deleted
2. User must not have active events (status: `draft`, `published`, `ongoing`)
3. Request must include valid confirmation

**Step 2: PII Anonymization**

All personally identifiable information is irreversibly anonymized:

```json
{
  "name": "John Doe" → "Deleted User a3f5b2c1",
  "email": "john@example.com" → "deleted_a3f5b2c1@anonymized.local",
  "password_hash": "***" → "[random hash - login disabled]"
}
```

**Step 3: Related Data Handling**

- **Events owned by user**: Preserved with anonymized owner
- **Staff assignments**: Removed (access revoked)
- **Check-ins performed by user**: Preserved with anonymized name in history

**Step 4: Audit Logging**

Deletion is recorded in `deletion_audit_log` table with:

- User snapshot before anonymization
- Deletion reason and timestamp
- Deleted by (admin or self)
- IP address and user agent

---

### Active Events Validation

**Error Response when user has active events:**

```json
{
  "success": false,
  "error": "USER_HAS_ACTIVE_EVENTS",
  "message": "User has active events. Complete or delete events before deleting user account.",
  "data": {
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "active_events_count": 3,
    "active_events": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Tech Conference 2025",
        "status": "published",
        "start_date": "2025-12-15T09:00:00Z",
        "participants_count": 150
      },
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "Workshop Series",
        "status": "draft",
        "start_date": "2026-01-10T10:00:00Z",
        "participants_count": 25
      }
    ],
    "suggestion": "Delete or complete events first, then retry user deletion"
  }
}
```

**Resolution Steps:**

1. List all events owned by the user
2. For each active event:
   - Complete the event (change status to `completed`)
   - Cancel the event (change status to `cancelled`)
   - Delete the event using `DELETE /api/v1/events/:id`
3. Retry user deletion after all events are resolved

---

## User Deletion States

### Before Deletion

```sql
SELECT * FROM users WHERE id = '660e8400...';
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "email": "john@example.com",
  "name": "John Doe",
  "role": "organizer",
  "deleted_at": null,
  "is_anonymized": false
}
```

### After Deletion (Anonymized)

```sql
SELECT * FROM users WHERE id = '660e8400...';
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "email": "deleted_a3f5b2c1@anonymized.local",
  "name": "Deleted User a3f5b2c1",
  "role": "organizer",
  "deleted_at": "2025-11-08T15:30:00Z",
  "deleted_by": "660e8400-e29b-41d4-a716-446655440000",
  "is_anonymized": true
}
```

**Important Notes:**

- Anonymization is **irreversible** - original data cannot be recovered
- User cannot log in after deletion (password hash is invalidated)
- Events remain accessible with "Deleted User" shown as organizer
- Email format ensures uniqueness constraint is maintained

---

## Data Retention Policy

**Soft-Deleted Users:**

- Anonymized user records remain in database indefinitely
- Deletion audit log retained for 3 years
- Events created by deleted users remain accessible

**Future Enhancement:**

Scheduled job to permanently delete anonymized users after 90 days:

```sql
-- Planned for future implementation
DELETE FROM users
WHERE deleted_at IS NOT NULL
  AND deleted_at < NOW() - INTERVAL '90 days'
  AND is_anonymized = true;
```

---

## Security Considerations

### Authentication

- Admin users can delete any user account
- Regular users can only delete their own account
- Staff users cannot delete accounts (requires organizer or admin role)

### Audit Trail

All user deletions are logged with:

- User ID and anonymized snapshot
- Deletion reason
- Who performed the deletion (admin or self)
- Timestamp and IP address
- Related data impact (events count, staff assignments removed)

### Data Protection

- PII is completely removed (not just hidden)
- Anonymization uses cryptographically secure random tokens
- Email uniqueness preserved with `deleted_[token]@anonymized.local` format
- Password hash replaced with random value (login permanently disabled)

---

## Error Codes

| Code                       | Message                                    | Description                                   |
| -------------------------- | ------------------------------------------ | --------------------------------------------- |
| `USER_NOT_FOUND`           | User not found                             | User ID does not exist                        |
| `USER_ALREADY_DELETED`     | User account already deleted               | User was previously deleted and anonymized    |
| `USER_HAS_ACTIVE_EVENTS`   | User has active events                     | Must resolve events before deletion           |
| `USER_DELETION_FORBIDDEN`  | Not authorized to delete this user         | Insufficient permissions                      |
| `INVALID_CONFIRMATION`     | Confirmation required                      | Request must include `confirm: true`          |
| `DELETION_REASON_REQUIRED` | Deletion reason required                   | Must provide reason for deletion              |
| `SELF_DELETION_ADMIN_ONLY` | Cannot delete admin user via self-deletion | Contact another admin to delete admin account |

---

## Examples

### Successful User Deletion

**Request:**

```bash
DELETE /api/v1/users/660e8400-e29b-41d4-a716-446655440000
Authorization: Bearer [access_token]
Content-Type: application/json

{
  "reason": "User requested GDPR data deletion",
  "confirm": true
}
```

**Response:**

```json
{
  "success": true,
  "message": "User account deleted and anonymized successfully",
  "data": {
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "anonymized": true,
    "deletion_type": "soft_delete_with_anonymization",
    "events_count": 5,
    "events_status": "preserved with anonymized owner",
    "staff_assignments_removed": 3,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

### Failed Deletion - Active Events

**Request:**

```bash
DELETE /api/v1/users/660e8400-e29b-41d4-a716-446655440000
Authorization: Bearer [access_token]
Content-Type: application/json

{
  "reason": "User requested account deletion",
  "confirm": true
}
```

**Response:**

```json
{
  "success": false,
  "error": "USER_HAS_ACTIVE_EVENTS",
  "message": "User has 3 active events. Complete or delete events before deleting user account.",
  "data": {
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "active_events_count": 3,
    "active_events": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Tech Conference 2025",
        "status": "published"
      }
    ]
  }
}
```

---

## Related Documentation

- [Authentication API](./authentication.md)
- [Events API](./events.md)
- [Deletion Audit Logs](./deletion_logs.md)
- [Security Design](../architecture/security.md)
- [Database Design](../architecture/database.md)
