# DevContainer Development Guide

## Overview

This guide covers setting up the ezQRin development environment using **DevContainers** (Development
Containers). DevContainers provide a consistent, reproducible development environment that runs
inside Docker, ensuring all developers have identical tooling, dependencies, and configurations.

### Why DevContainers?

**Benefits:**

- **Consistency**: Identical environment across all developers and CI/CD
- **Zero Setup**: Clone and open in container - no local Go/PostgreSQL/Redis installation needed
- **Isolation**: Project dependencies don't conflict with other projects
- **VS Code/Cursor Integration**: Seamless IDE experience with extensions, debugging, and
  IntelliSense
- **Delve Debugging**: Pre-configured Go debugger ready to use
- **Hot Reload**: Automatic code reloading with Air during development

---

## Prerequisites

### Required Software

- **Docker Desktop 20.10+** (or Docker Engine with Docker Compose)
  - [Download Docker Desktop](https://www.docker.com/products/docker-desktop)
- **VS Code or Cursor IDE**
  - [Download VS Code](https://code.visualstudio.com/)
  - [Download Cursor](https://cursor.sh/)
- **Dev Containers Extension** (VS Code/Cursor)
  - Extension ID: `ms-vscode-remote.remote-containers`

### System Requirements

- CPU: 2+ cores
- RAM: 8GB minimum (16GB recommended)
- Disk: 10GB free space
- OS: macOS, Linux, or Windows with WSL2

---

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/ezqrin/ezqrin-server.git
cd ezqrin-server
```

### 2. Open in Container

**VS Code:**

1. Open the project folder: `File > Open Folder`
2. When prompted "Reopen in Container", click **Reopen in Container**
3. Or use Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`): `Dev Containers: Reopen in Container`

**Cursor:**

1. Open the project folder
2. Use Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`): `Dev Containers: Reopen in Container`

**First-time setup** will take 3-5 minutes to build the container and install dependencies.

### 3. Verify Setup

Once the container is running, open a terminal inside the container:

```bash
# Check Go version
go version
# Expected: go version go1.25.5 linux/amd64

# Check database connection
psql -h postgres -U ezqrin -d ezqrin -c "SELECT version();"

# Check Redis connection
redis-cli -h redis ping
# Expected: PONG
```

### 4. Run Development Server

```bash
# Start with hot reload (Air)
air

# Or run directly
go run cmd/api/main.go
```

Access the API at `http://localhost:8080/health`

---

## DevContainer Configuration

### File Structure

```
.devcontainer/
├── devcontainer.json       # Main DevContainer configuration
├── Dockerfile              # Development container image
└── docker-compose.yml      # Services (API, PostgreSQL, Redis)
```

---

### devcontainer.json

Main configuration file for DevContainer behavior:

```json
{
  "name": "ezQRin Development",
  "dockerComposeFile": "docker-compose.yml",
  "service": "api",
  "workspaceFolder": "/app",

  // VS Code/Cursor customizations
  "customizations": {
    "vscode": {
      "extensions": ["golang.go", "ms-azuretools.vscode-docker", "eamodio.gitlens"],
      "settings": {
        "go.toolsManagement.checkForUpdates": "local",
        "go.useLanguageServer": true,
        "go.lintTool": "golangci-lint",
        "go.lintOnSave": "workspace"
      }
    }
  },

  // Port forwarding (host:container)
  "forwardPorts": [
    8080, // API server
    5432, // PostgreSQL
    6379, // Redis
    2345 // Delve debugger
  ],

  // Post-creation command
  "postCreateCommand": "go mod download",

  // Keep container running
  "shutdownAction": "stopCompose",

  // Run as non-root user
  "remoteUser": "vscode"
}
```

**Key Settings:**

- `dockerComposeFile`: References `docker-compose.yml` for multi-service setup
- `service`: Specifies which service is the development container (`api`)
- `workspaceFolder`: Working directory inside container
- `forwardPorts`: Exposes ports to host machine
- `postCreateCommand`: Runs after container creation

---

### Dockerfile

Development container image with Go, Delve, and development tools:

```dockerfile
FROM golang:1.25.5-alpine

# Install development dependencies
RUN apk add --no-cache \
    git \
    make \
    curl \
    postgresql-client \
    redis \
    bash \
    build-base

# Install Delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Install Air for hot reload
RUN go install github.com/cosmtrek/air@latest

# Install additional Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create non-root user
RUN addgroup -g 1000 vscode && \
    adduser -u 1000 -G vscode -s /bin/bash -D vscode

# Set working directory
WORKDIR /app

# Change ownership
RUN chown -R vscode:vscode /app

# Switch to non-root user
USER vscode

# Default command
CMD ["bash"]
```

**Installed Tools:**

- **Delve**: Go debugger for breakpoints and step-through debugging
- **Air**: Hot reload for rapid development
- **golangci-lint**: Comprehensive Go linter
- **PostgreSQL client**: Database CLI access
- **Redis CLI**: Redis debugging and testing

---

### docker-compose.yml

Multi-service development environment:

```yaml
version: "3.8"

services:
  api:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    container_name: ezqrin-dev
    volumes:
      - ..:/app:cached
      - go-modules:/go/pkg/mod
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=ezqrin_dev
      - DB_USER=ezqrin
      - DB_PASSWORD=dev_password
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=dev_secret_key_do_not_use_in_production_min_32_chars
      - ENV=development
      - LOG_LEVEL=debug
    ports:
      - "8080:8080" # API server
      - "2345:2345" # Delve debugger
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - ezqrin-network
    command: sleep infinity

  postgres:
    image: postgres:18-alpine
    container_name: ezqrin-postgres-dev
    environment:
      - POSTGRES_DB=ezqrin_dev
      - POSTGRES_USER=ezqrin
      - POSTGRES_PASSWORD=dev_password
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ezqrin"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - ezqrin-network

  redis:
    image: redis:8-alpine
    container_name: ezqrin-redis-dev
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - ezqrin-network

volumes:
  postgres-data:
  redis-data:
  go-modules:

networks:
  ezqrin-network:
    driver: bridge
```

**Service Details:**

- **api**: Development container with source code mounted
- **postgres**: PostgreSQL 18 with persistent storage
- **redis**: Redis 8 for caching and sessions

---

## Delve Debugging Setup

### Delve Installation

Delve is **pre-installed** in the DevContainer. Verify installation:

```bash
dlv version
```

### VS Code/Cursor Launch Configuration

Create `.vscode/launch.json` for debugging:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug API",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "DB_HOST": "postgres",
        "DB_PORT": "5432",
        "DB_NAME": "ezqrin_dev",
        "DB_USER": "ezqrin",
        "DB_PASSWORD": "dev_password",
        "REDIS_HOST": "redis",
        "REDIS_PORT": "6379",
        "JWT_SECRET": "dev_secret_key_do_not_use_in_production_min_32_chars",
        "ENV": "development",
        "LOG_LEVEL": "debug"
      },
      "args": []
    },
    {
      "name": "Debug Tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}",
      "env": {
        "DB_HOST": "postgres",
        "DB_PORT": "5432",
        "DB_NAME": "ezqrin_test",
        "DB_USER": "ezqrin",
        "DB_PASSWORD": "dev_password"
      }
    },
    {
      "name": "Attach to Remote Delve",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "/app",
      "port": 2345,
      "host": "localhost"
    }
  ]
}
```

### Debugging Workflows

**1. Standard Debugging (Recommended):**

1. Set breakpoints in your code
2. Press `F5` or select **Debug API** from Run menu
3. Application starts with debugger attached
4. Execution pauses at breakpoints

**2. Remote Debugging (Advanced):**

Start Delve server manually:

```bash
dlv debug cmd/api/main.go --headless --listen=:2345 --api-version=2
```

Then attach using **Attach to Remote Delve** configuration.

**3. Debug Tests:**

1. Select **Debug Tests** configuration
2. Run specific test files with breakpoints
3. Step through test execution

### Debugging Tips

**Breakpoint Types:**

- **Line Breakpoint**: Click left margin or press `F9`
- **Conditional Breakpoint**: Right-click breakpoint → Edit Breakpoint
- **Logpoint**: Right-click → Add Logpoint (non-breaking logging)

**Debugging Commands:**

- `F5`: Continue
- `F10`: Step Over
- `F11`: Step Into
- `Shift+F11`: Step Out
- `Shift+F5`: Stop Debugging

**View Variables:**

- Hover over variable to see value
- Use **Variables** panel in sidebar
- Add expressions to **Watch** panel

---

## Development Workflow

### Starting Development

```bash
# Inside DevContainer terminal

# 1. Install dependencies
go mod download

# 2. Run migrations
make migrate-up
# or
./scripts/migrate-up.sh

# 3. Start with hot reload
air
```

### Hot Reload Configuration (.air.toml)

Create `.air.toml` in project root:

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

**Air automatically:**

- Watches for file changes
- Rebuilds the application
- Restarts the server
- Preserves logs and output

---

## Database Management

### Access PostgreSQL

```bash
# Connect via psql
psql -h postgres -U ezqrin -d ezqrin_dev

# Run SQL file
psql -h postgres -U ezqrin -d ezqrin_dev < schema.sql

# Dump database
pg_dump -h postgres -U ezqrin ezqrin_dev > backup.sql
```

### Run Migrations

```bash
# Up migrations
./scripts/migrate-up.sh

# Down migrations
./scripts/migrate-down.sh

# Create new migration
migrate create -ext sql -dir internal/infrastructure/database/migrations -seq migration_name
```

### Database GUI Access

Access PostgreSQL from host machine:

- **Host**: `localhost`
- **Port**: `5432`
- **Database**: `ezqrin_dev`
- **User**: `ezqrin`
- **Password**: `dev_password`

**Recommended Tools:**

- [TablePlus](https://tableplus.com/)
- [DBeaver](https://dbeaver.io/)
- [pgAdmin](https://www.pgadmin.org/)

---

## Redis Management

### Access Redis CLI

```bash
# Connect to Redis
redis-cli -h redis

# Common commands
PING                    # Test connection
KEYS *                  # List all keys
GET key_name            # Get value
SET key_name value      # Set value
FLUSHALL                # Clear all data
```

### Monitor Redis

```bash
# Monitor all commands
redis-cli -h redis MONITOR

# Get server info
redis-cli -h redis INFO
```

---

## Testing

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/domain/entity

# Verbose output
go test -v ./...

# With race detection
go test -race ./...
```

### Ginkgo Tests (BDD)

```bash
# Install Ginkgo CLI
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Run Ginkgo tests
ginkgo -r

# With coverage
ginkgo -r --cover

# Watch mode (auto-run on changes)
ginkgo watch -r
```

---

## Common Tasks

### Environment Variables

Environment variables are set in `docker-compose.yml`. To modify:

1. Edit `.devcontainer/docker-compose.yml`
2. Rebuild container: `Dev Containers: Rebuild Container`

For production-like settings, create `.env` file (see [Environment Variables](./environment.md)).

### Install Go Packages

```bash
# Install package
go get github.com/some/package

# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Linting

```bash
# Run golangci-lint
golangci-lint run

# Auto-fix issues
golangci-lint run --fix

# Specific linters
golangci-lint run --enable-all
```

### Build Binary

```bash
# Development build
go build -o bin/ezqrin cmd/api/main.go

# Production build (optimized)
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/ezqrin cmd/api/main.go
```

---

## Troubleshooting

### Container Won't Start

**Check Docker:**

```bash
# Verify Docker is running
docker ps

# Check Docker Compose services
docker-compose -f .devcontainer/docker-compose.yml ps
```

**Rebuild Container:**

1. Command Palette → `Dev Containers: Rebuild Container`
2. Or: `Dev Containers: Rebuild Without Cache`

### Database Connection Failed

**Verify PostgreSQL is running:**

```bash
docker ps | grep postgres
```

**Test connection:**

```bash
psql -h postgres -U ezqrin -d ezqrin_dev
```

**Check environment variables:**

```bash
env | grep DB_
```

### Port Already in Use

**Find process using port:**

```bash
# macOS/Linux
lsof -i :8080

# Kill process
kill -9 <PID>
```

**Or change port** in `docker-compose.yml`:

```yaml
ports:
  - "8081:8080" # Use 8081 on host
```

### Hot Reload Not Working

**Check Air is running:**

```bash
ps aux | grep air
```

**Restart Air:**

```bash
# Stop current process (Ctrl+C)
# Restart
air
```

**Verify `.air.toml` exists** in project root.

### Debugger Won't Attach

**Check Delve port forwarding:**

```bash
# Verify port 2345 is forwarded
curl localhost:2345
```

**Restart with Delve:**

```bash
dlv debug cmd/api/main.go --headless --listen=:2345 --api-version=2
```

**Check launch.json configuration** matches port and paths.

### Go Modules Issues

**Clear module cache:**

```bash
go clean -modcache
go mod download
```

**Verify go.mod:**

```bash
go mod tidy
go mod verify
```

---

## Performance Optimization

### Volume Mounting Performance

**macOS/Windows users** may experience slow file I/O. Optimize with:

```yaml
volumes:
  - ..:/app:cached # Cached mode for better performance
```

**Alternatives:**

- Use named volumes for dependencies: `go-modules:/go/pkg/mod`
- Exclude heavy directories: `tmp/`, `node_modules/`

### Build Cache

Leverage Docker build cache:

```bash
# Build with cache
docker-compose -f .devcontainer/docker-compose.yml build

# Clear cache and rebuild
docker-compose -f .devcontainer/docker-compose.yml build --no-cache
```

---

## CI/CD Integration

DevContainer configuration ensures CI/CD environments match local development:

### GitHub Actions Example

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: golang:1.25.5-alpine
    services:
      postgres:
        image: postgres:18-alpine
        env:
          POSTGRES_DB: ezqrin_test
          POSTGRES_USER: ezqrin
          POSTGRES_PASSWORD: test_password
      redis:
        image: redis:8-alpine
    steps:
      - uses: actions/checkout@v3
      - name: Run tests
        run: go test ./...
```

---

## Cleanup

### Remove Containers and Volumes

```bash
# Stop and remove containers
docker-compose -f .devcontainer/docker-compose.yml down

# Remove volumes (WARNING: deletes data)
docker-compose -f .devcontainer/docker-compose.yml down -v

# Remove images
docker-compose -f .devcontainer/docker-compose.yml down --rmi all
```

### Prune Docker Resources

```bash
# Remove unused containers
docker container prune

# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Full cleanup (WARNING: removes all unused resources)
docker system prune -a --volumes
```

---

## Security Best Practices

### Development Security

1. **Never commit `.env` files** with secrets
2. **Use different credentials** for development and production
3. **Keep DevContainer images updated** regularly
4. **Limit exposed ports** to necessary services only

### Production Deployment

DevContainers are **for development only**. For production:

- Build optimized Docker images
- Use secrets management (Docker Secrets, Vault)
- Enable TLS/SSL
- Configure firewalls and network policies

See production deployment guides for details.

---

## Related Documentation

- [Environment Variables](./environment.md)
- [System Architecture](../architecture/overview.md)
- [Security Design](../architecture/security.md)
- [Testing Guide](../testing.md)

---

## Additional Resources

### DevContainers

- [VS Code DevContainers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [DevContainer Specification](https://containers.dev/)

### Go Development

- [Go Documentation](https://go.dev/doc/)
- [Delve Debugger](https://github.com/go-delve/delve)
- [Air Hot Reload](https://github.com/cosmtrek/air)

### Docker

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
