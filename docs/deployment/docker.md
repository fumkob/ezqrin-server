# Docker Deployment Guide

## Overview

This guide covers deploying ezQRin using Docker and Docker Compose for local development, testing,
and production environments.

---

## Prerequisites

**Required Software:**

- Docker Engine 20.10+
- Docker Compose 2.0+
- Git

**System Requirements:**

- CPU: 2+ cores
- RAM: 4GB minimum (8GB recommended)
- Disk: 10GB free space

---

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/ezqrin/ezqrin-server.git
cd ezqrin-server
```

### 2. Environment Configuration

```bash
cp .env.example .env
```

Edit `.env` file with your configuration (see [Environment Variables](./environment.md))

### 3. Start Services

```bash
docker-compose up -d
```

### 4. Run Migrations

```bash
docker-compose exec api ./ezqrin migrate up
```

### 5. Verify Installation

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "timestamp": "2025-11-08T10:00:00Z"
}
```

---

## Docker Compose Configuration

### Development Setup (docker-compose.yml)

```yaml
version: "3.8"

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
      target: development
    container_name: ezqrin-api
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=${DB_NAME}
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=${JWT_SECRET}
      - ENV=development
    volumes:
      - ./:/app
      - go-modules:/go/pkg/mod
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - ezqrin-network
    restart: unless-stopped

  postgres:
    image: postgres:18-alpine
    container_name: ezqrin-postgres
    environment:
      - POSTGRES_DB=${DB_NAME}
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - ezqrin-network
    restart: unless-stopped

  redis:
    image: redis:8-alpine
    container_name: ezqrin-redis
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
    restart: unless-stopped

  # Optional: Database administration tool
  pgadmin:
    image: dpage/pgadmin4:latest
    container_name: ezqrin-pgadmin
    environment:
      - PGADMIN_DEFAULT_EMAIL=admin@ezqrin.local
      - PGADMIN_DEFAULT_PASSWORD=admin
    ports:
      - "5050:80"
    depends_on:
      - postgres
    networks:
      - ezqrin-network
    restart: unless-stopped
    profiles:
      - tools

volumes:
  postgres-data:
  redis-data:
  go-modules:

networks:
  ezqrin-network:
    driver: bridge
```

---

### Production Setup (docker-compose.prod.yml)

```yaml
version: "3.8"

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
      target: production
    image: ezqrin/api:latest
    container_name: ezqrin-api-prod
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=${DB_NAME}
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=${JWT_SECRET}
      - ENV=production
      - LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - ezqrin-network
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          cpus: "2"
          memory: 2G
        reservations:
          cpus: "1"
          memory: 1G

  postgres:
    image: postgres:18-alpine
    container_name: ezqrin-postgres-prod
    environment:
      - POSTGRES_DB=${DB_NAME}
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./backups:/backups
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 30s
      timeout: 10s
      retries: 5
    networks:
      - ezqrin-network
    restart: always
    deploy:
      resources:
        limits:
          cpus: "2"
          memory: 2G
        reservations:
          cpus: "1"
          memory: 1G

  redis:
    image: redis:8-alpine
    container_name: ezqrin-redis-prod
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
    environment:
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5
    networks:
      - ezqrin-network
    restart: always

  # Nginx reverse proxy (optional)
  nginx:
    image: nginx:alpine
    container_name: ezqrin-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - api
    networks:
      - ezqrin-network
    restart: always
    profiles:
      - with-nginx

volumes:
  postgres-data:
  redis-data:

networks:
  ezqrin-network:
    driver: bridge
```

---

## Dockerfile

### Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ezqrin ./cmd/api

# Development stage
FROM golang:1.25.4-alpine AS development

# Install development tools
RUN apk add --no-cache git make curl

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Install air for hot reload
RUN go install github.com/cosmtrek/air@latest

# Copy source
COPY . .

EXPOSE 8080

# Use air for hot reload in development
CMD ["air", "-c", ".air.toml"]

# Production stage
FROM alpine:latest AS production

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S ezqrin && \
    adduser -u 1001 -S ezqrin -G ezqrin

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/ezqrin .

# Copy migrations (if needed)
COPY --from=builder /app/internal/infrastructure/database/migration/migrations ./migrations

# Change ownership
RUN chown -R ezqrin:ezqrin /app

# Switch to non-root user
USER ezqrin

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./ezqrin"]
```

---

## Common Docker Commands

### Start Services

```bash
# Development
docker-compose up -d

# Production
docker-compose -f docker-compose.prod.yml up -d

# With specific profile
docker-compose --profile tools up -d
```

### Stop Services

```bash
docker-compose down

# Remove volumes (WARNING: deletes data)
docker-compose down -v
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api

# Last 100 lines
docker-compose logs --tail=100 api
```

### Execute Commands

```bash
# Run migration
docker-compose exec api ./ezqrin migrate up

# Open shell in container
docker-compose exec api sh

# Run tests
docker-compose exec api go test ./...
```

### Rebuild Images

```bash
# Rebuild single service
docker-compose build api

# Rebuild without cache
docker-compose build --no-cache api

# Rebuild and restart
docker-compose up -d --build api
```

---

## Database Management

### Run Migrations

```bash
# Up migrations
docker-compose exec api ./ezqrin migrate up

# Down migrations
docker-compose exec api ./ezqrin migrate down

# Specific version
docker-compose exec api ./ezqrin migrate goto 3
```

### Database Backup

```bash
# Create backup
docker-compose exec postgres pg_dump -U ezqrin ezqrin > backup.sql

# Or using docker exec
docker exec ezqrin-postgres pg_dump -U ezqrin ezqrin > backup.sql

# Restore backup
docker-compose exec -T postgres psql -U ezqrin ezqrin < backup.sql
```

### Connect to Database

```bash
# Using psql in container
docker-compose exec postgres psql -U ezqrin -d ezqrin

# Using external psql
psql -h localhost -p 5432 -U ezqrin -d ezqrin
```

---

## Redis Management

### Connect to Redis

```bash
# Redis CLI
docker-compose exec redis redis-cli

# With password (production)
docker-compose exec redis redis-cli -a ${REDIS_PASSWORD}
```

### Clear Cache

```bash
# Flush all caches
docker-compose exec redis redis-cli FLUSHALL

# Flush specific database
docker-compose exec redis redis-cli FLUSHDB
```

---

## Monitoring & Health Checks

### Check Container Status

```bash
# List running containers
docker-compose ps

# Check health status
docker-compose ps | grep healthy
```

### Resource Usage

```bash
# Live resource monitoring
docker stats

# Specific container
docker stats ezqrin-api
```

### Application Health

```bash
# Health endpoint
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/health/ready

# Liveness check
curl http://localhost:8080/health/live
```

---

## Production Deployment

### Pre-Deployment Checklist

- [ ] Environment variables configured
- [ ] JWT secret generated (strong random string)
- [ ] Database password set (strong password)
- [ ] Redis password configured
- [ ] SSL certificates obtained
- [ ] Firewall rules configured
- [ ] Backup strategy established
- [ ] Monitoring tools set up

### Deployment Steps

1. **Pull Latest Code:**

```bash
git pull origin main
```

2. **Build Production Image:**

```bash
docker-compose -f docker-compose.prod.yml build
```

3. **Stop Old Containers:**

```bash
docker-compose -f docker-compose.prod.yml down
```

4. **Start New Containers:**

```bash
docker-compose -f docker-compose.prod.yml up -d
```

5. **Run Migrations:**

```bash
docker-compose -f docker-compose.prod.yml exec api ./ezqrin migrate up
```

6. **Verify Health:**

```bash
curl https://api.ezqrin.com/health
```

---

## Scaling

### Horizontal Scaling (Multiple API Instances)

```yaml
services:
  api:
    # ... other config
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
```

### Load Balancer Configuration

**Nginx (nginx.conf):**

```nginx
upstream api_backend {
    least_conn;
    server api:8080 max_fails=3 fail_timeout=30s;
    server api-2:8080 max_fails=3 fail_timeout=30s;
    server api-3:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name api.ezqrin.com;

    location / {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs api

# Check if port is in use
lsof -i :8080

# Remove and recreate
docker-compose down
docker-compose up -d
```

### Database Connection Issues

```bash
# Verify postgres is running
docker-compose ps postgres

# Check network connectivity
docker-compose exec api ping postgres

# Verify credentials
docker-compose exec postgres psql -U ezqrin -d ezqrin
```

### Performance Issues

```bash
# Check resource usage
docker stats

# Check database connections
docker-compose exec postgres psql -U ezqrin -c "SELECT count(*) FROM pg_stat_activity;"

# Check Redis memory
docker-compose exec redis redis-cli INFO memory
```

### Clean Up Resources

```bash
# Remove stopped containers
docker-compose rm

# Remove unused images
docker image prune -a

# Remove unused volumes (WARNING: data loss)
docker volume prune

# Complete cleanup (WARNING: removes all Docker resources)
docker system prune -a --volumes
```

---

## Security Best Practices

### Container Security

1. **Run as non-root user** (implemented in Dockerfile)
2. **Use official base images** (alpine variants)
3. **Keep images updated** regularly
4. **Scan for vulnerabilities:**

```bash
docker scan ezqrin/api:latest
```

### Network Security

1. **Use internal networks** for service communication
2. **Expose only necessary ports**
3. **Use TLS/SSL** for external communication
4. **Implement firewall rules**

### Data Security

1. **Encrypt volumes** (host-level encryption)
2. **Secure database passwords** (strong, randomized)
3. **Backup regularly** with encryption
4. **Use secrets management** (Docker secrets, Vault)

---

## Related Documentation

- [Environment Variables](./environment.md)
- [System Architecture](../architecture/overview.md)
- [Security Design](../architecture/security.md)
