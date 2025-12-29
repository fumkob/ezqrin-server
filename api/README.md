# ezQRin API Specification

This directory contains the OpenAPI 3.0+ specification for the ezQRin API, organized as modular YAML files for improved maintainability.

## 📁 Directory Structure

```
api/
├── openapi.yaml              # Main entry point (aggregator)
├── schemas/                  # Reusable schemas
│   ├── entities.yaml        # Core business entities (User, Event, Participant, CheckIn)
│   ├── enums.yaml           # Enumeration types (UserRole, EventStatus, etc.)
│   └── responses.yaml       # Response wrappers (StandardResponse, ErrorResponse, PaginationMeta)
├── components/               # Reusable components
│   ├── security.yaml        # Security schemes (JWT Bearer auth)
│   ├── responses.yaml       # HTTP response definitions (400, 401, 404, etc.)
│   └── parameters.yaml      # Query/path parameters (pagination, IDs, etc.)
└── paths/                    # API endpoint definitions
    ├── health.yaml          # Health check endpoints
    ├── auth.yaml            # Authentication endpoints (future)
    ├── users.yaml           # User management endpoints (future)
    ├── events.yaml          # Event management endpoints (future)
    ├── participants.yaml    # Participant management endpoints (future)
    ├── qrcode.yaml          # QR code endpoints (future)
    └── checkin.yaml         # Check-in endpoints (future)
```

## 🎯 Design Principles

### 1. **Modular Structure**
Each module is a self-contained YAML file focusing on a specific aspect of the API:
- **Schemas**: Define data models and types
- **Paths**: Define API endpoints and operations
- **Components**: Define reusable elements (responses, parameters, security)

### 2. **Single Source of Truth (SSOT)**
The OpenAPI specification is the authoritative definition of the API contract. All code is generated from this specification using `oapi-codegen`.

### 3. **Maintainability**
- One file per logical module (e.g., health endpoints, auth endpoints)
- Clear separation of concerns
- Easy to find and update specific endpoints
- Reduces merge conflicts in team environments

### 4. **Scalability**
New endpoints can be added by:
1. Creating a new file in `paths/` (e.g., `paths/webhooks.yaml`)
2. Adding a reference in `openapi.yaml`
3. Running `make gen-api` to regenerate code

## 🔧 Usage

### Generate API Code

```bash
# Generate type-safe Go code from the specification
make gen-api

# Or run the script directly
./scripts/gen-api.sh
```

Generated code will be placed in `internal/interface/api/generated/`

### Adding New Endpoints

**Example: Adding user management endpoints**

1. Create `api/paths/users.yaml`:
```yaml
/users:
  get:
    summary: List users
    operationId: listUsers
    tags:
      - users
    responses:
      '200':
        description: Success
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '../schemas/entities.yaml#/User'
```

2. Reference it in `api/openapi.yaml`:
```yaml
paths:
  /users:
    $ref: './paths/users.yaml#/~1users'
```

3. Regenerate code:
```bash
make gen-api
```

### Adding New Schemas

**Example: Adding a new entity**

1. Add to `api/schemas/entities.yaml`:
```yaml
Organization:
  type: object
  properties:
    id:
      type: string
      format: uuid
    name:
      type: string
```

2. Reference it in `api/openapi.yaml`:
```yaml
components:
  schemas:
    Organization:
      $ref: './schemas/entities.yaml#/Organization'
```

## 📖 Reference Syntax

OpenAPI $ref uses JSON Pointer syntax. For file references:

- Same file: `#/components/schemas/User`
- Different file: `./schemas/entities.yaml#/User`
- Path with slashes: `./paths/health.yaml#/~1health~1ready`
  - `~1` represents `/` in JSON Pointer
  - `/health/ready` becomes `~1health~1ready`

## 🔍 Validation

The OpenAPI specification should always be valid. Validation happens automatically during code generation, but you can also validate manually:

```bash
# Using openapi-generator-cli (if installed)
openapi-generator-cli validate -i api/openapi.yaml

# Validation is also done automatically by oapi-codegen
make gen-api
```

## 📚 Related Documentation

- **Implementation Guide**: See `docs/architecture/overview.md` for Clean Architecture patterns
- **API Documentation**: See `docs/api/README.md` for endpoint documentation
- **Code Generation**: See `config/oapi-codegen.yaml` for generation settings

## 🚀 Workflow

```
Edit openapi.yaml → make gen-api → Implement handlers → make test → make build
```

1. **Edit**: Modify `api/openapi.yaml` or module files
2. **Generate**: Run `make gen-api` to create type-safe Go code
3. **Implement**: Write handler logic in `internal/interface/api/handler/`
4. **Test**: Write tests using generated types
5. **Build**: Verify compilation with `make build`

## 📝 Best Practices

1. **Always use $ref** for reusable components
2. **Keep endpoints in separate files** by logical grouping (auth, events, etc.)
3. **Document all fields** with descriptions and examples
4. **Use consistent naming**: snake_case for JSON, PascalCase for schemas
5. **Version your API** in the URL path (`/api/v1`)
6. **Include examples** for all request/response schemas
7. **Define error responses** for all endpoints

## ⚠️ Important Notes

- **Do not edit generated code** in `internal/interface/api/generated/`
- **Always regenerate** after modifying the OpenAPI spec
- **Commit the generated code** to ensure build reproducibility
- **Test endpoints** after adding/modifying them

---

For questions or issues, refer to the [main documentation](../docs/api/README.md) or contact the development team.
