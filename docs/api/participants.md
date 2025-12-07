# Participants API

## Overview

The Participants API manages event attendees, including individual registration, bulk CSV imports,
and participant information updates.

## Endpoints

### Add Participant

Register a single participant for an event.

**Endpoint:** `POST /api/v1/events/:id/participants`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request Body:**

```json
{
  "name": "Jane Smith",
  "email": "jane@example.com",
  "qr_email": "jane.work@example.com",
  "employee_id": "EMP001",
  "phone": "+1-555-0123",
  "status": "confirmed",
  "payment_status": "paid",
  "payment_amount": 150.0,
  "payment_date": "2025-11-08T12:30:00Z",
  "metadata": {
    "company": "Tech Corp",
    "role": "Software Engineer",
    "dietary_restrictions": "Vegetarian"
  }
}
```

**Request Fields:**

| Field          | Type   | Required | Description                                                                                  |
| -------------- | ------ | -------- | -------------------------------------------------------------------------------------------- |
| name           | string | Yes      | Participant full name (1-255 characters)                                                     |
| email          | string | Yes      | Valid email address                                                                          |
| qr_email       | string | No       | Alternative email for QR code distribution (if NULL, uses primary email)                     |
| employee_id    | string | No       | Employee or staff ID (1-255 characters)                                                      |
| phone          | string | No       | Phone number in E.164 format                                                                 |
| status         | string | No       | Participation status: `tentative`, `confirmed`, `cancelled`, `declined` (default: tentative) |
| payment_status | string | No       | Payment status: `unpaid`, `paid` (default: unpaid)                                           |
| payment_amount | number | No       | Payment amount (decimal with 2 places), nullable                                             |
| payment_date   | string | No       | Payment date in ISO 8601 format, nullable                                                    |
| metadata       | object | No       | Custom key-value data (max 10KB)                                                             |

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Jane Smith",
    "email": "jane@example.com",
    "qr_email": "jane.work@example.com",
    "employee_id": "EMP001",
    "phone": "+1-555-0123",
    "status": "confirmed",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "payment_date": "2025-11-08T12:30:00Z",
    "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
    "qr_code_generated_at": "2025-11-08T10:00:00Z",
    "metadata": {
      "company": "Tech Corp",
      "role": "Software Engineer",
      "dietary_restrictions": "Vegetarian"
    },
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T10:00:00Z"
  },
  "message": "Participant added successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to add participants to this event
- `404 Not Found` - Event not found
- `409 Conflict` - Email already registered for this event

---

### Import Participants (CSV)

Bulk import participants from a CSV file.

**Endpoint:** `POST /api/v1/events/:id/participants/import`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request:** `multipart/form-data`

```
Content-Type: multipart/form-data

file: participants.csv
skip_duplicates: true
send_emails: false
```

**CSV Format:**

```csv
name,email,employee_id,phone,status,payment_status,payment_amount,payment_date,metadata
Jane Smith,jane@example.com,EMP001,+1-555-0123,confirmed,paid,150.00,2025-11-08T12:30:00Z,"{""company"":""Tech Corp""}"
John Doe,john@example.com,EMP002,+1-555-0124,tentative,unpaid,,,"{""company"":""StartupXYZ""}"
```

**CSV Fields:**

| Field          | Required | Description                                    |
| -------------- | -------- | ---------------------------------------------- |
| name           | Yes      | Participant name                               |
| email          | Yes      | Email address                                  |
| employee_id    | No       | Employee or staff ID                           |
| phone          | No       | Phone number                                   |
| status         | No       | Participation status (default: tentative)      |
| payment_status | No       | Payment status: unpaid, paid (default: unpaid) |
| payment_amount | No       | Payment amount as decimal number               |
| payment_date   | No       | Payment date in ISO 8601 format                |
| metadata       | No       | JSON string of custom data                     |

**Query Parameters:**

| Parameter       | Type    | Default | Description                          |
| --------------- | ------- | ------- | ------------------------------------ |
| skip_duplicates | boolean | false   | Skip rows with duplicate emails      |
| send_emails     | boolean | false   | Send QR codes via email after import |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "imported_count": 148,
    "skipped_count": 2,
    "failed_count": 0,
    "errors": [],
    "skipped_rows": [
      {
        "row": 5,
        "email": "duplicate@example.com",
        "reason": "Email already exists for this event"
      }
    ]
  },
  "message": "Import completed successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid CSV format or missing required fields
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to import to this event
- `404 Not Found` - Event not found
- `413 Payload Too Large` - CSV file exceeds size limit (10MB)

---

### List Participants

Retrieve a paginated list of event participants.

**Endpoint:** `GET /api/v1/events/:id/participants`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Query Parameters:**

| Parameter      | Type    | Required | Description                                                         |
| -------------- | ------- | -------- | ------------------------------------------------------------------- |
| page           | integer | No       | Page number (default: 1)                                            |
| per_page       | integer | No       | Items per page (default: 20, max: 100)                              |
| status         | string  | No       | Filter by status: `tentative`, `confirmed`, `cancelled`, `declined` |
| payment_status | string  | No       | Filter by payment status: `unpaid`, `paid`                          |
| checked_in     | boolean | No       | Filter by check-in status (true/false)                              |
| search         | string  | No       | Search in name and email                                            |
| sort           | string  | No       | Sort field: `name`, `email`, `created_at` (default: created_at)     |
| order          | string  | No       | Sort order: `asc`, `desc` (default: desc)                           |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "employee_id": "EMP001",
      "name": "Jane Smith",
      "email": "jane@example.com",
      "phone": "+1-555-0123",
      "status": "confirmed",
      "payment_status": "paid",
      "payment_amount": 150.0,
      "payment_date": "2025-11-08T12:30:00Z",
      "checked_in": true,
      "checked_in_at": "2025-12-15T09:15:00Z",
      "created_at": "2025-11-08T10:00:00Z",
      "updated_at": "2025-11-08T10:00:00Z"
    }
  ],
  "message": "Participants retrieved successfully",
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event not found

---

### Get Participant

Retrieve detailed information about a specific participant.

**Endpoint:** `GET /api/v1/events/:id/participants/:pid`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| id        | UUID | Event ID       |
| pid       | UUID | Participant ID |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "employee_id": "EMP001",
    "name": "Jane Smith",
    "email": "jane@example.com",
    "qr_email": "jane.work@example.com",
    "phone": "+1-555-0123",
    "status": "confirmed",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "payment_date": "2025-11-08T12:30:00Z",
    "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
    "qr_code_generated_at": "2025-11-08T10:00:00Z",
    "metadata": {
      "company": "Tech Corp",
      "role": "Software Engineer",
      "dietary_restrictions": "Vegetarian"
    },
    "checked_in": true,
    "checked_in_at": "2025-12-15T09:15:00Z",
    "checked_in_by": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "Admin User"
    },
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T10:00:00Z"
  },
  "message": "Participant retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event or participant not found

---

### Update Participant (Full)

Update participant information. All fields must be provided (full replacement).

**Endpoint:** `PUT /api/v1/events/:id/participants/:pid`

**Authentication:** Required (Event owner or Admin)

**Note:** For partial updates, use the PATCH endpoint instead.

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| id        | UUID | Event ID       |
| pid       | UUID | Participant ID |

**Request Body:**

```json
{
  "name": "Jane Smith-Johnson",
  "email": "jane.johnson@example.com",
  "qr_email": "jane.work@example.com",
  "employee_id": "EMP001",
  "phone": "+1-555-0125",
  "status": "confirmed",
  "payment_status": "paid",
  "payment_amount": 150.0,
  "payment_date": "2025-11-08T12:30:00Z",
  "metadata": {
    "company": "New Tech Corp",
    "role": "Senior Engineer"
  }
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Jane Smith-Johnson",
    "email": "jane.johnson@example.com",
    "qr_email": "jane.work@example.com",
    "employee_id": "EMP001",
    "phone": "+1-555-0125",
    "status": "confirmed",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "payment_date": "2025-11-08T12:30:00Z",
    "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
    "metadata": {
      "company": "New Tech Corp",
      "role": "Senior Engineer"
    },
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T11:00:00Z"
  },
  "message": "Participant updated successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to update this participant
- `404 Not Found` - Event or participant not found
- `409 Conflict` - Email already used by another participant in this event

---

### Update Participant (Partial)

Partially update participant information. Only provided fields will be updated.

**Endpoint:** `PATCH /api/v1/events/:id/participants/:pid`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| id        | UUID | Event ID       |
| pid       | UUID | Participant ID |

**Request Body:**

All fields are optional. Only provided fields will be updated.

```json
{
  "name": "Jane Smith-Johnson",
  "email": "jane@example.com",
  "qr_email": "jane.work@example.com",
  "employee_id": "EMP001",
  "status": "confirmed",
  "payment_status": "paid",
  "payment_amount": 150.0,
  "payment_date": "2025-11-08T12:30:00Z",
  "metadata": {
    "company": "New Tech Corp",
    "role": "Senior Engineer"
  }
}
```

**Available Fields:**

| Field          | Type   | Description                                                             |
| -------------- | ------ | ----------------------------------------------------------------------- |
| name           | string | Participant full name (1-255 characters)                                |
| email          | string | Valid email address                                                     |
| qr_email       | string | Alternative email for QR code distribution (nullable)                   |
| employee_id    | string | Employee or staff ID (1-255 characters)                                 |
| phone          | string | Phone number in E.164 format                                            |
| status         | string | Participation status: `tentative`, `confirmed`, `cancelled`, `declined` |
| payment_status | string | Payment status: `unpaid`, `paid`                                        |
| payment_amount | number | Payment amount (decimal with 2 places), nullable                        |
| payment_date   | string | Payment date in ISO 8601 format, nullable                               |
| metadata       | object | Custom key-value data (max 10KB)                                        |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Jane Smith-Johnson",
    "email": "jane@example.com",
    "qr_email": "jane.work@example.com",
    "employee_id": "EMP001",
    "phone": "+1-555-0123",
    "status": "confirmed",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "payment_date": "2025-11-08T12:30:00Z",
    "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
    "metadata": {
      "company": "New Tech Corp",
      "role": "Senior Engineer"
    },
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T14:30:00Z"
  },
  "message": "Participant updated successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to update this participant
- `404 Not Found` - Event or participant not found
- `409 Conflict` - Email already used by another participant in this event

---

### Delete Participant

Remove a participant from an event with payment protection to prevent accidental deletion of paid
participants.

**Endpoint:** `DELETE /api/v1/events/:id/participants/:pid`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| id        | UUID | Event ID       |
| pid       | UUID | Participant ID |

**Response:** `200 OK`

```json
{
  "success": true,
  "message": "Participant deleted successfully",
  "data": {
    "participant_id": "770e8400-e29b-41d4-a716-446655440000",
    "participant_name": "Jane Smith",
    "email": "jane@example.com",
    "checkin_deleted": true,
    "qr_code_invalidated": true,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

**Errors:**

- `400 Bad Request` - Cannot delete paid participant
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to delete this participant
- `404 Not Found` - Event or participant not found

**Validation Rules:**

**Payment Protection:**

- Participants with `payment_status = 'paid'` and `payment_amount > 0` **cannot be deleted**
- This protects financial records for accounting and audit purposes
- **Alternative:** Change participant `status` to `'cancelled'` instead of deleting

---

### Delete Participant - Error Response

**Error: Participant Has Payment Record**

```json
{
  "success": false,
  "error": "PARTICIPANT_HAS_PAYMENT",
  "message": "Cannot delete participant with payment record. Change status to 'cancelled' instead.",
  "data": {
    "participant_id": "770e8400-e29b-41d4-a716-446655440000",
    "participant_name": "Jane Smith",
    "email": "jane@example.com",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "payment_date": "2025-11-08T12:30:00Z",
    "alternative_action": {
      "method": "PATCH",
      "endpoint": "/api/v1/events/:id/participants/:pid",
      "body": {
        "status": "cancelled"
      },
      "description": "Cancel participant instead of deleting to preserve payment record"
    }
  }
}
```

---

### Delete Participant - Examples

**Example 1: Successful Deletion (No Payment)**

**Request:**

```bash
DELETE /api/v1/events/550e8400-e29b-41d4-a716-446655440000/participants/770e8400-e29b-41d4-a716-446655440000
Authorization: Bearer [access_token]
```

**Response:**

```json
{
  "success": true,
  "message": "Participant deleted successfully",
  "data": {
    "participant_id": "770e8400-e29b-41d4-a716-446655440000",
    "participant_name": "Jane Smith",
    "email": "jane@example.com",
    "payment_status": "unpaid",
    "checkin_deleted": false,
    "qr_code_invalidated": true,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

**Example 2: Failed Deletion (Has Payment)**

**Request:**

```bash
DELETE /api/v1/events/550e8400-e29b-41d4-a716-446655440000/participants/770e8400-e29b-41d4-a716-446655440000
Authorization: Bearer [access_token]
```

**Response:**

```json
{
  "success": false,
  "error": "PARTICIPANT_HAS_PAYMENT",
  "message": "Cannot delete participant with payment record",
  "data": {
    "participant_id": "770e8400-e29b-41d4-a716-446655440000",
    "payment_status": "paid",
    "payment_amount": 150.0,
    "alternative_action": "PATCH /api/v1/events/:id/participants/:pid with status='cancelled'"
  }
}
```

---

**Warning:** Deletion is irreversible and will delete:

- The participant's check-in record (if exists)
- The participant's QR code

**Important:**

- Deletion is logged in `deletion_audit_log` for compliance
- Participant snapshot is preserved in audit log for 3 years
- Payment protection prevents accidental financial record loss

---

### Export Participants (CSV)

Export all event participants to a CSV file.

**Endpoint:** `GET /api/v1/events/:id/participants/export`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Query Parameters:**

| Parameter        | Type    | Default | Description                       |
| ---------------- | ------- | ------- | --------------------------------- |
| status           | string  | all     | Filter by status or 'all'         |
| payment_status   | string  | all     | Filter by payment status or 'all' |
| checked_in       | boolean | -       | Filter by check-in status         |
| include_metadata | boolean | true    | Include metadata column           |

**Response:** `200 OK`

```
Content-Type: text/csv
Content-Disposition: attachment; filename="event_550e8400_participants.csv"

name,email,employee_id,phone,status,payment_status,payment_amount,payment_date,checked_in,checked_in_at,metadata
Jane Smith,jane@example.com,EMP001,+1-555-0123,confirmed,paid,150.00,2025-11-08T12:30:00Z,true,2025-12-15T09:15:00Z,"{""company"":""Tech Corp""}"
John Doe,john@example.com,EMP002,+1-555-0124,tentative,unpaid,,,false,,"{""company"":""StartupXYZ""}"
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event not found

---

## Participant Status

| Status      | Description                       | Typical Use Case            |
| ----------- | --------------------------------- | --------------------------- |
| `tentative` | Registration pending confirmation | Initial registration        |
| `confirmed` | Participation confirmed           | After payment/verification  |
| `cancelled` | Participant cancelled             | Cancellation by participant |
| `declined`  | Invitation declined               | Declined invitation         |

---

## Error Codes

| Code                             | Message                     | Description                   |
| -------------------------------- | --------------------------- | ----------------------------- |
| `PARTICIPANT_NOT_FOUND`          | Participant not found       | Participant ID does not exist |
| `PARTICIPANT_DUPLICATE_EMAIL`    | Email already registered    | Email exists for this event   |
| `PARTICIPANT_INVALID_STATUS`     | Invalid participant status  | Status value not allowed      |
| `PARTICIPANT_CSV_INVALID`        | Invalid CSV format          | CSV file format error         |
| `PARTICIPANT_CSV_TOO_LARGE`      | CSV file too large          | File exceeds 10MB limit       |
| `PARTICIPANT_METADATA_TOO_LARGE` | Metadata exceeds size limit | Metadata over 10KB            |
