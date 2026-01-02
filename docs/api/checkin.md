# Check-in API

## Overview

The Check-in API handles participant attendance tracking using QR code scanning or manual entry. It
provides real-time check-in capabilities and historical data retrieval.

## Endpoints

### Perform Check-in

Check in a participant to an event.

**Endpoint:** `POST /api/v1/events/:id/checkin`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request Body:**

```json
{
  "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
  "checkin_method": "qrcode",
  "device_info": {
    "device_type": "mobile",
    "os": "iOS 17.0",
    "app_version": "1.0.0"
  }
}
```

**Request Fields:**

| Field          | Type   | Required | Description                                  |
| -------------- | ------ | -------- | -------------------------------------------- |
| qr_code        | string | Yes\*    | QR code token from participant               |
| participant_id | UUID   | Yes\*    | Participant UUID (alternative to qr_code)    |
| checkin_method | string | No       | Method: `qrcode`, `manual` (default: qrcode) |
| device_info    | object | No       | Device metadata (max 5KB)                    |

\*Either `qr_code` or `participant_id` must be provided

**Response:** `201 Created`

```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "participant_id": "770e8400-e29b-41d4-a716-446655440000",
  "participant": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "name": "Jane Smith",
    "email": "jane@example.com"
  },
  "checked_in_at": "2025-12-15T09:15:00Z",
  "checked_in_by": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Staff User"
  },
  "checkin_method": "qrcode",
  "device_info": {
    "device_type": "mobile",
    "os": "iOS 17.0",
    "app_version": "1.0.0"
  }
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or missing required fields
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to perform check-in for this event
- `404 Not Found` - Event or participant not found
- `409 Conflict` - Participant already checked in
- `422 Unprocessable Entity` - Invalid QR code or expired token

---

### Get Check-in History

Retrieve check-in records for an event.

**Endpoint:** `GET /api/v1/events/:id/checkins`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Query Parameters:**

| Parameter | Type    | Required | Description                                                              |
| --------- | ------- | -------- | ------------------------------------------------------------------------ |
| page      | integer | No       | Page number (default: 1)                                                 |
| per_page  | integer | No       | Items per page (default: 20, max: 100)                                   |
| sort      | string  | No       | Sort field: `checked_in_at`, `participant_name` (default: checked_in_at) |
| order     | string  | No       | Sort order: `asc`, `desc` (default: desc)                                |
| search    | string  | No       | Search in participant name/email                                         |
| from_date | string  | No       | Filter check-ins from this datetime (ISO 8601)                           |
| to_date   | string  | No       | Filter check-ins until this datetime (ISO 8601)                          |
| method    | string  | No       | Filter by method: `qrcode`, `manual`                                     |

**Response:** `200 OK`

```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "participant": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Jane Smith",
        "email": "jane@example.com"
      },
      "checked_in_at": "2025-12-15T09:15:00Z",
      "checked_in_by": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "name": "Staff User"
      },
      "checkin_method": "qrcode"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 87,
    "total_pages": 5
  }
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this event
- `404 Not Found` - Event not found

---

### Cancel Check-in

Remove a check-in record (undo check-in).

**Endpoint:** `DELETE /api/v1/events/:id/checkins/:cid`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |
| cid       | UUID | Check-in ID |

**Response:** `204 No Content`

(Empty body)

**Errors:**

```json
{
  "type": "https://api.ezqrin.com/problems/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "Event or check-in record not found",
  "instance": "/api/v1/events/550e8400/checkins/880e8400",
  "code": "NOT_FOUND"
}
```

---

## Check-in Methods

### QR Code Check-in

**Process:**

1. Participant presents QR code (from email, wallet pass, or printed)
2. Staff scans QR code using mobile app or web interface
3. System validates QR code token
4. Check-in record created with timestamp

**QR Code Format:**

```
evt_{event_id}_prt_{participant_id}_{random_token}
```

**Validation Rules:**

- QR code must be valid for the event
- QR code must match a registered participant
- Participant cannot already be checked in
- Event must be in `published` or `ongoing` status

### Manual Check-in

**Process:**

1. Staff searches for participant by name or email
2. Staff manually confirms participant identity
3. Check-in record created with `manual` method

**Use Cases:**

- QR code not available
- QR code scanner malfunction
- Backup check-in method

---

## Real-time Features

### Check-in Rate Limiting

To prevent abuse and ensure system stability:

- **Per Event:** 50 check-ins per minute
- **Per IP:** 100 requests per minute
- **Per User:** 200 check-ins per minute

### Duplicate Prevention

The system enforces a unique constraint on `(event_id, participant_id)`:

- One check-in per participant per event
- Attempting duplicate check-in returns `409 Conflict`
- Use cancel check-in endpoint to undo, then check in again if needed

---

## Check-in Analytics

### Real-time Metrics

Available through the [Event Statistics](./events.md#get-event-statistics) endpoint:

- Total participants vs checked-in count
- Check-in rate percentage
- Check-ins by hour timeline
- Check-ins by method breakdown

### Export Options

Check-in data can be exported via:

- [Participant export](./participants.md#export-participants-csv) (includes check-in status)
- Check-in history endpoint with pagination
- Event statistics for aggregate data

---

## Error Codes

| Code                            | Message                          | Description                               |
| ------------------------------- | -------------------------------- | ----------------------------------------- |
| `CHECKIN_ALREADY_CHECKED_IN`    | Participant already checked in   | Duplicate check-in attempt                |
| `CHECKIN_INVALID_QR`            | Invalid or expired QR code       | QR code validation failed                 |
| `CHECKIN_NOT_FOUND`             | Check-in record not found        | Check-in ID does not exist                |
| `CHECKIN_RATE_LIMIT`            | Rate limit exceeded              | Too many check-in requests                |
| `CHECKIN_EVENT_NOT_ACTIVE`      | Event is not accepting check-ins | Event status not `published` or `ongoing` |
| `CHECKIN_PARTICIPANT_CANCELLED` | Participant status is cancelled  | Cannot check in cancelled participants    |

---

## Best Practices

### For Mobile Apps

1. **Offline Support:**
   - Cache participant list for offline validation
   - Queue check-ins and sync when online
   - Show clear offline/online status

2. **QR Code Scanning:**
   - Use device camera for fast scanning
   - Provide visual/audio feedback on successful scan
   - Handle poor lighting conditions

3. **Error Handling:**
   - Provide clear error messages
   - Offer manual check-in fallback
   - Log failed attempts for troubleshooting

### For Web Interfaces

1. **Real-time Updates:**
   - Use WebSocket or polling for live check-in count
   - Update participant list dynamically
   - Show recent check-in activity

2. **Search Optimization:**
   - Implement client-side filtering for speed
   - Provide autocomplete for participant search
   - Allow multiple search criteria

3. **Bulk Operations:**
   - Support batch check-in for group arrivals
   - Provide keyboard shortcuts for efficiency
   - Enable barcode scanner hardware integration
