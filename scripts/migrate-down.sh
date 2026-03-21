#!/bin/bash

# migrate-down.sh - Rollback the last database migration

set -e

# Save caller-provided environment variables before loading .env
_CALLER_DB_HOST="${DB_HOST:-}"
_CALLER_DB_PORT="${DB_PORT:-}"
_CALLER_DB_USER="${DB_USER:-}"
_CALLER_DB_PASSWORD="${DB_PASSWORD:-}"
_CALLER_DB_NAME="${DB_NAME:-}"
_CALLER_DB_SSL_MODE="${DB_SSL_MODE:-}"

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    set -a
    source <(grep -v '^#' .env | sed 's/#.*$//' | grep -v '^$')
    set +a
fi

# Restore caller-provided values (take precedence over .env), then apply defaults
export DB_HOST=${_CALLER_DB_HOST:-${DB_HOST:-postgres}}
export DB_PORT=${_CALLER_DB_PORT:-${DB_PORT:-5432}}
export DB_USER=${_CALLER_DB_USER:-${DB_USER:-ezqrin}}
export DB_PASSWORD=${_CALLER_DB_PASSWORD:-${DB_PASSWORD:-ezqrin_dev}}
export DB_NAME=${_CALLER_DB_NAME:-${DB_NAME:-ezqrin_db}}
export DB_SSL_MODE=${_CALLER_DB_SSL_MODE:-${DB_SSL_MODE:-disable}}

echo "Rolling back migrations..."
echo "Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"

# Rollback one migration step
go run cmd/migrate/main.go step -1

echo "Rollback completed successfully!"
