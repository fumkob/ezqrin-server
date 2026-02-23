.PHONY: help dev-up dev-rebuild dev-down dev-shell dev-logs dev-clean migrate-up migrate-down migrate-version migrate-create db-shell db-reset test build gen-api gen-mock gen-all lint-fix test-coverage-local test-unit-coverage-local

#
# Container Runtime Detection (Docker/Podman)
#

# Auto-detect container runtime (prefer podman if available)
CONTAINER_RUNTIME := $(shell command -v podman 2> /dev/null)
ifdef CONTAINER_RUNTIME
	COMPOSE_CMD := podman-compose
	RUNTIME_NAME := Podman
else
	CONTAINER_RUNTIME := $(shell command -v docker 2> /dev/null)
	COMPOSE_CMD := docker compose
	RUNTIME_NAME := Docker
endif

# Verify container runtime is available
check-runtime:
ifndef CONTAINER_RUNTIME
	@echo "ERROR: Neither Docker nor Podman is installed."
	@echo "Please install one of them to continue."
	@exit 1
endif

#
# Help
#

help:
	@echo "ezQRin Server - Makefile Commands"
	@echo "Container Runtime: $(RUNTIME_NAME)"
	@echo ""
	@echo "DevContainer Management (CLI):"
	@echo "  make dev-up          - Start DevContainer services (PostgreSQL, Redis, API)"
	@echo "  make dev-rebuild     - Rebuild and restart DevContainer services (use after Dockerfile changes)"
	@echo "  make dev-down        - Stop DevContainer services"
	@echo "  make dev-shell       - Open bash shell in dev container"
	@echo "  make dev-logs        - View all container logs"
	@echo "  make dev-clean       - Stop containers and remove volumes (WARNING: deletes data)"
	@echo "  make dev-ps          - Show running containers"
	@echo ""
	@echo "Database Migrations:"
	@echo "  make migrate-up      - Apply all pending migrations"
	@echo "  make migrate-down    - Rollback last migration"
	@echo "  make migrate-version - Show current migration version"
	@echo "  make migrate-reset   - Reset database (down all + up all)"
	@echo "  make migrate-test-up - Apply migrations to test database"
	@echo "  make db-shell        - Open PostgreSQL shell (psql)"
	@echo "  make db-reset        - Drop and recreate database"
	@echo "  make db-create-test  - Create test database"
	@echo ""
	@echo "Code Generation:"
	@echo "  make gen-api         - Generate API code from OpenAPI specification"
	@echo "  make gen-mock        - Generate test mocks (coming in Task 1.3)"
	@echo "  make gen-all         - Generate all code (API + mocks)"
	@echo ""
	@echo "Development:"
	@echo "  make test            - Run all tests (including integration)"
	@echo "  make test-unit       - Run only unit tests (fast)"
	@echo "  make test-setup      - Setup test database (create + migrate)"
	@echo "  make build           - Build the application"
	@echo "  make run             - Run the application with Air (hot reload)"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt             - Format Go code"
	@echo "  make lint            - Run linters"
	@echo "  make lint-fix        - Run linters with auto-fix"
	@echo "  make vet             - Run go vet"
	@echo ""
	@echo "Verification:"
	@echo "  make verify-setup    - Verify complete setup (migrations, DB connection)"

#
# DevContainer Management
#

# Start DevContainer services
dev-up: check-runtime
	@echo "Starting DevContainer services with $(RUNTIME_NAME)..."
	cd .devcontainer && $(COMPOSE_CMD) up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo "Services started successfully!"
	@echo ""
	@echo "PostgreSQL: localhost:5432"
	@echo "  User:     ezqrin"
	@echo "  Password: ezqrin_dev"
	@echo "  Database: ezqrin_db"
	@echo "Redis:      localhost:6379"
	@echo "API:        localhost:8080"
	@echo "Delve:      localhost:2345"
	@echo ""
	@echo "Next steps:"
	@echo "  - Enter dev container:  make dev-shell"
	@echo "  - Apply migrations:     make migrate-up"
	@echo "  - Verify setup:         make verify-setup"

# Rebuild and restart DevContainer services
dev-rebuild: check-runtime
	@echo "Rebuilding and restarting DevContainer services with $(RUNTIME_NAME)..."
	cd .devcontainer && $(COMPOSE_CMD) up -d --build
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo "Services rebuilt and started successfully!"

# Stop DevContainer services
dev-down: check-runtime
	@echo "Stopping DevContainer services..."
	cd .devcontainer && $(COMPOSE_CMD) down
	@echo "Services stopped."

# Show running containers
dev-ps: check-runtime
	@echo "Running containers:"
	cd .devcontainer && $(COMPOSE_CMD) ps

# Open bash shell in dev container
dev-shell: check-runtime
	@echo "Opening shell in dev container..."
	cd .devcontainer && $(COMPOSE_CMD) exec api /bin/bash

# View container logs
dev-logs: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) logs -f

# Clean up containers and volumes (WARNING: deletes data)
dev-clean: check-runtime
	@echo "WARNING: This will delete all container data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd .devcontainer && $(COMPOSE_CMD) down -v; \
		echo "Containers and volumes removed."; \
	else \
		echo "Cancelled."; \
	fi

#
# Database Migrations
#

# Apply all pending migrations
migrate-up: check-runtime
	@echo "Applying migrations..."
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && ./scripts/migrate-up.sh"

# Apply migrations to test database
migrate-test-up: check-runtime
	@echo "Applying migrations to test database..."
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && DB_NAME=ezqrin_test ./scripts/migrate-up.sh"

# Rollback last migration
migrate-down: check-runtime
	@echo "Rolling back last migration..."
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && ./scripts/migrate-down.sh"

# Show current migration version
migrate-version: check-runtime
	@echo "Current migration version:"
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go run cmd/migrate/main.go version"

# Reset migrations (down all + up all)
migrate-reset: check-runtime
	@echo "Resetting all migrations..."
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go run cmd/migrate/main.go down && go run cmd/migrate/main.go up"
	@echo "Migrations reset complete."

# Open PostgreSQL shell
db-shell: check-runtime
	@echo "Opening PostgreSQL shell..."
	cd .devcontainer && $(COMPOSE_CMD) exec postgres psql -U ezqrin -d ezqrin_db

# Drop and recreate database
db-reset: check-runtime
	@echo "WARNING: This will delete all database data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd .devcontainer && $(COMPOSE_CMD) exec -T postgres psql -U ezqrin -d postgres -c "DROP DATABASE IF EXISTS ezqrin_db;"; \
		cd .devcontainer && $(COMPOSE_CMD) exec -T postgres psql -U ezqrin -d postgres -c "CREATE DATABASE ezqrin_db;"; \
		echo "Database reset complete. Run 'make migrate-up' to apply migrations."; \
	else \
		echo "Cancelled."; \
	fi

# Create test database
db-create-test: check-runtime
	@echo "Creating test database 'ezqrin_test'..."
	cd .devcontainer && $(COMPOSE_CMD) exec -T postgres psql -U ezqrin -d postgres -c "CREATE DATABASE ezqrin_test;" || echo "Database 'ezqrin_test' already exists."

#
# Development
#

# Run all tests (including integration)
test: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go test -p 1 -count=1 -tags=integration ./..."

# Run only unit tests
test-unit: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go test -count=1 ./..."

# Run tests with coverage report (local, no container)
test-coverage-local:
	@./scripts/run-tests.sh

# Run unit tests with coverage (local)
test-unit-coverage-local:
	@./scripts/run-tests.sh --unit-only

# Setup test database (create + migrate)
test-setup: db-create-test migrate-test-up

# Build the application
build: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go build -o bin/ezqrin-server cmd/api/main.go"

# Run with Air (hot reload)
run: check-runtime
	@echo "Starting application with Air (hot reload)..."
	cd .devcontainer && $(COMPOSE_CMD) exec api bash -c "cd /workspace && air"

#
# Code Quality
#

# Format Go code
fmt: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && gofmt -s -w ."

# Run linters
lint: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && golangci-lint run ./..."

# Run linters with auto-fix
lint-fix: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && golangci-lint run --fix ./..."

# Run go vet
vet: check-runtime
	cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go vet ./..."

#
# Quick verification workflow
#

# Verify complete setup (for Task 0.2)
verify-setup: check-runtime
	@echo "=== Verifying ezQRin Server Setup ==="
	@echo "Container Runtime: $(RUNTIME_NAME)"
	@echo ""
	@echo "1. Checking DevContainer services..."
	@cd .devcontainer && $(COMPOSE_CMD) ps
	@echo ""
	@echo "2. Checking PostgreSQL connection..."
	@cd .devcontainer && $(COMPOSE_CMD) exec -T postgres pg_isready -U ezqrin -d ezqrin_db
	@echo ""
	@echo "3. Checking migration status..."
	@cd .devcontainer && $(COMPOSE_CMD) exec -T api bash -c "cd /workspace && go run cmd/migrate/main.go version || echo 'No migrations applied yet'"
	@echo ""
	@echo "4. Listing database tables..."
	@cd .devcontainer && $(COMPOSE_CMD) exec -T postgres psql -U ezqrin -d ezqrin_db -c "\\dt" || echo "No tables yet - run 'make migrate-up'"
	@echo ""
	@echo "=== Verification Complete ==="
	@echo ""
	@echo "If no tables are shown, run:"
	@echo "  make migrate-up"

#
# Code Generation
#
# Required tools are pre-installed in DevContainer (.devcontainer/Dockerfile)
# Run 'make dev-rebuild' after Dockerfile changes

# Generate API code from OpenAPI specification
gen-api:
	@echo "Generating API code from OpenAPI specification..."
	@bash scripts/gen-api.sh

# Generate test mocks (placeholder for Task 1.3)
gen-mock:
	@echo "Mock generation will be implemented in Task 1.3"
	@echo "This will use go.uber.org/mock to generate mocks for interfaces"

# Generate all code (API + mocks)
gen-all: gen-api gen-mock
	@echo ""
	@echo "=== All Code Generation Complete ==="
