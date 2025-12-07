# QR Code API

## Overview

The QR Code API manages QR code generation, distribution, and self-service participant access via
secure tokens for event participants.

**QR Code Generation:** Each participant automatically receives a unique QR code when registered for
an event (see [Participants API](./participants.md)). This endpoint focuses on distribution (email)
and retrieval of existing QR codes.

**Key Features:**

- Send QR codes to participants via email
- Multiple email templates
- Individual QR code retrieval (organizer/staff access)
- **Self-service participant access** via secure token-based URLs
- Check-in status tracking for participants
- Email template customization with secure access links

## Endpoints

### Send QR Codes via Email

Send QR codes to participants via email.

**Endpoint:** `POST /api/v1/events/:id/qrcodes/send`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| id        | UUID | Event ID    |

**Request Body:**

```json
{
  "participant_ids": [
    "770e8400-e29b-41d4-a716-446655440000",
    "771e8400-e29b-41d4-a716-446655440001"
  ],
  "send_to_all": false,
  "email_template": "default"
}
```

**Request Fields:**

| Field           | Type    | Required | Description                                                         |
| --------------- | ------- | -------- | ------------------------------------------------------------------- |
| participant_ids | array   | No\*     | Array of participant UUIDs to send to                               |
| send_to_all     | boolean | No\*     | Send to all confirmed participants (default: false)                 |
| email_template  | string  | No       | Email template: `default`, `minimal`, `detailed` (default: default) |

\*Either `participant_ids` or `send_to_all=true` must be provided

**Response:**

**For small batches (200 OK):** Synchronous response when total participants < 100

```json
{
  "success": true,
  "data": {
    "sent_count": 50,
    "failed_count": 2,
    "total": 52,
    "failures": [
      {
        "participant_id": "772e8400-e29b-41d4-a716-446655440002",
        "email": "invalid@example.com",
        "reason": "Invalid email address format"
      }
    ]
  },
  "message": "QR codes sent successfully"
}
```

**For large batches (202 Accepted):** Asynchronous processing when total participants >= 100

```json
{
  "success": true,
  "data": {
    "job_id": "job_550e8400_20251122_143022",
    "status": "queued",
    "total_participants": 5000,
    "estimated_completion": "2025-11-22T14:50:00Z",
    "progress": {
      "processed": 0,
      "sent": 0,
      "failed": 0
    }
  },
  "message": "Email sending job queued for processing"
}
```

**Partial Success (207 Multi-Status):** Some emails sent, some failed (only for synchronous
response)

```json
{
  "success": true,
  "data": {
    "sent_count": 148,
    "failed_count": 2,
    "total": 150,
    "failures": [
      {
        "participant_id": "772e8400-e29b-41d4-a716-446655440002",
        "email": "invalid@example.com",
        "reason": "Invalid email address format"
      }
    ]
  },
  "message": "Email send completed with errors"
}
```

**Errors:**

- `400 Bad Request` - Invalid request (no participants specified)
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Not authorized to send emails for this event
- `404 Not Found` - Event not found
- `429 Too Many Requests` - Email rate limit exceeded

---

### Get Email Job Status

Check the status of an asynchronous email sending job.

**Endpoint:** `GET /api/v1/jobs/:job_id`

**Authentication:** Required (Event owner or Admin)

**Path Parameters:**

| Parameter | Type   | Description               |
| --------- | ------ | ------------------------- |
| job_id    | string | Job ID from POST response |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "job_id": "job_550e8400_20251122_143022",
    "status": "in_progress",
    "total_participants": 5000,
    "progress": {
      "processed": 3200,
      "sent": 3150,
      "failed": 50,
      "percentage": 64
    },
    "estimated_completion": "2025-11-22T14:45:00Z",
    "started_at": "2025-11-22T14:25:00Z",
    "recent_failures": [
      {
        "participant_id": "772e8400-e29b-41d4-a716-446655440002",
        "email": "invalid@example.com",
        "reason": "Invalid email address format",
        "timestamp": "2025-11-22T14:30:15Z"
      }
    ]
  },
  "message": "Job in progress"
}
```

**Status Values:**

| Status        | Description                           |
| ------------- | ------------------------------------- |
| `queued`      | Job waiting to be processed           |
| `in_progress` | Job is currently processing           |
| `completed`   | Job completed successfully (all sent) |
| `partial`     | Job completed with some failures      |
| `failed`      | Job failed (critical error)           |
| `cancelled`   | Job was cancelled by user             |

**Errors:**

- `401 Unauthorized` - Authentication required
- `404 Not Found` - Job not found or expired
- `403 Forbidden` - Not authorized to check this job

---

### Get Individual QR Code

Retrieve QR code for a specific participant.

**Endpoint:** `GET /api/v1/participants/:pid/qrcode`

**Authentication:** Required (Event owner, assigned staff, or Admin)

**Path Parameters:**

| Parameter | Type | Description    |
| --------- | ---- | -------------- |
| pid       | UUID | Participant ID |

**Query Parameters:**

| Parameter | Type    | Default | Description                              |
| --------- | ------- | ------- | ---------------------------------------- |
| format    | string  | png     | Image format: `png`, `svg`, `json`       |
| size      | integer | 512     | QR code size in pixels (256-2048)        |
| download  | boolean | false   | Force download instead of inline display |

**Response (format=png):** `200 OK`

```
Content-Type: image/png
Content-Disposition: inline; filename="qrcode_770e8400.png"

[Binary PNG image data]
```

**Response (format=json):** `200 OK`

```json
{
  "success": true,
  "data": {
    "participant_id": "770e8400-e29b-41d4-a716-446655440000",
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "qr_code": "evt_550e8400_prt_770e8400_abc123def456",
    "qr_image_url": "https://api.ezqrin.com/qr/770e8400.png",
    "generated_at": "2025-11-08T10:00:00Z",
    "event": {
      "name": "Tech Conference 2025",
      "start_date": "2025-12-15T09:00:00Z",
      "location": "San Francisco Convention Center"
    }
  },
  "message": "QR code retrieved successfully"
}
```

**Errors:**

- `401 Unauthorized` - Authentication required
- `403 Forbidden` - No access to this participant's event
- `404 Not Found` - Participant not found

---

## QR Code Specifications

### Token Format

Each QR code contains a unique token with the following structure:

```
evt_{event_id}_prt_{participant_id}_{random_token}
```

**Components:**

- `evt_` prefix identifies this as an event token
- `event_id`: Short form of event UUID
- `prt_` separator for participant section
- `participant_id`: Short form of participant UUID
- `random_token`: 12-character cryptographic random string

**Example:**

```
evt_550e8400_prt_770e8400_abc123def456
```

### QR Code Properties

**Image Specifications:**

- **Error Correction:** Level H (30% recovery capability)
- **Encoding:** UTF-8
- **Module Size:** Calculated based on requested image size
- **Quiet Zone:** 4 modules (standard)

**Supported Formats:**

| Format | MIME Type        | Use Case                          |
| ------ | ---------------- | --------------------------------- |
| PNG    | image/png        | General purpose, web display      |
| SVG    | image/svg+xml    | High quality, scalable, printing  |
| JSON   | application/json | API response, programmatic access |

**Size Recommendations:**

| Use Case           | Recommended Size |
| ------------------ | ---------------- |
| Email display      | 256px or 512px   |
| Mobile app         | 512px            |
| Printing (4x4 cm)  | 1024px           |
| Large format print | 2048px           |

---

## Email Templates

### Default Template

**Subject:** Your QR Code for {Event Name}

**Contents:**

- Event details (name, date, time, location)
- Participant name
- QR code image (embedded)
- Check-in instructions
- Contact information

### Minimal Template

**Subject:** Check-in Code - {Event Name}

**Contents:**

- QR code image
- Event name and date
- Brief check-in instructions

### Detailed Template

**Subject:** Complete Event Information - {Event Name}

**Contents:**

- Full event description
- Agenda/schedule
- Participant details
- QR code
- Venue map/directions
- FAQs
- Contact information

### Email Template Customization

Event organizers can create and manage custom email templates for their events.

**Customization Features:**

- **HTML/Text Templates**: Support both HTML (recommended) and plain text
- **Dynamic Variables**: Insert participant and event information via placeholders
- **Template Reuse**: Save templates and reuse across events
- **Preview & Testing**: Test templates before sending to participants
- **Multiple Languages**: Customize templates per language (future feature)

#### Available Template Variables

**Event Information:**

```
{event_name}           - Event name
{event_date}           - Event date (formatted)
{event_time}           - Event start time
{event_end_time}       - Event end time
{event_location}       - Event physical location
{event_address}        - Full venue address
{event_timezone}       - Event timezone
{event_description}    - Event description/summary
{organizer_name}       - Event organizer name
{organizer_email}      - Event organizer email
{organizer_phone}      - Event organizer phone
{event_url}            - Link to event details page
```

**Participant Information:**

```
{participant_name}     - Participant full name
{participant_email}    - Participant email address
{participant_id}       - Participant UUID (system ID)
{qr_code_image}        - Embedded QR code (PNG)
{qr_code_link}         - Direct link to retrieve QR code (token-based)
{qr_code_refresh_link} - Link to refresh/download QR code again
{checkin_status_link}  - Link to check check-in status (token-based)
```

**System Information:**

```
{current_date}         - Current date
{current_year}         - Current year (for copyright)
{organization_name}    - Event organization name (from settings)
```

#### Template Management Endpoints

**Create Custom Template**

```
POST /api/v1/events/:event_id/email-templates
```

**Request (Example with token-based participant access):**

```json
{
  "name": "Event Check-in Information",
  "description": "Email template with direct links for participant access",
  "template_type": "html",
  "subject": "Your {event_name} Check-in Information",
  "body": "<html><body>Hello {participant_name},<br/><br/>You are registered for <strong>{event_name}</strong> on {event_date} at {event_time}.<br/><br/><h3>Check-in</h3><p>Show your QR code at the check-in desk:</p><p><strong>{qr_code_image}</strong></p><p>Or view your QR code: <a href=\"{qr_code_link}\">View QR Code</a></p><p><a href=\"{qr_code_refresh_link}\">Download QR Code</a></p><br/><h3>Check Your Status</h3><p><a href=\"{checkin_status_link}\">Check if you're checked in</a></p><br/><hr/><p>Location: {event_location}</p><p>Questions? Contact us at {organizer_email}</p></body></html>",
  "is_default": false,
  "is_global": false
}
```

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "tpl_550e8400-e29b-41d4-a716-446655440000",
    "name": "VIP Welcome Email",
    "created_at": "2025-11-08T10:00:00Z"
  }
}
```

**List Event Templates**

```
GET /api/v1/events/:event_id/email-templates
```

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "tpl_550e8400-e29b-41d4-a716-446655440000",
      "name": "Default Template",
      "is_system": true,
      "created_at": "2025-11-08T10:00:00Z"
    },
    {
      "id": "tpl_550e8401-e29b-41d4-a716-446655440001",
      "name": "VIP Welcome Email",
      "is_system": false,
      "created_at": "2025-11-08T14:22:00Z"
    }
  ]
}
```

**Preview Template**

```
POST /api/v1/events/:event_id/email-templates/:template_id/preview
```

**Request:**

```json
{
  "participant_id": "770e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "subject": "Welcome to Tech Conference 2025 - VIP Access",
    "body_preview": "<html><body>Hello John Doe,<br/><br/>You are invited to Tech Conference 2025 on 2025-12-15...",
    "rendered_correctly": true,
    "variables_found": ["{participant_name}", "{event_name}", "{event_date}"],
    "variables_missing": []
  }
}
```

**Update Template**

```
PUT /api/v1/events/:event_id/email-templates/:template_id
```

**Delete Template**

```
DELETE /api/v1/events/:event_id/email-templates/:template_id
```

#### Template Validation

**Automatic Validation:**

- All required variables present (participant_name, event_name, etc.)
- Valid variable syntax: `{variable_name}`
- Subject length between 5-200 characters
- HTML body length between 50-50,000 characters
- No embedded malicious scripts (HTML sanitization)

**Validation Error Response:**

```json
{
  "success": false,
  "error": {
    "code": "TEMPLATE_VALIDATION_FAILED",
    "message": "Template validation failed",
    "details": {
      "missing_variables": ["participant_email"],
      "invalid_syntax": ["{invalid_var}"],
      "length_issues": []
    }
  }
}
```

#### Best Practices

**Do's:**

- ✅ Use built-in templates as starting points
- ✅ Test templates with preview before using in campaigns
- ✅ Include participant name and event details
- ✅ Provide clear check-in instructions
- ✅ Add contact information for support
- ✅ Use responsive HTML for email compatibility

**Don'ts:**

- ❌ Don't include participant passwords or sensitive data
- ❌ Don't use external images (use embedded or CDN links)
- ❌ Don't exceed 50,000 characters for email body
- ❌ Don't remove required variables without testing
- ❌ Don't use unsupported variables (will show as literal text)

---

## Security Considerations

### QR Code Security

1. **Token Signing:**
   - Each QR code token includes HMAC signature
   - Prevents token forgery and tampering
   - Server-side validation on check-in

2. **Rate Limiting:**
   - See [Rate Limiting Strategy](./rate_limits.md) for comprehensive limits
   - Email sending: 100 emails/minute per event, 1,000/hour, 5,000/day per organizer

### Privacy Protection

1. **Email Handling:**
   - Emails sent via secure SMTP (TLS)
   - No email addresses exposed in API responses
   - Unsubscribe option included

2. **Data Minimization:**
   - QR codes contain only necessary identifiers
   - Personal data not embedded in QR code
   - Lookup required for participant details

---

## Best Practices

### For Event Organizers

1. **Send QR codes early:**
   - Send 1-2 weeks before event
   - Send reminder 1 day before event

2. **Test QR codes:**
   - Verify scanning works with your check-in app
   - Ensure proper display on various devices

3. **Provide alternatives:**
   - Include participant ID in email
   - Offer manual check-in option
   - Print backup QR codes for on-site

### For Developers

1. **Handle errors gracefully:**
   - Retry failed email sends
   - Log generation failures
   - Provide clear error messages

2. **Optimize performance:**
   - Generate QR codes asynchronously for large batches
   - Cache generated images
   - Use CDN for QR code delivery

3. **Monitor usage:**
   - Track QR code scans
   - Monitor email delivery rates
   - Alert on unusual patterns

---

## Error Codes

| Code                   | Message                         | Description               |
| ---------------------- | ------------------------------- | ------------------------- |
| `QR_GENERATION_FAILED` | Failed to generate QR code      | Image generation error    |
| `QR_INVALID_FORMAT`    | Invalid format requested        | Unsupported image format  |
| `QR_SIZE_OUT_OF_RANGE` | QR code size out of valid range | Size not between 256-2048 |
| `EMAIL_SEND_FAILED`    | Failed to send email            | SMTP or delivery error    |
| `EMAIL_RATE_LIMIT`     | Email rate limit exceeded       | Too many emails sent      |
