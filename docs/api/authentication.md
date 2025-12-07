# Authentication API

## Overview

The ezQRin API uses JWT (JSON Web Token) based authentication. Access tokens are short-lived (15
minutes) while refresh tokens have a longer lifespan (7 days).

## Endpoints

### Register User

Create a new user account.

**Endpoint:** `POST /api/v1/auth/register`

**Headers:**

```
Content-Type: application/json
```

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "name": "John Doe",
  "role": "organizer"
}
```

**Request Fields:**

| Field    | Type   | Required | Description                                                          |
| -------- | ------ | -------- | -------------------------------------------------------------------- |
| email    | string | Yes      | Valid email address                                                  |
| password | string | Yes      | Minimum 8 characters, must include uppercase, lowercase, and numbers |
| name     | string | Yes      | Full name (1-255 characters)                                         |
| role     | string | No       | User role: `organizer` (default), `staff`, `admin`                   |

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "organizer",
      "created_at": "2025-11-08T10:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900
  },
  "message": "User registered successfully"
}
```

**Errors:**

- `400 Bad Request` - Invalid request data or validation failed
- `409 Conflict` - Email already registered

---

### Login

Authenticate a user and receive access tokens.

**Endpoint:** `POST /api/v1/auth/login`

**Headers:**

```
Content-Type: application/json
```

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "client_type": "mobile"
}
```

**Request Fields:**

| Field       | Type   | Required | Description                                 |
| ----------- | ------ | -------- | ------------------------------------------- |
| email       | string | Yes      | Registered email address                    |
| password    | string | Yes      | User password                               |
| client_type | string | No       | Client type: `web`, `mobile` (default: web) |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "organizer"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900
  },
  "message": "Login successful"
}
```

**Errors:**

- `401 Unauthorized` - Invalid credentials
- `429 Too Many Requests` - Rate limit exceeded (5 attempts per 15 minutes)

---

### Refresh Token

Obtain a new access token using a valid refresh token.

**Endpoint:** `POST /api/v1/auth/refresh`

**Headers:**

```
Content-Type: application/json
```

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Request Fields:**

| Field         | Type   | Required | Description                             |
| ------------- | ------ | -------- | --------------------------------------- |
| refresh_token | string | Yes      | Valid refresh token from login/register |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900
  },
  "message": "Token refreshed successfully"
}
```

**Errors:**

- `401 Unauthorized` - Invalid or expired refresh token

---

### Logout

Invalidate the current access and refresh tokens.

**Endpoint:** `POST /api/v1/auth/logout`

**Headers:**

```
Authorization: Bearer <access_token>
```

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "message": "Logout successful"
}
```

**Errors:**

- `401 Unauthorized` - Invalid access token

---

## Token Usage

### Access Token

Include the access token in the Authorization header for all protected endpoints:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Token Payload

JWT tokens contain the following claims:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "organizer",
  "exp": 1636372800,
  "iat": 1636371900
}
```

### Token Expiration

**Access Token:**

- All clients: 15 minutes (900 seconds)

**Refresh Token:**

- Web clients: 7 days (604,800 seconds)
- Mobile clients: 90 days (7,776,000 seconds)

### Refresh Strategy

Implement token refresh before expiration:

1. Monitor access token expiration time
2. Call `/auth/refresh` endpoint with refresh token before access token expires
3. Update stored tokens with new values
4. If refresh token expires, redirect user to login

---

## Authorization

### User Roles

| Role        | Permissions                                                                     |
| ----------- | ------------------------------------------------------------------------------- |
| `admin`     | Full system access, manage all events, users, and staff assignments             |
| `organizer` | Create and manage own events, participants, check-ins, and assign staff         |
| `staff`     | View assigned events, participants, and perform check-ins for those events only |

### Resource Access Control

- **Events:**
  - Admin: All events
  - Organizer: Only events they created (organizer_id match)
  - Staff: Only events they are assigned to (via event_staff_assignments)

- **Participants:**
  - Admin: All participants
  - Organizer: Participants for their events
  - Staff: Participants only for assigned events

- **Check-ins:**
  - Admin: Can perform check-ins for any event
  - Organizer: Can perform check-ins for their events
  - Staff: Can perform check-ins only for assigned events

- **Staff Management:**
  - Admin: Can assign/remove staff for any event
  - Organizer: Can assign/remove staff for their own events
  - Staff: Cannot manage staff assignments

---

## Security Best Practices

1. **Store tokens securely:**
   - Use secure, HTTP-only cookies for web applications
   - Use secure storage (Keychain/Keystore) for mobile apps

2. **Handle token expiration:**
   - Implement automatic token refresh
   - Handle 401 responses gracefully

3. **Password requirements:**
   - Minimum 8 characters
   - At least one uppercase letter
   - At least one lowercase letter
   - At least one number
   - Special characters recommended

4. **Rate limiting:**
   - Login attempts: 5 per 15 minutes per IP
   - Register attempts: 10 per hour per IP
   - See [Rate Limiting Strategy](./rate_limits.md) for comprehensive API rate limits

---

## Error Codes

| Code                       | Message                             | Description                       |
| -------------------------- | ----------------------------------- | --------------------------------- |
| `AUTH_INVALID_CREDENTIALS` | Invalid email or password           | Login credentials are incorrect   |
| `AUTH_ACCOUNT_LOCKED`      | Account temporarily locked          | Too many failed login attempts    |
| `AUTH_INVALID_TOKEN`       | Invalid or expired token            | Token is malformed or expired     |
| `AUTH_EMAIL_EXISTS`        | Email already registered            | Email is already in use           |
| `AUTH_WEAK_PASSWORD`       | Password does not meet requirements | Password is too weak              |
| `AUTH_UNAUTHORIZED`        | Unauthorized access                 | Missing or invalid authentication |

---

## Related Documentation

- [User Account Management](./users.md) - User profile management and account deletion
- [Rate Limiting Strategy](./rate_limits.md) - API rate limits
- [Error Codes](./error_codes.md) - Complete error code reference
- [Security Design](../architecture/security.md) - System security architecture
