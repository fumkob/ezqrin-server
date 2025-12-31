#!/bin/bash

# migrate-down.sh - Rollback the last database migration

set -e

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    set -a
    source <(grep -v '^#' .env | sed 's/#.*$//' | grep -v '^$')
    set +a
fi

# Set default values if not provided
export DB_HOST=${DB_HOST:-postgres}
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-ezqrin}
export DB_PASSWORD=${DB_PASSWORD:-ezqrin_dev}
export DB_NAME=${DB_NAME:-ezqrin_db}
export DB_SSL_MODE=${DB_SSL_MODE:-disable}

echo "Rolling back migrations..."
echo "Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"

# Rollback one migration step
go run cmd/migrate/main.go step -1

echo "Rollback completed successfully!"
