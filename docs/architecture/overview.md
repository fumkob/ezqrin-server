# System Architecture Overview

## Introduction

ezQRin is built using clean architecture principles with a focus on maintainability, testability,
and scalability. This document provides a high-level overview of the system architecture, design
patterns, and technology stack.

---

## Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose                  |
| --------- | ---------- | ------- | ------------------------ |
| Language  | Go         | 1.25.4+ | Backend implementation   |
| Database  | PostgreSQL | 18+     | Primary data store       |
| Cache     | Redis      | 8+      | Session and data caching |
| API       | REST       | -       | Primary API interface    |

### Key Libraries

**Web Framework:**

- `gin-gonic/gin` - High-performance HTTP router and middleware framework

**Database:**

- `jackc/pgx/v5` - PostgreSQL driver with connection pooling
- `golang-migrate/migrate` - Database migration management
- `sqlc` - Type-safe SQL query generation

**Authentication:**

- `golang-jwt/jwt` - JWT token generation and validation
- `bcrypt` - Secure password hashing

**QR Code:**

- `skip2/go-qrcode` - QR code generation and encoding

**Validation:**

- `go-playground/validator/v10` - Struct validation and input sanitization

**API Code Generation:**

- `oapi-codegen` - OpenAPI-based code generation for handlers, DTOs, and clients

**Logging:**

- `uber-go/zap` - Structured logging for performance and observability

**Testing:**

- `stretchr/testify` - Test assertions and mocking
- `httptest` - HTTP handler testing

---

## Architectural Patterns

### Clean Architecture (Layered Design)

The system follows clean architecture principles with clear separation of concerns across four
layers:

```
┌─────────────────────────────────────────┐
│   Presentation Layer (API)              │
│   - REST/GraphQL Handlers               │
│   - Request/Response DTOs               │
│   - Middleware (Auth, Logging, CORS)    │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│   Application Layer (Use Cases)         │
│   - Business Logic                      │
│   - Service Orchestration               │
│   - Transaction Management              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│   Domain Layer                          │
│   - Entities (Event, Participant, etc.) │
│   - Business Rules                      │
│   - Repository Interfaces               │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│   Infrastructure Layer                  │
│   - Repository Implementations          │
│   - External Services (Email, Storage)  │
│   - Database Access                     │
└─────────────────────────────────────────┘
```

### Layer Responsibilities

**1. Presentation Layer**

- HTTP request/response handling
- Input validation and sanitization
- Authentication and authorization
- Response formatting
- Error handling and mapping

**2. Application Layer**

- Business logic orchestration
- Transaction coordination
- Cross-cutting concerns
- Use case implementation
- Service composition

**3. Domain Layer**

- Core business entities
- Domain-specific rules
- Repository contracts
- Domain services
- Value objects

**4. Infrastructure Layer**

- Database operations
- External API integrations
- File system operations
- Email delivery
- Caching implementation

---

## Dependency Flow

### Dependency Inversion Principle

```
External Layer → Application Layer → Domain Layer
                                    ↑
Infrastructure ──────────────────────┘
(implements domain interfaces)
```

**Key Principles:**

- **Outer layers depend on inner layers** (never the reverse)
- **Domain layer has no external dependencies** (pure business logic)
- **Interfaces defined in domain layer** (implemented in infrastructure)
- **Dependency injection** used throughout

---

## Directory Structure

```
ezqrin-server/
├── cmd/                        # Application entry points
│   ├── api/                    # API server
│   └── cli/                    # CLI tools
│
├── internal/                   # Private application code
│   ├── domain/                 # Domain layer
│   │   ├── entity/             # Business entities
│   │   ├── repository/         # Repository interfaces
│   │   └── service/            # Domain services
│   │
│   ├── usecase/                # Application layer
│   │   ├── event/              # Event use cases
│   │   ├── participant/        # Participant use cases
│   │   ├── checkin/            # Check-in use cases
│   │   ├── staff/              # Staff assignment use cases
│   │   └── stats/              # Statistics use cases
│   │
│   ├── interface/              # Presentation layer
│   │   ├── api/                # REST API
│   │   │   ├── generated/      # OpenAPI generated code (DO NOT EDIT)
│   │   │   │   ├── models.gen.go    # Request/Response DTOs
│   │   │   │   ├── server.gen.go    # Gin server interfaces
│   │   │   │   └── client.gen.go    # API client
│   │   │   ├── handler/        # HTTP handlers (implement generated interfaces)
│   │   │   ├── middleware/     # HTTP middleware
│   │   │   ├── request/        # Custom request DTOs (if needed)
│   │   │   └── response/       # Custom response DTOs (if needed)
│   │   └── graphql/            # GraphQL (future)
│   │
│   └── infrastructure/         # Infrastructure layer
│       ├── database/           # Database implementations
│       ├── cache/              # Cache implementations
│       ├── qrcode/             # QR code generator
│       ├── wallet/             # Wallet integration
│       ├── email/              # Email service
│       └── storage/            # File storage
│
├── pkg/                        # Public libraries
│   ├── crypto/                 # Cryptography utilities
│   ├── validator/              # Custom validators
│   ├── logger/                 # Logging utilities
│   └── errors/                 # Error definitions
│
├── config/                     # Configuration files
│   ├── oapi-codegen.yaml       # OpenAPI code generation config
│   └── ...                     # Other configs
│
├── scripts/                    # Build and deployment scripts
│
├── docs/                       # Documentation
│   ├── api/                    # API documentation
│   │   ├── openapi.yaml        # OpenAPI 3.0+ specification (SSOT)
│   │   ├── README.md           # API overview
│   │   └── ...                 # Endpoint documentation
│   ├── architecture/           # Architecture docs
│   └── deployment/             # Deployment guides
│
└── deployments/                # Deployment configurations
```

---

## Data Flow

### Request Lifecycle

```
1. Client Request
        ↓
2. Middleware Chain (Auth, Logging, CORS, Rate Limit)
        ↓
3. Router → Handler (Presentation Layer)
        ↓
4. Request Validation & DTO Mapping
        ↓
5. Use Case Execution (Application Layer)
        ↓
6. Domain Logic & Repository Calls (Domain Layer)
        ↓
7. Database Operations (Infrastructure Layer)
        ↓
8. Response DTO Mapping
        ↓
9. JSON Response to Client
```

### Example: Create Participant Flow

```
POST /api/v1/events/:id/participants
        ↓
[Middleware: Auth, Logging]
        ↓
[Handler: participant.CreateHandler]
        ↓
[Validation: Request DTO]
        ↓
[Use Case: participant.CreateParticipant]
        ↓
[Domain: Participant Entity + QR Code Generation]
        ↓
[Repository: Save to PostgreSQL]
        ↓
[Response: Participant DTO with QR Code]
        ↓
201 Created Response
```

---

## Design Patterns

### Repository Pattern

**Purpose:** Abstract data access logic from business logic

```go
// Domain layer - interface
type ParticipantRepository interface {
    Create(ctx context.Context, participant *Participant) error
    FindByID(ctx context.Context, id uuid.UUID) (*Participant, error)
    FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*Participant, error)
    Update(ctx context.Context, participant *Participant) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// Infrastructure layer - implementation
type PostgresParticipantRepository struct {
    db *pgx.Pool
}
```

**Benefits:**

- Testable business logic (mock repositories)
- Database-agnostic domain layer
- Centralized data access logic

---

### Service Layer Pattern

**Purpose:** Orchestrate complex business operations

```go
type ParticipantService struct {
    participantRepo repository.ParticipantRepository
    qrCodeService   service.QRCodeService
    emailService    service.EmailService
}

func (s *ParticipantService) RegisterAndNotify(
    ctx context.Context,
    participant *Participant,
) error {
    // 1. Generate QR code
    qrCode, err := s.qrCodeService.Generate(participant)

    // 2. Save participant
    err = s.participantRepo.Create(ctx, participant)

    // 3. Send confirmation email
    err = s.emailService.SendQRCode(participant, qrCode)

    return nil
}
```

**Benefits:**

- Clear business logic organization
- Transaction management
- Reusable service components

---

### Middleware Pattern

**Purpose:** Cross-cutting concerns in HTTP pipeline

```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c.Request)
        claims, err := validateToken(token, jwtSecret)
        if err != nil {
            c.AbortWithStatus(401)
            return
        }
        c.Set("user", claims)
        c.Next()
    }
}
```

**Built-in Middleware:**

- Authentication (JWT validation)
- Logging (request/response)
- CORS (cross-origin requests)
- Rate limiting (abuse prevention)
- Error recovery (panic handling)

---

## Data Storage

### PostgreSQL Schema

**Core Tables:**

- `users` - System users (organizers, staff, admins)
- `events` - Event information
- `participants` - Event participants
- `checkins` - Check-in records
- `event_staff_assignments` - Staff-to-event assignments for access control

**Key Features:**

- UUID primary keys (security and distribution)
- Foreign key constraints with cascading deletes (referential integrity)
- Indexed columns for performance (foreign keys, search fields)
- Timestamps on all tables (audit trail)
- JSON columns for flexible metadata (participants.metadata)

See [Database Design](./database.md) for detailed schema.

---

### Redis Caching

**Cache Strategy:**

- **Event data:** 5 minutes TTL
- **Participant lists:** 1 minute TTL
- **Statistics:** 30 seconds TTL
- **Session data:** Token expiration time

**Cache Invalidation:**

- Write-through on updates
- Cache-aside pattern
- TTL-based expiration

---

## API Design

### RESTful Principles

**Resource-Based URLs:**

```
/api/v1/events                     # Event collection
/api/v1/events/:id                 # Event resource
/api/v1/events/:id/participants    # Nested participants
/api/v1/events/:id/staff           # Staff assignments
/api/v1/events/:id/staff/:staff_id # Specific staff assignment
```

**HTTP Methods:**

- `GET` - Retrieve resources
- `POST` - Create resources
- `PUT` - Update resources (full)
- `PATCH` - Update resources (partial)
- `DELETE` - Delete resources

**Status Codes:**

- `2xx` - Success
- `4xx` - Client errors
- `5xx` - Server errors

See [API Documentation](../api/) for complete reference.

---

### API-First Development with OpenAPI

**Single Source of Truth:**

ezQRin adopts an **API-first development approach** where the OpenAPI specification serves as the
**Single Source of Truth (SSOT)** for all API contracts. This ensures consistency between
documentation, server implementation, and client expectations.

**Code Generation with oapi-codegen:**

We use [`oapi-codegen`](https://github.com/deepmap/oapi-codegen) to automatically generate:

- **Request/Response DTOs** - Type-safe data structures
- **HTTP Handlers (Gin)** - Router interfaces with type safety
- **API Client** - Client libraries for testing and integration
- **Validation Logic** - Input validation based on OpenAPI schemas

**Development Workflow:**

```
1. Design API in OpenAPI Specification (YAML/JSON)
        ↓
2. Generate Go Code via oapi-codegen
        ↓
3. Implement Business Logic in Generated Handlers
        ↓
4. Run Tests with Generated Client
        ↓
5. Deploy with Auto-Generated Documentation
```

**Benefits:**

- **Design-First:** API contracts defined before implementation
- **Type Safety:** Compile-time guarantees for request/response structures
- **Consistency:** Single source ensures docs and code never drift
- **Rapid Development:** Reduced boilerplate code generation
- **Client SDKs:** Automatic client library generation for frontend/mobile
- **Validation:** Schema-driven input validation without manual code

**OpenAPI Specification Location:**

```
docs/api/openapi.yaml          # Main OpenAPI 3.0+ specification
```

**Code Generation Configuration:**

```yaml
# config/oapi-codegen.yaml
package: api
generate:
  models: true              # Generate request/response models
  gin-server: true          # Generate Gin server interfaces
  client: true              # Generate API client
  strict-server: true       # Strict mode for type safety
output: internal/interface/api/generated
```

**Generated Code Structure:**

```
internal/interface/api/
├── generated/              # Auto-generated (DO NOT EDIT manually)
│   ├── models.gen.go       # Request/Response DTOs
│   ├── server.gen.go       # Gin server interfaces
│   └── client.gen.go       # API client
│
└── handler/                # Business logic implementation
    ├── event.go            # Implements generated EventHandler interface
    ├── participant.go      # Implements generated ParticipantHandler interface
    └── checkin.go          # Implements generated CheckinHandler interface
```

**Example: Generated Interface Implementation**

```go
// Generated by oapi-codegen
type ServerInterface interface {
    CreateEvent(c *gin.Context)
    GetEvent(c *gin.Context, eventId string)
    UpdateEvent(c *gin.Context, eventId string)
}

// Manual implementation in handler/event.go
type EventHandler struct {
    eventService usecase.EventService
}

// Implements generated ServerInterface
func (h *EventHandler) CreateEvent(c *gin.Context) {
    var req CreateEventRequest // Generated DTO
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Message: err.Error()})
        return
    }
    // Business logic implementation
    event, err := h.eventService.Create(c.Request.Context(), &req)
    // ...
}
```

**Maintenance Workflow:**

1. Update `docs/api/openapi.yaml` when API changes
2. Run `make generate` or `npm run gen:api` to regenerate code
3. Implement new interfaces or update existing implementations
4. Run tests to verify compatibility
5. Commit both spec and generated code together

**Version Control:**

- ✅ **Commit OpenAPI spec** (`docs/api/openapi.yaml`)
- ✅ **Commit generated code** (`internal/interface/api/generated/*`)
- ℹ️ Generated code is tracked to ensure consistency across environments

---

## Security Architecture

### Authentication Flow

```
1. User Login (POST /auth/login)
        ↓
2. Validate Credentials
        ↓
3. Generate JWT Tokens
   - Access Token (15 min)
   - Refresh Token (7 days for web, 90 days for mobile)
        ↓
4. Return Tokens to Client
        ↓
5. Client Stores Tokens
        ↓
6. Subsequent Requests Include Access Token
        ↓
7. Token Validation on Each Request
```

### Authorization Model

**Role-Based Access Control (RBAC):**

- `admin` - Full system access to all resources
- `organizer` - Create and manage own events, assign staff
- `staff` - Access only assigned events, perform check-ins

**Resource-Level Authorization:**

- **Events:** Admin sees all, Organizer sees own, Staff sees assigned
- **Participants:** Access controlled through event permissions
- **Check-ins:** Organizer and assigned Staff can perform
- **Staff Management:** Organizer and Admin can assign/remove staff

See [Security Design](./security.md) for detailed security measures.

---

## Scalability Considerations

### Horizontal Scaling

**Stateless Design:**

- No server-side session state
- JWT tokens for authentication
- Shared Redis for distributed caching
- Load balancer compatible

**Database Scaling:**

- Read replicas for query distribution
- Connection pooling (pgx)
- Prepared statements (query caching)
- Efficient indexing strategy

---

### Performance Optimization

**Caching Strategy:**

- Redis for frequently accessed data
- Query result caching
- Static asset CDN (future)

**Database Optimization:**

- Indexed foreign keys
- Compound indexes for common queries
- Batch operations for bulk imports
- Connection pool tuning

**Concurrent Processing:**

- Goroutines for parallel tasks
- Worker pools for background jobs
- Async email sending
- Bulk QR code generation

---

## Monitoring & Observability

### Application Metrics

**Performance Metrics:**

- Request count and latency
- Database query performance
- Cache hit/miss rates
- Error rates and types

**Business Metrics:**

- Event creation rate
- Participant registrations
- Check-in success rate
- QR code generation stats

### Health Checks

```
GET /health          # Basic health check
GET /health/live     # Kubernetes liveness probe
GET /health/ready    # Kubernetes readiness probe
```

**Checks Include:**

- Database connectivity
- Redis connectivity
- Disk space availability
- Memory usage

---

## Testing Strategy

### Test Pyramid

```
        ┌───────────┐
        │    E2E    │ (10%)
        └───────────┘
      ┌───────────────┐
      │  Integration  │ (30%)
      └───────────────┘
    ┌───────────────────┐
    │   Unit Tests      │ (60%)
    └───────────────────┘
```

**Unit Tests:**

- Domain logic testing
- Use case testing
- Utility function testing
- Target: 80% code coverage

**Integration Tests:**

- Repository testing with test database
- API endpoint testing
- Middleware testing
- External service integration testing

**E2E Tests:**

- Complete user workflows
- Critical business paths
- Cross-component interaction

---

## Deployment Architecture

### Container-Based Deployment

```
┌─────────────────────────────────────────┐
│         Load Balancer / Ingress         │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│      API Server Instances (3+)          │
│  (Horizontally Scaled, Stateless)       │
└─────────────────────────────────────────┘
          ↓                      ↓
┌──────────────────┐   ┌──────────────────┐
│   PostgreSQL     │   │      Redis       │
│   (Primary +     │   │   (Cluster)      │
│    Replicas)     │   │                  │
└──────────────────┘   └──────────────────┘
```

**Components:**

- **API Servers:** Multiple instances behind load balancer
- **PostgreSQL:** Primary with read replicas
- **Redis:** Cluster for high availability
- **Storage:** Persistent volumes for QR codes and logs

See [Deployment Guide](../deployment/) for setup instructions.

---

## Future Enhancements

### Phase 2 Features

**GraphQL API:**

- Flexible query capabilities
- Real-time subscriptions
- Reduced over-fetching

**Microservices:**

- Event service
- Check-in service
- Notification service
- Analytics service

**Advanced Features:**

- Real-time check-in dashboard
- Analytics and reporting
- Mobile SDK
- Multi-tenant support

---

## Related Documentation

- [Database Design](./database.md)
- [Security Design](./security.md)
- [API Reference](../api/)
- [Deployment Guide](../deployment/)
