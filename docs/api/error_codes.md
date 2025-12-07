# Error Code Reference

Complete list of error codes returned by the ezQRin API with detailed explanations and solutions.

## Error Response Format

All errors follow a standard format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field": "Details about the error"
    }
  },
  "request_id": "req_123456789"
}
```

**Standard HTTP Status Codes:**

- `400 Bad Request` - Client error (invalid input)
- `401 Unauthorized` - Authentication required or failed
- `403 Forbidden` - Authorized but not permitted
- `404 Not Found` - Resource doesn't exist
- `409 Conflict` - State conflict (e.g., duplicate email)
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

---

## Authentication Errors

### AUTH_INVALID_CREDENTIALS

- **HTTP Status:** 401 Unauthorized
- **Message:** Invalid email or password
- **Cause:** Login credentials don't match any user
- **Solution:** Verify email and password are correct, or reset password
- **Retry:** No, create account or reset password instead

### AUTH_INVALID_TOKEN

- **HTTP Status:** 401 Unauthorized
- **Message:** Invalid or expired token
- **Cause:** JWT token is malformed, expired, or tampered with
- **Solution:** Login again to get new token, or refresh token
- **Retry:** No, re-authenticate required

### AUTH_SESSION_EXPIRED

- **HTTP Status:** 401 Unauthorized
- **Message:** Session has expired
- **Cause:** Token was valid but has now expired (15 minutes for access token)
- **Solution:** Use refresh token to get new access token
- **Retry:** Yes, with refreshed token

### AUTH_EMAIL_EXISTS

- **HTTP Status:** 409 Conflict
- **Message:** Email already registered
- **Cause:** Email address is already associated with an account
- **Solution:** Login with this email, or use different email for new account
- **Retry:** No, use existing account

### AUTH_WEAK_PASSWORD

- **HTTP Status:** 400 Bad Request
- **Message:** Password does not meet requirements
- **Cause:** Password is too weak (need 8+ chars, uppercase, lowercase, number)
- **Solution:** Create stronger password
- **Retry:** Yes, with stronger password

### AUTH_ACCOUNT_LOCKED

- **HTTP Status:** 403 Forbidden
- **Message:** Account temporarily locked
- **Cause:** Too many failed login attempts (5 in 15 minutes)
- **Solution:** Wait 15 minutes before retrying login
- **Retry:** Yes, after lockout period expires

### AUTH_UNAUTHORIZED

- **HTTP Status:** 401 Unauthorized
- **Message:** Unauthorized access
- **Cause:** Missing authentication header or token not provided
- **Solution:** Include Authorization header with valid token
- **Retry:** Yes, with proper authentication

### AUTH_INSUFFICIENT_PERMISSIONS

- **HTTP Status:** 403 Forbidden
- **Message:** Insufficient permissions for this operation
- **Cause:** User role doesn't have permission for this operation
- **Solution:** Admin can assign higher role, or use account with permission
- **Retry:** No, need permission/role change

### AUTH_MAGIC_LINK_INVALID

- **HTTP Status:** 401 Unauthorized
- **Message:** Invalid or expired magic link
- **Cause:** Magic link token is invalid, expired, or already used
- **Solution:** Request new magic link via email
- **Retry:** No, get new magic link

### AUTH_MAGIC_LINK_USED

- **HTTP Status:** 401 Unauthorized
- **Message:** Magic link already used
- **Cause:** Magic link is single-use and has been used
- **Solution:** Request new magic link via email
- **Retry:** No, get new magic link

### AUTH_RATE_LIMIT

- **HTTP Status:** 429 Too Many Requests
- **Message:** Too many authentication attempts
- **Cause:** Rate limit exceeded (5 logins per 15 minutes per IP)
- **Solution:** Wait before retrying, check for automated attacks
- **Retry:** Yes, after rate limit window passes

---

## Event Errors

### EVENT_NOT_FOUND

- **HTTP Status:** 404 Not Found
- **Message:** Event not found
- **Cause:** Event ID doesn't exist or was deleted
- **Solution:** Verify event ID, or list events to find correct ID
- **Retry:** No, use correct event ID

### EVENT_INVALID_STATUS

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid event status for this operation
- **Cause:** Event status (past, cancelled, etc.) prevents operation
- **Solution:** Check event status, only ongoing/future events allow certain ops
- **Retry:** No, wait for event status change

### EVENT_DUPLICATE_NAME

- **HTTP Status:** 409 Conflict
- **Message:** Event with this name already exists
- **Cause:** Event name is not unique within organization
- **Solution:** Use different event name
- **Retry:** Yes, with different name

### EVENT_INVALID_DATE_RANGE

- **HTTP Status:** 400 Bad Request
- **Message:** End date must be after start date
- **Cause:** Event end_date is before or equal to start_date
- **Solution:** Ensure end_date > start_date
- **Retry:** Yes, with valid dates

### EVENT_CAPACITY_EXCEEDED

- **HTTP Status:** 400 Bad Request
- **Message:** Event capacity exceeded
- **Cause:** Too many participants for event capacity
- **Solution:** Increase event capacity or reduce participants
- **Retry:** Yes, after adjusting capacity

---

## Deletion Errors

### USER_HAS_ACTIVE_EVENTS

- **HTTP Status:** 400 Bad Request
- **Message:** Cannot delete user with active events
- **Cause:** User owns events that are not completed or cancelled
- **Solution:** Complete or cancel all events before deleting user
- **Retry:** Yes, after event status changes

### USER_ALREADY_DELETED

- **HTTP Status:** 404 Not Found
- **Message:** User has already been deleted
- **Cause:** User was previously deleted and anonymized
- **Solution:** User no longer exists in the system
- **Retry:** No, user is permanently anonymized

### USER_DELETION_FORBIDDEN

- **HTTP Status:** 403 Forbidden
- **Message:** User deletion not permitted
- **Cause:** Admin-only operation, or user attempting to delete other users
- **Solution:** Only admins or the user themselves can delete their account
- **Retry:** No, need proper authorization

### PARTICIPANT_HAS_PAYMENT

- **HTTP Status:** 400 Bad Request
- **Message:** Cannot delete participant with payment record
- **Cause:** Participant has payment_status='paid' and payment_amount > 0
- **Details:** Shows payment amount and suggests status change alternative
- **Solution:** Change participant status to 'cancelled' instead of deleting
- **Retry:** No, use status update instead

### EVENT_IS_ONGOING

- **HTTP Status:** 400 Bad Request
- **Message:** Cannot delete ongoing event
- **Cause:** Event status is 'ongoing' and deletion is not permitted
- **Solution:** Wait for event to complete or change status to 'cancelled' first
- **Retry:** Yes, after event status changes

### EVENT_HAS_PAID_PARTICIPANTS

- **HTTP Status:** 400 Bad Request
- **Message:** Event has participants with payment records
- **Cause:** Event contains paid participants, deletion will affect financial records
- **Details:** Shows count of paid participants
- **Solution:** Use force=true parameter with confirmation if deletion is required
- **Retry:** Yes, with force parameter and confirmation

### DELETION_LOG_FORBIDDEN

- **HTTP Status:** 403 Forbidden
- **Message:** Deletion log access forbidden
- **Cause:** Admin-only operation, non-admin user attempted to access logs
- **Solution:** Only admins can view deletion audit logs
- **Retry:** No, need admin role

### INVALID_CONFIRMATION

- **HTTP Status:** 400 Bad Request
- **Message:** Deletion confirmation required
- **Cause:** Deletion requires explicit confirmation but it wasn't provided
- **Solution:** Include confirm: true in request body for destructive operations
- **Retry:** Yes, with confirmation parameter

### DELETION_REASON_REQUIRED

- **HTTP Status:** 400 Bad Request
- **Message:** Deletion reason is required
- **Cause:** Deletion requires a reason for audit trail but none was provided
- **Solution:** Include reason field in request body
- **Retry:** Yes, with reason provided

---

## Participant Errors

### PARTICIPANT_NOT_FOUND

- **HTTP Status:** 404 Not Found
- **Message:** Participant not found
- **Cause:** Participant ID doesn't exist or was deleted
- **Solution:** Verify participant ID, or list participants
- **Retry:** No, use correct participant ID

### PARTICIPANT_DUPLICATE_EMAIL

- **HTTP Status:** 409 Conflict
- **Message:** Participant with this email already registered
- **Cause:** Email is already registered for this event
- **Solution:** Use different email or update existing participant
- **Retry:** Yes, with different email

### PARTICIPANT_INVALID_EMAIL

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid email address format
- **Cause:** Email doesn't match standard email format
- **Solution:** Provide valid email address
- **Retry:** Yes, with valid email

### PARTICIPANT_ALREADY_CHECKED_IN

- **HTTP Status:** 400 Bad Request
- **Message:** Participant already checked in
- **Cause:** Participant has already completed check-in
- **Solution:** No action needed, or uncheck-in if required
- **Retry:** No, participant already checked in

### PARTICIPANT_NOT_REGISTERED

- **HTTP Status:** 403 Forbidden
- **Message:** Not registered for this event
- **Cause:** Participant trying to access event they're not registered for
- **Solution:** Register for event first
- **Retry:** No, need to register first

---

## QR Code Errors

### QR_CODE_GENERATION_FAILED

- **HTTP Status:** 500 Internal Server Error
- **Message:** Failed to generate QR code
- **Cause:** QR code image generation encountered error
- **Solution:** Retry request, contact support if persists
- **Retry:** Yes, with exponential backoff

### QR_CODE_INVALID_FORMAT

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid format requested
- **Cause:** Format parameter not one of: png, svg, json
- **Solution:** Use valid format (png, svg, or json)
- **Retry:** Yes, with valid format

### QR_CODE_SIZE_OUT_OF_RANGE

- **HTTP Status:** 400 Bad Request
- **Message:** QR code size out of valid range
- **Cause:** Size parameter not between 256 and 2048
- **Solution:** Use size between 256-2048 pixels
- **Retry:** Yes, with valid size

---

## Email Errors

### EMAIL_SEND_FAILED

- **HTTP Status:** 500 Internal Server Error
- **Message:** Failed to send email
- **Cause:** SMTP or email provider error
- **Solution:** Retry later, check email configuration
- **Retry:** Yes, implemented automatically

### EMAIL_INVALID_ADDRESS

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid email address
- **Cause:** Email address format is invalid
- **Solution:** Fix email address format
- **Retry:** Yes, with valid email

### EMAIL_RATE_LIMIT

- **HTTP Status:** 429 Too Many Requests
- **Message:** Email rate limit exceeded
- **Cause:** Too many emails sent (see Rate Limiting Strategy)
- **Solution:** Wait before sending more emails
- **Retry:** Yes, after rate limit window

### EMAIL_BOUNCE

- **HTTP Status:** 400 Bad Request
- **Message:** Email address is bouncing
- **Cause:** Email provider rejected address (hard bounce)
- **Solution:** Use valid email address, contact participant
- **Retry:** No, address is invalid

### EMAIL_UNSUBSCRIBED

- **HTTP Status:** 403 Forbidden
- **Message:** Recipient has unsubscribed
- **Cause:** Participant unsubscribed from emails
- **Solution:** Respect unsubscribe, contact participant for re-subscription
- **Retry:** No, respect unsubscribe preference

---

## Template Errors

### TEMPLATE_VALIDATION_FAILED

- **HTTP Status:** 400 Bad Request
- **Message:** Template validation failed
- **Cause:** Template has syntax errors or missing required variables
- **Details:** Lists missing variables and invalid syntax
- **Solution:** Fix template variables and syntax
- **Retry:** Yes, with corrected template

### TEMPLATE_NOT_FOUND

- **HTTP Status:** 404 Not Found
- **Message:** Email template not found
- **Cause:** Template ID doesn't exist
- **Solution:** Use valid template ID from list templates endpoint
- **Retry:** No, use correct template ID

### TEMPLATE_INVALID_VARIABLE

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid template variable
- **Cause:** Template uses variable that doesn't exist
- **Solution:** Use valid variable name
- **Retry:** Yes, with valid variable

---

## Check-in Errors

### CHECKIN_QR_CODE_INVALID

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid QR code
- **Cause:** QR code is malformed, expired, or tampered
- **Solution:** Verify QR code format, get new code if needed
- **Retry:** Yes, with valid QR code

### CHECKIN_QR_CODE_NOT_FOUND

- **HTTP Status:** 404 Not Found
- **Message:** QR code not found
- **Cause:** QR code doesn't match any participant
- **Solution:** Verify QR code is for this event
- **Retry:** No, need correct QR code

### CHECKIN_ALREADY_CHECKED_IN

- **HTTP Status:** 409 Conflict
- **Message:** Participant already checked in
- **Cause:** Participant has already been checked in
- **Solution:** No action needed, check-in complete
- **Retry:** No, already checked in

### CHECKIN_EVENT_NOT_STARTED

- **HTTP Status:** 400 Bad Request
- **Message:** Event has not started
- **Cause:** Check-in before event start time
- **Solution:** Wait for event to start, or use force override flag
- **Retry:** Yes, after event start time

### CHECKIN_EVENT_ENDED

- **HTTP Status:** 400 Bad Request
- **Message:** Event has ended
- **Cause:** Check-in after event end time
- **Solution:** Event is over, no more check-ins allowed
- **Retry:** No, event is complete

### CHECKIN_RATE_LIMIT

- **HTTP Status:** 429 Too Many Requests
- **Message:** Check-in rate limit exceeded
- **Cause:** Too many check-ins (see Rate Limiting Strategy)
- **Solution:** Pace check-ins or contact support for higher limits
- **Retry:** Yes, after rate limit window

---

## Job/Async Errors

### JOB_NOT_FOUND

- **HTTP Status:** 404 Not Found
- **Message:** Job not found
- **Cause:** Job ID doesn't exist or was deleted
- **Solution:** Verify job ID, may have expired (7 days)
- **Retry:** No, job not found

### JOB_FAILED

- **HTTP Status:** 500 Internal Server Error
- **Message:** Job processing failed
- **Cause:** Background job encountered critical error
- **Solution:** Contact support with job ID
- **Retry:** No, manual intervention needed

### JOB_EXPIRED

- **HTTP Status:** 410 Gone
- **Message:** Job has expired
- **Cause:** Job data was automatically cleaned up (7 days)
- **Solution:** Re-submit the operation
- **Retry:** Yes, with new request

### JOB_CANCELLED

- **HTTP Status:** 400 Bad Request
- **Message:** Job was cancelled
- **Cause:** Job was manually cancelled by user or admin
- **Solution:** Re-submit if operation still needed
- **Retry:** Yes, with new request

---

## Validation Errors

### VALIDATION_FAILED

- **HTTP Status:** 400 Bad Request
- **Message:** Validation failed
- **Details:** List of validation errors with field names
- **Cause:** Input data doesn't meet requirements
- **Solution:** Fix validation errors and retry
- **Retry:** Yes, with corrected data

### INVALID_REQUEST_BODY

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid request body
- **Cause:** Request body is not valid JSON
- **Solution:** Ensure Content-Type is application/json and JSON is valid
- **Retry:** Yes, with valid JSON

### INVALID_PARAMETER

- **HTTP Status:** 400 Bad Request
- **Message:** Invalid parameter value
- **Cause:** Query parameter or path parameter has invalid value
- **Solution:** Use valid parameter value
- **Retry:** Yes, with valid parameter

### MISSING_REQUIRED_FIELD

- **HTTP Status:** 400 Bad Request
- **Message:** Missing required field
- **Details:** Lists missing field names
- **Cause:** Required field not provided in request
- **Solution:** Include all required fields
- **Retry:** Yes, with required fields

---

## Rate Limiting Errors

### RATE_LIMIT_EXCEEDED

- **HTTP Status:** 429 Too Many Requests
- **Message:** Rate limit exceeded
- **Details:** Shows limit, remaining, reset time
- **Cause:** Too many requests in time window
- **Solution:** Wait and retry later (see Retry-After header)
- **Retry:** Yes, use Retry-After header for timing
- **Headers:**
  - `X-RateLimit-Limit: 100`
  - `X-RateLimit-Remaining: 0`
  - `X-RateLimit-Reset: 1701345600`
  - `Retry-After: 45`

---

## Server Errors

### INTERNAL_SERVER_ERROR

- **HTTP Status:** 500 Internal Server Error
- **Message:** Internal server error
- **Cause:** Unexpected server error
- **Solution:** Retry with exponential backoff, contact support
- **Retry:** Yes, with exponential backoff
- **Details:** Request ID included for support ticket

### SERVICE_UNAVAILABLE

- **HTTP Status:** 503 Service Unavailable
- **Message:** Service temporarily unavailable
- **Cause:** Server maintenance or temporary outage
- **Solution:** Retry later, check status page
- **Retry:** Yes, after waiting

### DATABASE_ERROR

- **HTTP Status:** 500 Internal Server Error
- **Message:** Database error
- **Cause:** Database connection or query error
- **Solution:** Retry later, contact support if persistent
- **Retry:** Yes, with exponential backoff

### RESOURCE_EXHAUSTED

- **HTTP Status:** 503 Service Unavailable
- **Message:** Resource limit exceeded
- **Cause:** Server resources exhausted (memory, connections, etc.)
- **Solution:** Retry later, contact support for high-volume needs
- **Retry:** Yes, after waiting

---

## Common Solutions by Scenario

### "I can't send emails"

1. Check [EMAIL_RATE_LIMIT](#email_rate_limit) - Wait before retrying
2. Check [EMAIL_INVALID_ADDRESS](#email_invalid_address) - Verify email format
3. Check [EMAIL_SEND_FAILED](#email_send_failed) - Retry later
4. Contact support if persists

### "Check-in not working"

1. Check [CHECKIN_QR_CODE_INVALID](#checkin_qr_code_invalid) - Verify QR code
2. Check [CHECKIN_ALREADY_CHECKED_IN](#checkin_already_checked_in) - Already done
3. Check [CHECKIN_EVENT_NOT_STARTED](#checkin_event_not_started) - Wait for start time
4. Check [CHECKIN_RATE_LIMIT](#checkin_rate_limit) - Pace check-ins
5. Contact support if persists

### "Authentication fails"

1. Check [AUTH_INVALID_CREDENTIALS](#auth_invalid_credentials) - Verify password
2. Check [AUTH_ACCOUNT_LOCKED](#auth_account_locked) - Wait 15 minutes
3. Check [AUTH_SESSION_EXPIRED](#auth_session_expired) - Refresh token
4. Check [AUTH_INVALID_TOKEN](#auth_invalid_token) - Login again
5. Contact support if persists

### "API call returns 400/validation error"

1. Check [VALIDATION_FAILED](#validation_failed) - Fix listed errors
2. Check [MISSING_REQUIRED_FIELD](#missing_required_field) - Add required fields
3. Check [INVALID_PARAMETER](#invalid_parameter) - Fix parameter values
4. Consult endpoint documentation for requirements
5. Contact support with details

### "I can't delete a user/event/participant"

1. Check [USER_HAS_ACTIVE_EVENTS](#user_has_active_events) - Complete or cancel events first
2. Check [PARTICIPANT_HAS_PAYMENT](#participant_has_payment) - Use status='cancelled' instead
3. Check [EVENT_IS_ONGOING](#event_is_ongoing) - Wait for event to end or cancel first
4. Check [EVENT_HAS_PAID_PARTICIPANTS](#event_has_paid_participants) - Use force=true with
   confirmation
5. Check [USER_DELETION_FORBIDDEN](#user_deletion_forbidden) - Verify authorization
6. Check [INVALID_CONFIRMATION](#invalid_confirmation) - Include confirm: true in request
7. Check [DELETION_REASON_REQUIRED](#deletion_reason_required) - Provide deletion reason
8. Consult deletion documentation for requirements

---

## Error Code Index

| Code                           | HTTP Status | Category       |
| ------------------------------ | ----------- | -------------- |
| AUTH_INVALID_CREDENTIALS       | 401         | Authentication |
| AUTH_INVALID_TOKEN             | 401         | Authentication |
| AUTH_SESSION_EXPIRED           | 401         | Authentication |
| AUTH_EMAIL_EXISTS              | 409         | Authentication |
| AUTH_WEAK_PASSWORD             | 400         | Authentication |
| AUTH_ACCOUNT_LOCKED            | 403         | Authentication |
| AUTH_UNAUTHORIZED              | 401         | Authentication |
| AUTH_INSUFFICIENT_PERMISSIONS  | 403         | Authentication |
| AUTH_MAGIC_LINK_INVALID        | 401         | Authentication |
| AUTH_MAGIC_LINK_USED           | 401         | Authentication |
| AUTH_RATE_LIMIT                | 429         | Authentication |
| EVENT_NOT_FOUND                | 404         | Event          |
| EVENT_INVALID_STATUS           | 400         | Event          |
| EVENT_DUPLICATE_NAME           | 409         | Event          |
| EVENT_INVALID_DATE_RANGE       | 400         | Event          |
| EVENT_CAPACITY_EXCEEDED        | 400         | Event          |
| USER_HAS_ACTIVE_EVENTS         | 400         | Deletion       |
| USER_ALREADY_DELETED           | 404         | Deletion       |
| USER_DELETION_FORBIDDEN        | 403         | Deletion       |
| PARTICIPANT_HAS_PAYMENT        | 400         | Deletion       |
| EVENT_IS_ONGOING               | 400         | Deletion       |
| EVENT_HAS_PAID_PARTICIPANTS    | 400         | Deletion       |
| DELETION_LOG_FORBIDDEN         | 403         | Deletion       |
| INVALID_CONFIRMATION           | 400         | Deletion       |
| DELETION_REASON_REQUIRED       | 400         | Deletion       |
| PARTICIPANT_NOT_FOUND          | 404         | Participant    |
| PARTICIPANT_DUPLICATE_EMAIL    | 409         | Participant    |
| PARTICIPANT_INVALID_EMAIL      | 400         | Participant    |
| PARTICIPANT_ALREADY_CHECKED_IN | 400         | Participant    |
| PARTICIPANT_NOT_REGISTERED     | 403         | Participant    |
| QR_CODE_GENERATION_FAILED      | 500         | QR Code        |
| QR_CODE_INVALID_FORMAT         | 400         | QR Code        |
| QR_CODE_SIZE_OUT_OF_RANGE      | 400         | QR Code        |
| EMAIL_SEND_FAILED              | 500         | Email          |
| EMAIL_INVALID_ADDRESS          | 400         | Email          |
| EMAIL_RATE_LIMIT               | 429         | Email          |
| EMAIL_BOUNCE                   | 400         | Email          |
| EMAIL_UNSUBSCRIBED             | 403         | Email          |
| TEMPLATE_VALIDATION_FAILED     | 400         | Template       |
| TEMPLATE_NOT_FOUND             | 404         | Template       |
| TEMPLATE_INVALID_VARIABLE      | 400         | Template       |
| CHECKIN_QR_CODE_INVALID        | 400         | Check-in       |
| CHECKIN_QR_CODE_NOT_FOUND      | 404         | Check-in       |
| CHECKIN_ALREADY_CHECKED_IN     | 409         | Check-in       |
| CHECKIN_EVENT_NOT_STARTED      | 400         | Check-in       |
| CHECKIN_EVENT_ENDED            | 400         | Check-in       |
| CHECKIN_RATE_LIMIT             | 429         | Check-in       |
| JOB_NOT_FOUND                  | 404         | Job            |
| JOB_FAILED                     | 500         | Job            |
| JOB_EXPIRED                    | 410         | Job            |
| JOB_CANCELLED                  | 400         | Job            |
| VALIDATION_FAILED              | 400         | Validation     |
| INVALID_REQUEST_BODY           | 400         | Validation     |
| INVALID_PARAMETER              | 400         | Validation     |
| MISSING_REQUIRED_FIELD         | 400         | Validation     |
| RATE_LIMIT_EXCEEDED            | 429         | Rate Limit     |
| INTERNAL_SERVER_ERROR          | 500         | Server         |
| SERVICE_UNAVAILABLE            | 503         | Server         |
| DATABASE_ERROR                 | 500         | Server         |
| RESOURCE_EXHAUSTED             | 503         | Server         |

---

## Related Documentation

- [Schemas: Error Response Format](./schemas.md#error-response-format)
- [Rate Limiting Strategy](./rate_limits.md)
- [Testing Guide: Error Scenario Testing](./testing.md#2-error-scenario-testing)

---

**Last Updated:** 2025-11-22
