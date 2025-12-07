# Database Design

## Overview

ezQRin uses PostgreSQL 18 as the primary relational database. The schema is designed for data
integrity, query performance, and scalability.

---

## Database Schema

### Entity Relationship Diagram

```
┌──────────────┐
│    users     │
└──────┬───────┘
       │ 1             ┌──────────────────────────┐
       │               │ event_staff_assignments  │
       │ N             └────────┬─────────────────┘
┌──────▼───────┐         N     │ N
│    events    │───────────────┘
└──────┬───────┘ 1
       │                 ┌──────────────┐
       │ N               │   checkins   │
       │                 └──────┬───────┘
       │ 1                      │ N
       │                        │
       │ N                      │ 1
┌──────▼──────────┐             │
│  participants   │─────────────┘
└─────────────────┘
```

---

## Table Definitions

### users

Stores system users who can create and manage events.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'organizer',
    deleted_at TIMESTAMP NULL,
    deleted_by UUID REFERENCES users(id) NULL,
    is_anonymized BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
```

**Columns:**

| Column        | Type         | Constraints                            | Description                        |
| ------------- | ------------ | -------------------------------------- | ---------------------------------- |
| id            | UUID         | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique user identifier             |
| email         | VARCHAR(255) | UNIQUE, NOT NULL                       | User email address                 |
| password_hash | VARCHAR(255) | NOT NULL                               | bcrypt hashed password (cost=12)   |
| name          | VARCHAR(255) | NOT NULL                               | User full name                     |
| role          | VARCHAR(50)  | NOT NULL, DEFAULT 'organizer'          | User role: admin, organizer, staff |
| deleted_at    | TIMESTAMP    | NULL                                   | Soft delete timestamp              |
| deleted_by    | UUID         | REFERENCES users(id), NULL             | User who performed deletion        |
| is_anonymized | BOOLEAN      | DEFAULT false                          | PII anonymization flag             |
| created_at    | TIMESTAMP    | NOT NULL, DEFAULT NOW()                | Record creation time               |
| updated_at    | TIMESTAMP    | NOT NULL, DEFAULT NOW()                | Record last update time            |

**Indexes:**

- `idx_users_email` - Fast email lookup for authentication
- `idx_users_role` - Filter users by role
- `idx_users_deleted_at` - Partial index for soft-deleted users (non-NULL only)

**Business Rules:**

- Email must be unique across the system
- Password stored as bcrypt hash (never plain text)
- Role determines system permissions
- Soft delete: `deleted_at` set when user is deleted
- PII anonymization: When deleted, `name` and `email` are anonymized, `is_anonymized` set to true
- Deleted users remain in database to preserve event ownership

---

### events

Stores event information created by organizers.

```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    location VARCHAR(500),
    timezone VARCHAR(100) DEFAULT 'Asia/Tokyo',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_organizer_id ON events(organizer_id);
CREATE INDEX idx_events_start_date ON events(start_date);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_created_at ON events(created_at);
```

**Columns:**

| Column       | Type         | Constraints                                      | Description                          |
| ------------ | ------------ | ------------------------------------------------ | ------------------------------------ |
| id           | UUID         | PRIMARY KEY, DEFAULT gen_random_uuid()           | Unique event identifier              |
| organizer_id | UUID         | NOT NULL, REFERENCES users(id) ON DELETE CASCADE | Event creator/owner                  |
| name         | VARCHAR(255) | NOT NULL                                         | Event name                           |
| description  | TEXT         | -                                                | Event description (unlimited length) |
| start_date   | TIMESTAMP    | NOT NULL                                         | Event start date and time            |
| end_date     | TIMESTAMP    | -                                                | Event end date and time              |
| location     | VARCHAR(500) | -                                                | Event venue or location              |
| timezone     | VARCHAR(100) | DEFAULT 'Asia/Tokyo'                             | IANA timezone identifier             |
| status       | VARCHAR(50)  | NOT NULL, DEFAULT 'draft'                        | Event status                         |
| created_at   | TIMESTAMP    | NOT NULL, DEFAULT NOW()                          | Record creation time                 |
| updated_at   | TIMESTAMP    | NOT NULL, DEFAULT NOW()                          | Record last update time              |

**Indexes:**

- `idx_events_organizer_id` - Find events by organizer
- `idx_events_start_date` - Sort/filter by event date
- `idx_events_status` - Filter by event status
- `idx_events_created_at` - Sort by creation date

**Business Rules:**

- Each event belongs to one organizer
- Deleting organizer cascades to their events
- Status: draft, published, ongoing, completed, cancelled
- end_date must be after start_date (enforced in application layer)

---

### participants

Stores event participants and their registration information.

```sql
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    employee_id VARCHAR(255),
    phone VARCHAR(50),
    qr_email VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'tentative',
    qr_code VARCHAR(255) UNIQUE NOT NULL,
    qr_code_generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB,
    payment_status VARCHAR(50) DEFAULT 'unpaid',
    payment_amount NUMERIC(10, 2),
    payment_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_event_email UNIQUE(event_id, email)
);

CREATE INDEX idx_participants_event_id ON participants(event_id);
CREATE INDEX idx_participants_employee_id ON participants(employee_id);
CREATE INDEX idx_participants_email ON participants(email);
CREATE INDEX idx_participants_qr_email ON participants(qr_email) WHERE qr_email IS NOT NULL;
CREATE INDEX idx_participants_qr_code ON participants(qr_code);
CREATE INDEX idx_participants_status ON participants(status);
CREATE INDEX idx_participants_payment_status ON participants(payment_status);
CREATE INDEX idx_participants_created_at ON participants(created_at);
CREATE INDEX idx_participants_metadata ON participants USING gin(metadata);
```

**Columns:**

| Column               | Type          | Constraints                                       | Description                      |
| -------------------- | ------------- | ------------------------------------------------- | -------------------------------- |
| id                   | UUID          | PRIMARY KEY, DEFAULT gen_random_uuid()            | Unique participant identifier    |
| event_id             | UUID          | NOT NULL, REFERENCES events(id) ON DELETE CASCADE | Associated event                 |
| name                 | VARCHAR(255)  | NOT NULL                                          | Participant full name            |
| email                | VARCHAR(255)  | NOT NULL                                          | Primary participant email        |
| employee_id          | VARCHAR(255)  | -                                                 | Employee or staff ID             |
| phone                | VARCHAR(50)   | -                                                 | Participant phone (E.164 format) |
| qr_email             | VARCHAR(255)  | -                                                 | Alternative email for QR code    |
| status               | VARCHAR(50)   | NOT NULL, DEFAULT 'tentative'                     | Participation status             |
| qr_code              | VARCHAR(255)  | UNIQUE, NOT NULL                                  | Unique QR code token             |
| qr_code_generated_at | TIMESTAMP     | NOT NULL, DEFAULT NOW()                           | QR generation timestamp          |
| metadata             | JSONB         | -                                                 | Custom participant data          |
| payment_status       | VARCHAR(50)   | DEFAULT 'unpaid'                                  | Payment status: unpaid, paid     |
| payment_amount       | NUMERIC(10,2) | -                                                 | Payment amount (nullable)        |
| payment_date         | TIMESTAMP     | -                                                 | Payment date (nullable)          |
| created_at           | TIMESTAMP     | NOT NULL, DEFAULT NOW()                           | Record creation time             |
| updated_at           | TIMESTAMP     | NOT NULL, DEFAULT NOW()                           | Record last update time          |

**Indexes:**

- `idx_participants_event_id` - Find participants by event
- `idx_participants_employee_id` - Search by employee ID
- `idx_participants_email` - Search by primary email
- `idx_participants_qr_email` - Search by QR email (partial index, only non-NULL values)
- `idx_participants_qr_code` - Fast QR code lookup (critical for check-in)
- `idx_participants_status` - Filter by status
- `idx_participants_payment_status` - Filter by payment status
- `idx_participants_created_at` - Sort by registration date
- `idx_participants_metadata` - GIN index for JSONB queries

**Constraints:**

- `unique_event_email` - One email per event (prevents duplicate registrations)
- `qr_code` UNIQUE - Each QR code is globally unique

**Business Rules:**

- Participant belongs to one event
- Employee ID is optional; can be used for internal organization/tracking
- Primary email unique within event scope (can register for multiple events)
- QR email is optional; if NULL, QR code sent to primary email
- QR code globally unique across all events
- Status: tentative, confirmed, cancelled, declined
- Payment status: unpaid, paid (independent from participation status)
- Payment amount and date are optional (nullable) supplementary information
- Metadata stores custom fields (max 10KB)

---

### checkins

Stores participant check-in records.

```sql
CREATE TABLE checkins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    checked_in_at TIMESTAMP NOT NULL DEFAULT NOW(),
    checked_in_by UUID REFERENCES users(id),
    checkin_method VARCHAR(50) NOT NULL DEFAULT 'qrcode',
    device_info JSONB,

    CONSTRAINT unique_event_participant_checkin UNIQUE(event_id, participant_id)
);

CREATE INDEX idx_checkins_event_id ON checkins(event_id);
CREATE INDEX idx_checkins_participant_id ON checkins(participant_id);
CREATE INDEX idx_checkins_checked_in_at ON checkins(checked_in_at);
CREATE INDEX idx_checkins_checked_in_by ON checkins(checked_in_by);
```

**Columns:**

| Column         | Type        | Constraints                                             | Description                         |
| -------------- | ----------- | ------------------------------------------------------- | ----------------------------------- |
| id             | UUID        | PRIMARY KEY, DEFAULT gen_random_uuid()                  | Unique check-in identifier          |
| event_id       | UUID        | NOT NULL, REFERENCES events(id) ON DELETE CASCADE       | Associated event                    |
| participant_id | UUID        | NOT NULL, REFERENCES participants(id) ON DELETE CASCADE | Checked-in participant              |
| checked_in_at  | TIMESTAMP   | NOT NULL, DEFAULT NOW()                                 | Check-in timestamp                  |
| checked_in_by  | UUID        | REFERENCES users(id)                                    | User who performed check-in         |
| checkin_method | VARCHAR(50) | NOT NULL, DEFAULT 'qrcode'                              | Method: qrcode, manual              |
| device_info    | JSONB       | -                                                       | Device metadata (OS, version, etc.) |

**Indexes:**

- `idx_checkins_event_id` - Find check-ins by event
- `idx_checkins_participant_id` - Find check-in by participant
- `idx_checkins_checked_in_at` - Sort by check-in time
- `idx_checkins_checked_in_by` - Track who performed check-ins

**Constraints:**

- `unique_event_participant_checkin` - One check-in per participant per event

**Business Rules:**

- One check-in per participant per event
- Deleting event or participant cascades to check-in records
- checked_in_by can be NULL (self-service kiosk)

---

### event_staff_assignments

Stores staff assignments to events, enabling role-based access control for staff users.

```sql
CREATE TABLE event_staff_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),

    CONSTRAINT unique_event_staff UNIQUE(event_id, staff_id)
);

CREATE INDEX idx_event_staff_event_id ON event_staff_assignments(event_id);
CREATE INDEX idx_event_staff_staff_id ON event_staff_assignments(staff_id);
CREATE INDEX idx_event_staff_assigned_by ON event_staff_assignments(assigned_by);
```

**Columns:**

| Column      | Type      | Constraints                                       | Description                  |
| ----------- | --------- | ------------------------------------------------- | ---------------------------- |
| id          | UUID      | PRIMARY KEY, DEFAULT gen_random_uuid()            | Unique assignment identifier |
| event_id    | UUID      | NOT NULL, REFERENCES events(id) ON DELETE CASCADE | Associated event             |
| staff_id    | UUID      | NOT NULL, REFERENCES users(id) ON DELETE CASCADE  | Assigned staff user          |
| assigned_at | TIMESTAMP | NOT NULL, DEFAULT NOW()                           | Assignment timestamp         |
| assigned_by | UUID      | REFERENCES users(id)                              | User who made the assignment |

**Indexes:**

- `idx_event_staff_event_id` - Find staff by event
- `idx_event_staff_staff_id` - Find events by staff (critical for access control)
- `idx_event_staff_assigned_by` - Track who made assignments

**Constraints:**

- `unique_event_staff` - One assignment per staff per event (prevents duplicates)

**Business Rules:**

- Staff can only be assigned to events they have access to
- Only event organizers and admins can assign staff
- Staff must have role='staff' in users table
- Deleting event cascades to remove staff assignments
- Deleting user cascades to remove their assignments
- assigned_by can be NULL (legacy assignments or system-assigned)

---

### deletion_audit_log

Stores comprehensive audit trail of all deletion operations for compliance and troubleshooting.

```sql
CREATE TABLE deletion_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    entity_snapshot JSONB,
    deleted_by UUID REFERENCES users(id),
    deleted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deletion_type VARCHAR(20) NOT NULL,
    deletion_reason TEXT,
    cascade_effects JSONB,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_deletion_audit_entity ON deletion_audit_log(entity_type, entity_id);
CREATE INDEX idx_deletion_audit_deleted_by ON deletion_audit_log(deleted_by);
CREATE INDEX idx_deletion_audit_deleted_at ON deletion_audit_log(deleted_at);
```

**Columns:**

| Column          | Type        | Constraints                            | Description                             |
| --------------- | ----------- | -------------------------------------- | --------------------------------------- |
| id              | UUID        | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique audit log identifier             |
| entity_type     | VARCHAR(50) | NOT NULL                               | Type: user, event, participant, checkin |
| entity_id       | UUID        | NOT NULL                               | ID of deleted entity                    |
| entity_snapshot | JSONB       | -                                      | Snapshot of entity before deletion      |
| deleted_by      | UUID        | REFERENCES users(id)                   | User who performed deletion             |
| deleted_at      | TIMESTAMP   | NOT NULL, DEFAULT NOW()                | Deletion timestamp                      |
| deletion_type   | VARCHAR(20) | NOT NULL                               | Type: hard, soft, anonymize             |
| deletion_reason | TEXT        | -                                      | Reason for deletion                     |
| cascade_effects | JSONB       | -                                      | Impact of cascading deletions           |
| ip_address      | INET        | -                                      | Client IP address                       |
| user_agent      | TEXT        | -                                      | Client user agent string                |

**Indexes:**

- `idx_deletion_audit_entity` - Find logs by entity type and ID
- `idx_deletion_audit_deleted_by` - Track deletions by user
- `idx_deletion_audit_deleted_at` - Sort/filter by deletion time

**Business Rules:**

- Append-only table (logs cannot be modified or deleted manually)
- Automatically populated on all deletion operations
- Retention period: 3 years
- Admin-only access for security and compliance
- Entity snapshot stores key data before deletion (not sensitive fields like passwords)
- Cascade effects track impact of cascading deletions (counts, not actual data)

**Example Entity Snapshot:**

```json
{
  "name": "Tech Conference 2025",
  "start_date": "2025-12-15T09:00:00Z",
  "status": "cancelled",
  "participants_count": 150,
  "organizer": {
    "id": "...",
    "name": "John Doe"
  }
}
```

**Example Cascade Effects:**

```json
{
  "participants_deleted": 150,
  "paid_participants_deleted": 87,
  "checkins_deleted": 45,
  "staff_assignments_removed": 3
}
```

---

## Data Types & Constraints

### UUID vs Integer IDs

**Why UUID:**

- **Security:** Non-sequential prevents enumeration attacks
- **Distribution:** Can generate IDs in application layer
- **Scalability:** No coordination needed for distributed systems
- **Merge-friendly:** No ID conflicts when merging databases

**Drawback:**

- Slightly larger storage (16 bytes vs 4/8 bytes)
- Random ordering (mitigated with created_at indexes)

---

### JSONB for Metadata

**Use Cases:**

- Participant custom fields (company, dietary restrictions, etc.)
- Device information (OS, browser, app version)
- Flexible schema without migrations

**Advantages:**

- Binary format (faster than JSON)
- Indexable with GIN indexes
- Query support with JSON operators

**Example Queries:**

```sql
-- Find participants from specific company
SELECT * FROM participants
WHERE metadata->>'company' = 'Tech Corp';

-- Find participants with dietary restrictions
SELECT * FROM participants
WHERE metadata ? 'dietary_restrictions';
```

---

### Timestamps

**Strategy:**

- All tables have `created_at` and `updated_at`
- Use `TIMESTAMP` (not `TIMESTAMPTZ`) with UTC convention
- Application layer handles timezone conversion

**Benefits:**

- Audit trail
- Soft delete capability (future)
- Data analytics and reporting

---

## Enumeration Types

### Event Status

| Value     | Description                     | Allowed Transitions    |
| --------- | ------------------------------- | ---------------------- |
| draft     | Being prepared                  | → published, cancelled |
| published | Active, accepting registrations | → ongoing, cancelled   |
| ongoing   | Currently happening             | → completed            |
| completed | Finished                        | (terminal)             |
| cancelled | Cancelled before completion     | (terminal)             |

---

### Participant Status

| Value     | Description              | Use Case                   |
| --------- | ------------------------ | -------------------------- |
| tentative | Awaiting confirmation    | Initial registration       |
| confirmed | Confirmed attendance     | After payment/verification |
| cancelled | Cancelled by participant | Participant cancellation   |
| declined  | Invitation declined      | Declined invitation        |

---

### Check-in Method

| Value  | Description           |
| ------ | --------------------- |
| qrcode | Scanned QR code       |
| manual | Manual entry by staff |

---

## Performance Optimization

### Index Strategy

**Primary Keys:**

- All tables use UUID primary keys
- Automatic index creation

**Foreign Keys:**

- Explicit indexes on all foreign keys
- Improves JOIN performance

**Composite Indexes:**

- `unique_event_email` on participants
- `unique_event_participant_checkin` on checkins

**GIN Indexes:**

- JSONB metadata fields
- Supports flexible queries on JSON data

---

### Query Optimization

**Common Query Patterns:**

1. **Find event with participants:**

```sql
SELECT e.*, COUNT(p.id) as participant_count
FROM events e
LEFT JOIN participants p ON e.id = p.event_id
WHERE e.id = $1
GROUP BY e.id;
```

2. **Check-in statistics:**

```sql
SELECT
    e.id,
    COUNT(DISTINCT p.id) as total_participants,
    COUNT(DISTINCT c.id) as checked_in_count
FROM events e
LEFT JOIN participants p ON e.id = p.event_id
LEFT JOIN checkins c ON e.id = c.event_id
WHERE e.id = $1
GROUP BY e.id;
```

3. **Participant search:**

```sql
SELECT * FROM participants
WHERE event_id = $1
  AND (name ILIKE $2 OR email ILIKE $2)
ORDER BY created_at DESC
LIMIT 20 OFFSET $3;
```

4. **Staff accessible events:**

```sql
SELECT e.*, COUNT(p.id) as participant_count
FROM events e
INNER JOIN event_staff_assignments esa ON e.id = esa.event_id
LEFT JOIN participants p ON e.id = p.event_id
WHERE esa.staff_id = $1
GROUP BY e.id
ORDER BY e.start_date DESC;
```

5. **Check staff assignment:**

```sql
SELECT EXISTS(
    SELECT 1 FROM event_staff_assignments
    WHERE event_id = $1 AND staff_id = $2
) AS is_assigned;
```

---

### Connection Pooling

**pgx Configuration:**

```go
config, _ := pgxpool.ParseConfig(connString)
config.MaxConns = 25              // Max concurrent connections
config.MinConns = 5               // Min idle connections
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 30 * time.Minute
```

**Benefits:**

- Reuse existing connections
- Limit database load
- Fast query execution

---

## Migration Strategy

**Migration Tool:**

- **golang-migrate** for version-controlled schema migrations
- Up/down migrations for rollback capability
- CLI and programmatic API support

**Migration Approach:**

- Sequential numbered migrations (e.g., 000001, 000002, etc.)
- Each migration includes `.up.sql` and `.down.sql` files
- Idempotent operations where possible

---

## Data Integrity

### Foreign Key Constraints

**ON DELETE CASCADE:**

- Deleting event → deletes participants, check-ins, and staff assignments
- Deleting participant → deletes their check-in

**Soft Delete (Users):**

- Deleting user → **soft delete with anonymization** (events are preserved)
- User's events remain accessible with anonymized owner information
- Staff assignments are removed (hard delete, access revoked)
- Check-ins performed by deleted user are preserved with anonymized name

**Important Note on User Deletion:**

Users are **not hard deleted** to preserve event ownership and data integrity. Instead:

1. PII is anonymized (`name`, `email`, `password_hash`)
2. `deleted_at` and `is_anonymized` flags are set
3. Events created by user remain in database
4. Staff assignments are removed immediately

**Benefits:**

- Maintains referential integrity
- Automatic cleanup of orphaned records
- Prevents data inconsistency
- Preserves event ownership while removing PII

---

### Unique Constraints

**Preventing Duplicates:**

- `users.email` - One email per user
- `participants(event_id, email)` - One registration per event
- `participants.qr_code` - Globally unique QR codes
- `checkins(event_id, participant_id)` - One check-in per participant

---

### NOT NULL Constraints

**Required Fields:**

- All IDs and foreign keys
- User credentials (email, password_hash)
- Core event data (name, start_date)
- Participant essentials (name, email, qr_code)

---

## Monitoring

### Key Metrics

**Performance:**

- Query execution time
- Connection pool utilization
- Cache hit ratio
- Index usage statistics

**Capacity:**

- Database size growth
- Table row counts
- Index sizes
- Disk space usage

**Health:**

- Replication lag (if using replicas)
- Long-running queries
- Lock contention
- Failed queries

---

## Scalability Considerations

### Read Replicas

**Use Cases:**

- Distribute read load across multiple instances
- Analytics and reporting queries
- Geographic distribution

**Configuration:**

- Primary: All writes
- Replicas: Read-only queries (event lists, statistics)
- Async replication (acceptable eventual consistency)

---

### Partitioning (Future)

**Partition Strategy:**

- Partition `checkins` by event_id or date
- Archive old events to separate partitions
- Query performance for active events maintained

**When to Implement:**

- > 10 million check-in records
- Query performance degradation
- Storage optimization needs

---

## Security

### Access Control

**Database Users:**

- `ezqrin_admin` - Full access (migrations)
- `ezqrin_app` - CRUD operations (application)
- `ezqrin_readonly` - Read-only (analytics)

**Permissions:**

```sql
-- Application user
GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA public
TO ezqrin_app;

-- Read-only user
GRANT SELECT
ON ALL TABLES IN SCHEMA public
TO ezqrin_readonly;
```

---

### Encryption

**At Rest:**

- PostgreSQL transparent data encryption (TDE)
- Encrypted backups

**In Transit:**

- TLS 1.3 for client-server connections
- Certificate-based authentication (optional)

**Application Level:**

- Sensitive fields encrypted before storage (future)
- Password hashing (bcrypt)

---

## Related Documentation

- [System Architecture](./overview.md)
- [Security Design](./security.md)
- [API Reference](../api/)
