#!/bin/bash

# migrate-up.sh - Apply all pending database migrations

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

echo "Running migrations..."
echo "Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"

# Run migrations
go run cmd/migrate/main.go up

echo "Migrations completed successfully!"
