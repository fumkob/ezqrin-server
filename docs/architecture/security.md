# Security Design

## Overview

Security is a fundamental aspect of ezQRin's architecture. This document outlines authentication,
authorization, data protection, and security best practices implemented throughout the system.

---

## Authentication

### JWT-Based Authentication

**Token Types:**

- **Access Token:** Short-lived (15 minutes), used for API requests
- **Refresh Token:** Long-lived (7 days for web, 90 days for mobile), used to obtain new access
  tokens

**Token Flow:**

```
1. User Login → Validate Credentials
2. Generate Access Token (15 min expiry)
3. Generate Refresh Token (7 day expiry)
4. Return Both Tokens
5. Client Stores Tokens Securely
6. Client Sends Access Token with Each Request
7. Server Validates Token on Each Request
8. Before Access Token Expires → Refresh
```

---

### Token Structure

**Access Token Payload:**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "organizer",
  "iat": 1699459200,
  "exp": 1699460100
}
```

**Token Claims:**

| Claim   | Description                         |
| ------- | ----------------------------------- |
| user_id | Unique user identifier (UUID)       |
| email   | User email address                  |
| role    | User role (admin, organizer, staff) |
| iat     | Issued at timestamp (Unix)          |
| exp     | Expiration timestamp (Unix)         |

---

### Token Generation

**Algorithm:** HS256 (HMAC with SHA-256)

**Process:**

```go
import "github.com/golang-jwt/jwt/v5"

func GenerateAccessToken(user *User, secret string) (string, error) {
    claims := jwt.MapClaims{
        "user_id": user.ID,
        "email":   user.Email,
        "role":    user.Role,
        "iat":     time.Now().Unix(),
        "exp":     time.Now().Add(15 * time.Minute).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

---

### Token Validation

**Validation Steps:**

1. Extract token from Authorization header
2. Parse and verify signature
3. Check expiration time
4. Validate required claims
5. Extract user information

**Middleware Implementation:**

```go
func AuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := extractToken(c.Request)

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, errors.New("invalid signing method")
            }
            return []byte(secret), nil
        })

        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["user_id"])
        c.Set("role", claims["role"])
        c.Next()
    }
}
```

---

### Token Storage

**Client-Side Storage:**

**Web Applications:**

- HTTP-only cookies (recommended)
- Prevents XSS access to tokens
- Automatic transmission with requests

```go
http.SetCookie(w, &http.Cookie{
    Name:     "access_token",
    Value:    token,
    HttpOnly: true,
    Secure:   true, // HTTPS only
    SameSite: http.SameSiteStrictMode,
    MaxAge:   900, // 15 minutes
})
```

**Mobile Applications:**

- iOS: Keychain
- Android: EncryptedSharedPreferences
- Never store in plain text

**Single Page Applications (SPA):**

- Memory storage (most secure, lost on refresh)
- sessionStorage (cleared on tab close)
- Avoid localStorage (XSS vulnerable)

---

## Authorization

### Role-Based Access Control (RBAC)

**User Roles:**

| Role      | Permissions                                           | Use Case              |
| --------- | ----------------------------------------------------- | --------------------- |
| admin     | Full system access, manage all resources              | System administrators |
| organizer | Create and manage own events, participants, check-ins | Event creators        |
| staff     | Perform check-ins for assigned events                 | Event staff           |

---

### Permission Matrix

| Resource            | Admin | Organizer      | Staff               |
| ------------------- | ----- | -------------- | ------------------- |
| Create Event        | ✅    | ✅             | ❌                  |
| Edit Own Event      | ✅    | ✅             | ❌                  |
| Edit Any Event      | ✅    | ❌             | ❌                  |
| Delete Event        | ✅    | ✅ (own)       | ❌                  |
| Add Participant     | ✅    | ✅ (own event) | ❌                  |
| Import Participants | ✅    | ✅ (own event) | ❌                  |
| View Participants   | ✅    | ✅ (own event) | ✅ (assigned event) |
| Perform Check-in    | ✅    | ✅ (own event) | ✅ (assigned event) |
| Cancel Check-in     | ✅    | ✅ (own event) | ❌                  |
| Generate QR Codes   | ✅    | ✅ (own event) | ❌                  |
| View Statistics     | ✅    | ✅ (own event) | ✅ (assigned event) |

---

### Resource-Level Authorization

**Event Ownership Check:**

```go
func (h *EventHandler) UpdateEvent(c *gin.Context) {
    eventID := c.Param("id")
    userID := c.GetString("user_id")
    role := c.GetString("role")

    event, err := h.eventRepo.FindByID(c.Request.Context(), eventID)
    if err != nil {
        c.JSON(404, gin.H{"error": "event not found"})
        return
    }

    // Admins can update any event, organizers only their own
    if role != "admin" && event.OrganizerID != userID {
        c.JSON(403, gin.H{"error": "forbidden"})
        return
    }

    // Proceed with update
}
```

**Staff Assignment Check:**

```go
func (h *EventHandler) GetEvent(c *gin.Context) {
    eventID := c.Param("id")
    userID := c.GetString("user_id")
    role := c.GetString("role")

    event, err := h.eventRepo.FindByID(c.Request.Context(), eventID)
    if err != nil {
        c.JSON(404, gin.H{"error": "event not found"})
        return
    }

    // Check access permissions
    hasAccess := false

    switch role {
    case "admin":
        hasAccess = true
    case "organizer":
        hasAccess = event.OrganizerID == userID
    case "staff":
        // Check if staff is assigned to this event
        isAssigned, err := h.staffRepo.CheckAssignment(c.Request.Context(), eventID, userID)
        if err != nil {
            c.JSON(500, gin.H{"error": "internal server error"})
            return
        }
        hasAccess = isAssigned
    }

    if !hasAccess {
        c.JSON(403, gin.H{"error": "forbidden"})
        return
    }

    // Return event data
}
```

**List Events with Staff Filtering:**

```go
func (h *EventHandler) ListEvents(c *gin.Context) {
    userID := c.GetString("user_id")
    role := c.GetString("role")

    var events []*Event
    var err error

    switch role {
    case "admin":
        // Admins see all events
        events, err = h.eventRepo.FindAll(c.Request.Context())
    case "organizer":
        // Organizers see only their events
        events, err = h.eventRepo.FindByOrganizerID(c.Request.Context(), userID)
    case "staff":
        // Staff see only assigned events
        events, err = h.eventRepo.FindByStaffID(c.Request.Context(), userID)
    default:
        c.JSON(403, gin.H{"error": "forbidden"})
        return
    }

    if err != nil {
        c.JSON(500, gin.H{"error": "internal server error"})
        return
    }

    c.JSON(200, gin.H{"data": events})
}
```

**Participant Access Control:**

```go
func (h *ParticipantHandler) GetParticipant(c *gin.Context) {
    participantID := c.Param("pid")
    userID := c.GetString("user_id")
    role := c.GetString("role")

    participant, err := h.participantRepo.FindByID(c.Request.Context(), participantID)
    if err != nil {
        c.JSON(404, gin.H{"error": "participant not found"})
        return
    }

    event, _ := h.eventRepo.FindByID(c.Request.Context(), participant.EventID)

    // Check access through event ownership or staff assignment
    hasAccess := false

    switch role {
    case "admin":
        hasAccess = true
    case "organizer":
        hasAccess = event.OrganizerID == userID
    case "staff":
        // Staff can access participants only for assigned events
        isAssigned, err := h.staffRepo.CheckAssignment(c.Request.Context(), event.ID, userID)
        if err != nil {
            c.JSON(500, gin.H{"error": "internal server error"})
            return
        }
        hasAccess = isAssigned
    }

    if !hasAccess {
        c.JSON(403, gin.H{"error": "forbidden"})
        return
    }

    // Return participant data
    c.JSON(200, gin.H{"data": participant})
}
```

**Check-in Access Control:**

```go
func (h *CheckinHandler) CreateCheckin(c *gin.Context) {
    eventID := c.Param("id")
    userID := c.GetString("user_id")
    role := c.GetString("role")

    event, err := h.eventRepo.FindByID(c.Request.Context(), eventID)
    if err != nil {
        c.JSON(404, gin.H{"error": "event not found"})
        return
    }

    // Check if user can perform check-in
    canCheckin := false

    switch role {
    case "admin":
        canCheckin = true
    case "organizer":
        canCheckin = event.OrganizerID == userID
    case "staff":
        // Staff can check-in only for assigned events
        isAssigned, err := h.staffRepo.CheckAssignment(c.Request.Context(), eventID, userID)
        if err != nil {
            c.JSON(500, gin.H{"error": "internal server error"})
            return
        }
        canCheckin = isAssigned
    }

    if !canCheckin {
        c.JSON(403, gin.H{"error": "forbidden"})
        return
    }

    // Proceed with check-in
}
```

---

## Password Security

### Password Hashing

**Algorithm:** bcrypt with cost factor 12

**Implementation:**

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword(
        []byte(password),
        12, // cost factor
    )
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword(
        []byte(hash),
        []byte(password),
    )
    return err == nil
}
```

**Cost Factor:**

- Cost 12 ≈ 250ms on modern hardware
- Balances security and user experience
- Adjustable based on hardware improvements

---

### Password Requirements

**Enforced Rules:**

- Minimum 8 characters
- Maximum 128 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- Special characters recommended

**Validation:**

```go
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    if len(password) > 128 {
        return errors.New("password must not exceed 128 characters")
    }

    var (
        hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
        hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
        hasNumber  = regexp.MustCompile(`[0-9]`).MatchString(password)
    )

    if !hasUpper || !hasLower || !hasNumber {
        return errors.New("password must contain uppercase, lowercase, and numbers")
    }

    return nil
}
```

---

### Password Storage

**Security Measures:**

- Never store passwords in plain text
- Hash generated on registration/password change
- Original password discarded immediately
- Hash never transmitted or logged

---

## QR Code Security

### Token Design

**Format:** `evt_{event_id}_prt_{participant_id}_{random_token}`

**Components:**

- Event ID (shortened UUID)
- Participant ID (shortened UUID)
- Random token (12 chars, cryptographically secure)

**Generation:**

```go
import "crypto/rand"

func GenerateQRCode(eventID, participantID uuid.UUID) (string, error) {
    randomBytes := make([]byte, 9) // 9 bytes = 12 base64 chars
    _, err := rand.Read(randomBytes)
    if err != nil {
        return "", err
    }

    randomToken := base64.URLEncoding.EncodeToString(randomBytes)
    qrCode := fmt.Sprintf(
        "evt_%s_prt_%s_%s",
        eventID.String()[:8],
        participantID.String()[:8],
        randomToken[:12],
    )

    return qrCode, nil
}
```

---

### QR Code Validation

**Validation Process:**

1. Parse QR code format
2. Extract event ID and participant ID
3. Query database for participant
4. Verify QR code matches stored value
5. Check event status (must be published or ongoing)
6. Check participant status (not cancelled)
7. Verify not already checked in

**Security Checks:**

```go
func ValidateQRCode(qrCode string) (*Participant, error) {
    // Parse QR code
    parts := strings.Split(qrCode, "_")
    if len(parts) != 5 || parts[0] != "evt" || parts[2] != "prt" {
        return nil, errors.New("invalid QR code format")
    }

    eventIDShort := parts[1]
    participantIDShort := parts[3]

    // Find participant by QR code
    participant, err := participantRepo.FindByQRCode(ctx, qrCode)
    if err != nil {
        return nil, errors.New("invalid QR code")
    }

    // Verify QR code hasn't been tampered
    if participant.QRCode != qrCode {
        return nil, errors.New("QR code mismatch")
    }

    // Additional business logic validation...
    return participant, nil
}
```

---

### QR Code Expiration

**Strategy:** QR codes remain valid until event completion

**Future Enhancement:** Time-based expiration

- Set expiration date (e.g., 1 day after event)
- Include expiration in QR payload
- Validate timestamp on check-in

---

## Data Protection

### Input Validation

**All User Input Validated:**

- Type validation (string, int, UUID, etc.)
- Format validation (email, phone, URL)
- Length validation (min/max constraints)
- Content validation (allowed characters)

**Validation Library:**

```go
import "github.com/go-playground/validator/v10"

type CreateEventRequest struct {
    Name        string    `json:"name" validate:"required,min=1,max=255"`
    Description string    `json:"description" validate:"max=5000"`
    StartDate   time.Time `json:"start_date" validate:"required"`
    EndDate     time.Time `json:"end_date" validate:"omitempty,gtfield=StartDate"`
    Location    string    `json:"location" validate:"max=500"`
    Timezone    string    `json:"timezone" validate:"omitempty,timezone"`
}
```

---

### SQL Injection Prevention

**Prepared Statements:**

- All database queries use parameterized queries
- User input never concatenated into SQL strings
- pgx driver handles escaping automatically

**Example:**

```go
// Safe - parameterized query
query := `SELECT * FROM events WHERE id = $1 AND organizer_id = $2`
row := db.QueryRow(ctx, query, eventID, userID)

// NEVER DO THIS - SQL injection vulnerable
query := fmt.Sprintf("SELECT * FROM events WHERE id = '%s'", eventID)
```

---

### XSS Protection

**API-Only Architecture:**

- Server returns JSON, not HTML
- No server-side rendering of user content
- Client responsibility to sanitize before DOM insertion

**HTTP Headers:**

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
```

---

### CSRF Protection

**Token-Based API:**

- Stateless JWT authentication
- No session cookies (or HTTP-only if used)
- CORS properly configured

**CORS Configuration:**

```go
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "https://app.ezqrin.com")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

---

## Rate Limiting

### API Rate Limits

**Limits by Endpoint Type:**

| Endpoint Type  | Limit         | Window               | Action on Exceed                    |
| -------------- | ------------- | -------------------- | ----------------------------------- |
| Standard API   | 100 requests  | 1 minute             | 429 Too Many Requests               |
| Authentication | 5 attempts    | 15 minutes           | 429 + Account lock after 5 failures |
| Check-in       | 50 requests   | 1 minute per event   | 429 Too Many Requests               |
| Email sending  | 100 emails    | 1 hour per organizer | 429 Too Many Requests               |
| QR generation  | 1000 requests | 1 hour per event     | 429 Too Many Requests               |

---

### Rate Limit Implementation

**Redis-Based Rate Limiting:**

```go
func RateLimitMiddleware(redisClient *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := fmt.Sprintf("ratelimit:%s", c.ClientIP())

        count, err := redisClient.Incr(c.Request.Context(), key).Result()
        if err != nil {
            c.Next()
            return
        }

        if count == 1 {
            redisClient.Expire(c.Request.Context(), key, 1*time.Minute)
        }

        if count > 100 {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "rate limit exceeded",
            })
            return
        }

        c.Header("X-RateLimit-Limit", "100")
        c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", 100-count))
        c.Next()
    }
}
```

---

### Account Lockout

**Failed Login Protection:**

- Track failed login attempts per email
- Lock account after 5 consecutive failures
- Lockout duration: 15 minutes
- Clear counter on successful login

**Implementation:**

```go
func CheckLoginAttempts(email string, redisClient *redis.Client) error {
    key := fmt.Sprintf("login_attempts:%s", email)
    attempts, _ := redisClient.Get(ctx, key).Int()

    if attempts >= 5 {
        return errors.New("account temporarily locked")
    }

    return nil
}

func RecordFailedLogin(email string, redisClient *redis.Client) {
    key := fmt.Sprintf("login_attempts:%s", email)
    redisClient.Incr(ctx, key)
    redisClient.Expire(ctx, key, 15*time.Minute)
}
```

---

## Communication Security

### TLS/HTTPS

**Requirements:**

- TLS 1.3 minimum
- Valid SSL certificate (Let's Encrypt or commercial CA)
- HTTP Strict Transport Security (HSTS)
- Redirect HTTP to HTTPS

**HSTS Header:**

```
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

---

### API Security Headers

**Standard Security Headers:**

```go
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'none'")
        c.Header("Referrer-Policy", "no-referrer")
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        c.Next()
    }
}
```

---

## Logging & Monitoring

### Security Event Logging

**Logged Events:**

- All authentication attempts (success/failure)
- Authorization failures (403 responses)
- Rate limit violations
- Invalid token usage
- Suspicious activity patterns
- **Deletion operations** (all entity deletions)

**Log Format:**

```json
{
  "timestamp": "2025-11-08T10:00:00Z",
  "level": "WARN",
  "event": "authentication_failed",
  "email": "user@example.com",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "reason": "invalid_credentials"
}
```

---

### Deletion Audit Logging

**Purpose:**

- Compliance requirements (data protection, GDPR)
- Security auditing and forensics
- Troubleshooting and debugging
- Financial record protection

**Deletion Audit Log Structure:**

All deletion operations are automatically logged to `deletion_audit_log` table:

```json
{
  "id": "audit-log-uuid",
  "entity_type": "event",
  "entity_id": "deleted-entity-uuid",
  "entity_snapshot": {
    "name": "Tech Conference 2025",
    "start_date": "2025-12-15T09:00:00Z",
    "participants_count": 150
  },
  "deleted_by": "user-uuid",
  "deleted_at": "2025-11-08T15:30:00Z",
  "deletion_type": "hard",
  "deletion_reason": "Event cancelled",
  "cascade_effects": {
    "participants_deleted": 150,
    "checkins_deleted": 87
  },
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0..."
}
```

**Logged Deletion Types:**

| Type        | Description                     | Entities                        |
| ----------- | ------------------------------- | ------------------------------- |
| `hard`      | Physical deletion from database | Events, Participants, Check-ins |
| `soft`      | Logical deletion with flag      | (Reserved for future)           |
| `anonymize` | Soft delete + PII anonymization | Users                           |

**Retention Period:** 3 years

**Access Control:** Admin only via `GET /api/v1/deletion-logs`

---

### Sensitive Data Handling

**Never Log:**

- Passwords (plain or hashed)
- Full credit card numbers
- JWT tokens
- QR codes (log IDs only)
- Personal identification numbers

**Deletion Audit Log - Included Data:**

- Entity snapshots (names, emails, event details)
- Deletion reasons and timestamps
- IP addresses and user agents
- Cascade effect counts (not actual deleted data)

**Deletion Audit Log - Excluded Data:**

- Password hashes (never included in snapshots)
- JWT tokens or session IDs
- Full payment card numbers
- QR code values (only invalidation status)

**User Anonymization Process:**

When a user is deleted, PII is irreversibly anonymized:

```go
// Before anonymization
user.Name = "John Doe"
user.Email = "john@example.com"
user.PasswordHash = "$2a$12$..."

// After anonymization
user.Name = "Deleted User a3f5b2c1"
user.Email = "deleted_a3f5b2c1@anonymized.local"
user.PasswordHash = "[random hash - login disabled]"
user.IsAnonymized = true
user.DeletedAt = NOW()
```

**Anonymization Guarantees:**

- Original data cannot be recovered (irreversible)
- Unique constraint preserved (`deleted_[token]@anonymized.local`)
- Login permanently disabled (password hash replaced)
- Events remain accessible with anonymized owner

**Log Sanitization:**

```go
func SanitizeForLog(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "[invalid-email]"
    }
    return parts[0][:1] + "***@" + parts[1]
}
```

---

## Compliance & Best Practices

### OWASP Top 10 Protection

| Risk                                        | Protection Mechanism                        |
| ------------------------------------------- | ------------------------------------------- |
| Injection                                   | Parameterized queries, input validation     |
| Broken Authentication                       | JWT, bcrypt, rate limiting, MFA (future)    |
| Sensitive Data Exposure                     | TLS, encrypted storage, minimal logging     |
| XML External Entities                       | N/A (JSON-only API)                         |
| Broken Access Control                       | RBAC, resource ownership checks             |
| Security Misconfiguration                   | Security headers, minimal permissions       |
| XSS                                         | API-only, proper headers                    |
| Insecure Deserialization                    | Safe JSON parsing, validation               |
| Using Components with Known Vulnerabilities | Dependency scanning, regular updates        |
| Insufficient Logging & Monitoring           | Structured logging, security event tracking |

---

### Data Privacy

**GDPR Compliance (if applicable):**

- User consent for data processing
- Right to access personal data
- Right to deletion (account deletion)
- Data portability (export functionality)
- Privacy policy and terms of service

**Data Minimization:**

- Collect only necessary information
- No tracking or analytics without consent
- Limited data retention periods

---

## Future Enhancements

### Planned Security Features

**Multi-Factor Authentication (MFA):**

- TOTP (Time-based One-Time Password)
- SMS verification
- Email verification codes

**Advanced Audit Logging:**

- Comprehensive action audit trail
- Data access logs
- Export for compliance

**Enhanced QR Security:**

- Time-based QR codes
- Single-use tokens
- Biometric verification integration

**API Key Management:**

- API keys for programmatic access
- Key rotation policies
- Granular permissions

---

## Related Documentation

- [System Architecture](./overview.md)
- [Database Design](./database.md)
- [Authentication API](../api/authentication.md)
