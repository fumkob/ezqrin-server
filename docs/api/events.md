# Events API

## Overview

The Events API allows organizers to create, manage, and retrieve event information. Events are the
core entity that participants register for and check in to.

## Endpoints

### Create Event

Create a new event.

**Endpoint:** `POST /api/v1/events`

**Authentication:** Required (Organizer or Admin)

**Request Body:**

```json
{
  "name": "Tech Conference 2025",
  "description": "Annual technology conference featuring industry leaders",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T18:00:00Z",
  "location": "San Francisco Convention Center",
  "timezone": "America/Los_Angeles",
  "status": "draft"
}
```

**Request Fields:**

| Field       | Type   | Required | Description                                                                              |
| ----------- | ------ | -------- | ---------------------------------------------------------------------------------------- |
| name        | string | Yes      | Event name (1-255 characters)                                                            |
| description | string | No       | Event description (max 5000 characters)                                                  |
| start_date  | string | Yes      | ISO 8601 datetime                                                                        |
| end_date    | string | No       | ISO 8601 datetime (must be after start_date)                                             |
| location    | string | No       | Event venue/location (max 500 characters)                                                |
| timezone    | string | No       | IANA timezone (default: Asia/Tokyo)                                                      |
| status      | string | No       | Event status: `draft`, `published`, `ongoing`, `completed`, `cancelled` (default: draft) |

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "organizer_id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Tech Conference 2025",
    "description": "Annual technology conference featuring industry leaders",
    "start_date": "2025-12-15T09:00:00Z",
    "end_date": "2025-12-15T18:00:00Z",
    "location": "San Francisco Convention Center",
    "timezone": "America/Los_Angeles",
    "status": "draft",
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T10:00:00Z"
  },
  "message": "Event created successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `422 Unprocessable Entity` - Validation failed (e.g., end_date before start_date)

---

### List Events

Retrieve a paginated list of events. The results are filtered based on user role:

- **Admin**: All events
- **Organizer**: Only events they created
- **Staff**: Only events they are assigned to

**Endpoint:** `GET /api/v1/events`

**Authentication:** Required

**Query Parameters:**

| Parameter | Type    | Required | Description                                                                 |
| --------- | ------- | -------- | --------------------------------------------------------------------------- |
| page      | integer | No       | Page number (default: 1)                                                    |
| per_page  | integer | No       | Items per page (default: 20, max: 100)                                      |
| status    | string  | No       | Filter by status: `draft`, `published`, `ongoing`, `completed`, `cancelled` |
| sort      | string  | No       | Sort field: `created_at`, `start_date`, `name` (default: created_at)        |
| order     | string  | No       | Sort order: `asc`, `desc` (default: desc)                                   |
| search    | string  | No       | Search in event name and description                                        |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "organizer_id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "Tech Conference 2025",
      "description": "Annual technology conference featuring industry leaders",
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
  ],
  "message": "Events retrieved successfully",
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

**Errors:**

- `401 Unauthorized` - Authentication required

---

### Get Event

Retrieve detailed information about a specific event.

**Endpoint:** `GET /api/v1/events/:id`

**Authentication:** Required

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "organizer_id": "660e8400-e29b-41d4-a716-446655440000",
    "organizer": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "John Doe",
      "email": "john@example.com"
    },
    "name": "Tech Conference 2025",
    "description": "Annual technology conference featuring industry leaders",
    "start_date": "2025-12-15T09:00:00Z",
    "end_date": "2025-12-15T18:00:00Z",
    "location": "San Francisco Convention Center",
    "timezone": "America/Los_Angeles",
    "status": "published",
    "participant_count": 150,
    "checked_in_count": 87,
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T10:00:00Z"
  },
  "message": "Event retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event not found

---

### Update Event

Update an existing event.

**Endpoint:** `PUT /api/v1/events/:id`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request Body:**

```json
{
  "name": "Tech Conference 2025 - Updated",
  "description": "Updated description",
  "start_date": "2025-12-15T09:00:00Z",
  "end_date": "2025-12-15T18:00:00Z",
  "location": "Updated Location",
  "timezone": "America/Los_Angeles",
  "status": "published"
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "organizer_id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Tech Conference 2025 - Updated",
    "description": "Updated description",
    "start_date": "2025-12-15T09:00:00Z",
    "end_date": "2025-12-15T18:00:00Z",
    "location": "Updated Location",
    "timezone": "America/Los_Angeles",
    "status": "published",
    "created_at": "2025-11-08T10:00:00Z",
    "updated_at": "2025-11-08T11:00:00Z"
  },
  "message": "Event updated successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to update this event
- `404 Not Found` - Event not found
- `422 Unprocessable Entity` - Validation failed

---

### Delete Event

Delete an event and all associated data (participants, check-ins) with validation rules to prevent
accidental data loss.

**Endpoint:** `DELETE /api/v1/events/:id?force=true`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Query Parameters:**

| Parameter | Type    | Required | Description                                                        |
| --------- | ------- | -------- | ------------------------------------------------------------------ |
| force     | boolean | No       | Force deletion even with paid participants (requires confirmation) |

**Request Body (when force=true):**

```json
{
  "reason": "Event cancelled due to venue unavailability",
  "confirm_paid_participants_deleted": true
}
```

**Request Fields:**

| Field                             | Type    | Required | Description                                |
| --------------------------------- | ------- | -------- | ------------------------------------------ |
| reason                            | string  | Yes      | Reason for deletion (max 500 characters)   |
| confirm_paid_participants_deleted | boolean | Yes      | Must be `true` to delete paid participants |

**Response:** `200 OK`

```json
{
  "success": true,
  "message": "Event deleted successfully",
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_name": "Tech Conference 2025",
    "participants_deleted": 150,
    "paid_participants_deleted": 87,
    "unpaid_participants_deleted": 63,
    "checkins_deleted": 45,
    "staff_assignments_removed": 3,
    "qr_codes_invalidated": 150,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or missing required fields
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to delete this event
- `404 Not Found` - Event not found
- `409 Conflict` - Event is ongoing (cannot delete)
- `422 Unprocessable Entity` - Event has paid participants (requires force=true)

**Validation Rules:**

1. **Ongoing Event Protection:**
   - Cannot delete events with `status = 'ongoing'`
   - Event must be in `draft`, `published`, `completed`, or `cancelled` status
   - Error: `EVENT_IS_ONGOING`

2. **Paid Participant Warning:**
   - If event has participants with `payment_status = 'paid'`:
     - Requires `force=true` query parameter
     - Requires confirmation in request body
     - Returns count of paid participants that will be deleted
   - Error: `EVENT_HAS_PAID_PARTICIPANTS` (without force=true)

---

### Delete Event - Error Responses

**Error: Event is Ongoing**

```json
{
  "success": false,
  "error": "EVENT_IS_ONGOING",
  "message": "Cannot delete event with status 'ongoing'. Complete or cancel the event first.",
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_name": "Tech Conference 2025",
    "status": "ongoing",
    "suggestion": "Change event status to 'completed' or 'cancelled' before deletion"
  }
}
```

**Error: Event Has Paid Participants**

```json
{
  "success": false,
  "error": "EVENT_HAS_PAID_PARTICIPANTS",
  "message": "Event has 87 paid participants. Use force=true to confirm deletion.",
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_name": "Tech Conference 2025",
    "total_participants": 150,
    "paid_participants_count": 87,
    "total_payment_amount": 13050.0,
    "warning": "Deleting this event will permanently remove payment records",
    "required_action": "Add ?force=true parameter and confirm in request body"
  }
}
```

---

### Delete Event - Examples

**Example 1: Successful Deletion (No Paid Participants)**

**Request:**

```bash
DELETE /api/v1/events/550e8400-e29b-41d4-a716-446655440000
Authorization: Bearer [access_token]
```

**Response:**

```json
{
  "success": true,
  "message": "Event deleted successfully",
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_name": "Tech Conference 2025",
    "participants_deleted": 150,
    "paid_participants_deleted": 0,
    "checkins_deleted": 45,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

**Example 2: Force Deletion with Paid Participants**

**Request:**

```bash
DELETE /api/v1/events/550e8400-e29b-41d4-a716-446655440000?force=true
Authorization: Bearer [access_token]
Content-Type: application/json

{
  "reason": "Event cancelled - venue double booked",
  "confirm_paid_participants_deleted": true
}
```

**Response:**

```json
{
  "success": true,
  "message": "Event deleted successfully",
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_name": "Tech Conference 2025",
    "participants_deleted": 150,
    "paid_participants_deleted": 87,
    "unpaid_participants_deleted": 63,
    "checkins_deleted": 45,
    "staff_assignments_removed": 3,
    "deleted_at": "2025-11-08T15:30:00Z"
  }
}
```

---

**Warning:** This operation is irreversible and will delete:

- All event participants (including paid participants if force=true)
- All check-in records
- All QR codes
- All staff assignments
- All associated data

**Important:**

- Deletion is logged in `deletion_audit_log` for compliance
- Entity snapshot is preserved in audit log for 3 years
- Payment record deletion is logged for financial auditing

---

### Get Event Statistics

Retrieve statistics and metrics for an event.

**Endpoint:** `GET /api/v1/events/:id/stats`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "total_participants": 150,
    "checked_in_count": 87,
    "pending_count": 63,
    "check_in_rate": 58.0,
    "status_breakdown": {
      "confirmed": 120,
      "tentative": 25,
      "cancelled": 5
    },
    "checkin_timeline": [
      {
        "hour": "2025-12-15T09:00:00Z",
        "count": 25
      },
      {
        "hour": "2025-12-15T10:00:00Z",
        "count": 42
      }
    ],
    "checkin_methods": {
      "qrcode": 85,
      "manual": 2
    }
  },
  "message": "Event statistics retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event not found

---

### Assign Staff to Event

Assign a staff user to an event, granting them access to view participants and perform check-ins.

**Endpoint:** `POST /api/v1/events/:id/staff`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request Body:**

```json
{
  "staff_id": "770e8400-e29b-41d4-a716-446655440000"
}
```

**Request Fields:**

| Field    | Type | Required | Description                            |
| -------- | ---- | -------- | -------------------------------------- |
| staff_id | UUID | Yes      | ID of user with role='staff' to assign |

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "staff_id": "770e8400-e29b-41d4-a716-446655440000",
    "staff": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Jane Smith",
      "email": "jane@example.com",
      "role": "staff"
    },
    "assigned_at": "2025-11-08T14:00:00Z",
    "assigned_by": "660e8400-e29b-41d4-a716-446655440000"
  },
  "message": "Staff assigned successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or user is not a staff member
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to assign staff to this event
- `404 Not Found` - Event or user not found
- `409 Conflict` - Staff already assigned to this event

---

### List Event Staff

Get all staff members assigned to an event.

**Endpoint:** `GET /api/v1/events/:id/staff`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "staff_id": "770e8400-e29b-41d4-a716-446655440000",
      "staff": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Jane Smith",
        "email": "jane@example.com",
        "role": "staff"
      },
      "assigned_at": "2025-11-08T14:00:00Z",
      "assigned_by": "660e8400-e29b-41d4-a716-446655440000"
    }
  ],
  "message": "Staff list retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to view staff for this event
- `404 Not Found` - Event not found

---

### Remove Staff from Event

Remove a staff member's assignment from an event.

**Endpoint:** `DELETE /api/v1/events/:id/staff/:staff_id`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description   |
| --------- | ---- | ------------- |
| id        | UUID | Event ID      |
| staff_id  | UUID | Staff user ID |

**Response:** `200 OK`

```json
{
  "success": true,
  "message": "Staff removed successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to remove staff from this event
- `404 Not Found` - Event, staff user, or assignment not found

---

## Event Status Lifecycle

```
draft → published → ongoing → completed
   ↓                              ↓
cancelled ← ← ← ← ← ← ← ← ← ← cancelled
```

### Status Descriptions

| Status      | Description                                  | Allowed Actions                         |
| ----------- | -------------------------------------------- | --------------------------------------- |
| `draft`     | Event is being prepared                      | Edit, delete, add participants, publish |
| `published` | Event is visible and accepting registrations | Edit, add participants, start, cancel   |
| `ongoing`   | Event is currently happening                 | Check-in participants, view stats       |
| `completed` | Event has ended                              | View data, export reports, archive      |
| `cancelled` | Event was cancelled                          | View data only                          |

---

## Error Codes

| Code                          | Message                              | Description                          |
| ----------------------------- | ------------------------------------ | ------------------------------------ |
| `EVENT_NOT_FOUND`             | Event not found                      | Event ID does not exist              |
| `EVENT_UNAUTHORIZED`          | Not authorized to access this event  | User does not have access to event   |
| `EVENT_INVALID_DATES`         | Invalid event dates                  | End date is before start date        |
| `EVENT_INVALID_STATUS`        | Invalid status transition            | Status change not allowed            |
| `EVENT_CANNOT_DELETE`         | Cannot delete event                  | Event has active check-ins           |
| `EVENT_IS_ONGOING`            | Cannot delete ongoing event          | Event status is 'ongoing'            |
| `EVENT_HAS_PAID_PARTICIPANTS` | Event has paid participants          | Requires force=true to delete        |
| `STAFF_NOT_FOUND`             | Staff user not found                 | Staff user ID does not exist         |
| `STAFF_INVALID_ROLE`          | User is not a staff member           | User must have role='staff'          |
| `STAFF_ALREADY_ASSIGNED`      | Staff already assigned to this event | Duplicate assignment not allowed     |
| `STAFF_ASSIGNMENT_NOT_FOUND`  | Staff assignment not found           | No assignment exists for this staff  |
| `STAFF_UNAUTHORIZED`          | Not authorized for staff operations  | Only event owner or admin can assign |
