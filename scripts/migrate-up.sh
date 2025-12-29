#!/bin/bash

# migrate-up.sh - Apply all pending database migrations

set -e

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    export $(grep -v '^#' .env | xargs)
fi

# Set default values if not provided
export DATABASE_HOST=${DATABASE_HOST:-postgres}
export DATABASE_PORT=${DATABASE_PORT:-5432}
export DATABASE_USER=${DATABASE_USER:-ezqrin}
export DATABASE_PASSWORD=${DATABASE_PASSWORD:-ezqrin_dev}
export DATABASE_NAME=${DATABASE_NAME:-ezqrin_db}
export DATABASE_SSLMODE=${DATABASE_SSLMODE:-disable}

echo "Running migrations..."
echo "Database: $DATABASE_USER@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME"

# Run migrations
go run cmd/migrate/main.go up

echo "Migrations completed successfully!"
