# ezQRin Server

A Go-based backend API server for QR code generation and event check-in management.
Built with Clean Architecture principles and an OpenAPI-first development approach.

---

## Overview

ezQRin Server provides a REST API for managing events, participant registration, QR code distribution,
and attendee check-ins. It is designed for event organizers who need a reliable, scalable system
for real-time check-in operations.

**Technology Stack:**

| Component     | Technology      | Version |
| ------------- | --------------- | ------- |
| Language      | Go              | 1.25.5  |
| Web Framework | Gin             | -       |
| Database      | PostgreSQL      | 18+     |
| Cache         | Redis           | 8+      |
| Auth          | JWT             | -       |
| Logging       | Zap             | -       |
| Testing       | Ginkgo / Gomega | -       |

---

## Quick Start

### Option 1: DevContainer (Recommended)

The recommended way to develop is using [Dev Containers](https://containers.dev/), which provides
a fully configured environment with Go, PostgreSQL, Redis, Delve debugger, and all tooling
pre-installed.

**Prerequisites:**

- [Docker Desktop](https://www.docker.com/products/docker-desktop) 20.10+ (or Podman)
- [VS Code](https://code.visualstudio.com/) or [Cursor](https://cursor.sh/)
- [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

**Steps:**

```bash
git clone https://github.com/fumkob/ezqrin-server.git
cd ezqrin-server
```

Open the project in VS Code or Cursor, then select **Reopen in Container** when prompted.
The first build takes 3-5 minutes. After that, inside the container terminal:

```bash
# Apply database migrations
make migrate-up

# Start the server with hot reload
air
```

The API is available at `http://localhost:8080`.

---

### Option 2: Local Development (Without DevContainer)

**Prerequisites:**

- Go 1.25.5+
- PostgreSQL 18+
- Redis 8+
- `make`

**Steps:**

```bash
# Clone repository
git clone https://github.com/fumkob/ezqrin-server.git
cd ezqrin-server

# Set up secret environment variables
cp .env.secrets.example .env.secrets
# Edit .env.secrets with your database credentials and JWT secret

# Start PostgreSQL and Redis (or use your own instances)
make dev-up

# Apply database migrations
make migrate-up

# Run the server
go run cmd/api/main.go
```

---

## API Reference

The API is defined using **OpenAPI 3.0** as the Single Source of Truth (SSOT).

See [`api/openapi.yaml`](./api/openapi.yaml) for the complete API specification.

For a high-level overview of available endpoints and workflows, see
[`docs/api/README.md`](./docs/api/README.md).

**Health check endpoints:**

```
GET /health        # Basic health check
GET /health/ready  # Readiness probe
GET /health/live   # Liveness probe
```

---

## Development Workflow

### Make Commands

**DevContainer Management:**

```bash
make dev-up          # Start DevContainer services (PostgreSQL, Redis, API)
make dev-down        # Stop DevContainer services
make dev-shell       # Open bash shell in the dev container
make dev-logs        # View all container logs
make dev-rebuild     # Rebuild services (use after Dockerfile changes)
make dev-clean       # Stop containers and remove volumes (deletes data)
```

**Database Migrations:**

```bash
make migrate-up      # Apply all pending migrations
make migrate-down    # Rollback last migration
make migrate-version # Show current migration version
make migrate-reset   # Reset database (down all + up all)
make migrate-create NAME=migration_name  # Create new migration files
make db-shell        # Open PostgreSQL shell (psql)
make db-reset        # Drop and recreate the database
```

**Code Generation:**

```bash
make gen-api         # Generate Go code from OpenAPI specification
make gen-mock        # Generate mocks from interfaces
make gen-all         # Generate all code (API + mocks)
```

**Testing:**

```bash
make test            # Run all tests (including integration)
make test-unit       # Run only unit tests
make test-setup      # Set up test database (create + migrate)
```

**Code Quality:**

```bash
make fmt             # Auto-format Go code
make lint            # Run linters (golangci-lint)
make lint-fix        # Run linters with auto-fix
make vet             # Run go vet
```

**Build:**

```bash
make build           # Build the application binary
make run             # Run with Air (hot reload)
```

---

## Running Tests

```bash
# All tests (including integration, requires running services)
make test

# Unit tests only
make test-unit

# Set up test database first if running integration tests
make test-setup
make test
```

Tests use **Ginkgo / Gomega** for BDD-style test descriptions. See
[`docs/architecture/overview.md`](./docs/architecture/overview.md) for the testing strategy.

---

## Project Structure

```
ezqrin-server/
├── api/                        # OpenAPI specification (SSOT) and generated code
│   └── openapi.yaml            # Main OpenAPI 3.0 specification
├── cmd/
│   ├── api/                    # API server entry point
│   └── migrate/                # Migration execution tool
├── config/
│   ├── default.yaml            # Base configuration
│   ├── development.yaml        # Development environment overrides
│   └── production.yaml         # Production environment overrides
├── docs/
│   ├── api/                    # API documentation and endpoint guides
│   ├── architecture/           # Architecture and design documentation
│   └── deployment/             # Deployment and environment guides
├── internal/                   # Private application code (Clean Architecture)
│   ├── domain/                 # Entities, repository interfaces, business rules
│   ├── interface/api/handler/  # HTTP handlers
│   ├── repository/             # Repository implementations
│   └── usecase/                # Business logic
├── pkg/                        # Public packages
│   ├── crypto/                 # QR code and token generation
│   ├── errors/                 # Error definitions
│   ├── logger/                 # Logging setup
│   └── validator/              # Input validation
├── scripts/                    # Build and operational scripts
├── .devcontainer/              # DevContainer configuration
│   ├── devcontainer.json       # DevContainer settings
│   ├── Dockerfile              # Development container image
│   └── docker-compose.yaml     # Local services (PostgreSQL, Redis)
├── .env.secrets.example        # Template for secret environment variables
├── docker-compose.prod.yml     # Production Docker Compose configuration
├── Makefile                    # Build and development commands
└── go.mod                      # Go module definition
```

---

## Configuration

ezQRin uses a layered configuration system:

1. `config/default.yaml` - Base configuration (committed)
2. `config/development.yaml` or `config/production.yaml` - Environment overrides (committed)
3. Environment variables - Secrets and runtime overrides (highest priority, never committed)

Secret variables are managed separately in `.env.secrets` (local) or environment variables
(production). See [Configuration Reference](./docs/deployment/environment.md) for all available
settings.

---

## Documentation

| Document              | Location                                | Purpose                          |
| --------------------- | --------------------------------------- | -------------------------------- |
| API Specification     | `api/openapi.yaml`                      | OpenAPI 3.0 spec (SSOT)          |
| API Overview          | `docs/api/README.md`                    | Endpoint guide and workflows     |
| Architecture Overview | `docs/architecture/overview.md`         | Clean Architecture design        |
| Database Schema       | `docs/architecture/database.md`         | Entity relationships and schema  |
| Security Design       | `docs/architecture/security.md`         | Auth, authorization, data safety |
| DevContainer Guide    | `docs/deployment/docker.md`             | Development environment setup    |
| Configuration Ref     | `docs/deployment/environment.md`        | All environment variables        |
| Deployment Guide      | `DEPLOYMENT.md`                         | Production deployment            |

---

## License

This project is private and proprietary. All rights reserved.
