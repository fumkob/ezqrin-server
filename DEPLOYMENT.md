# Production Deployment Guide

This guide covers deploying ezQRin Server to a production environment using Docker and
Docker Compose.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Production Setup](#production-setup)
- [Environment Variables](#environment-variables)
- [Running Migrations](#running-migrations)
- [Health Checks and Monitoring](#health-checks-and-monitoring)
- [Backup and Recovery](#backup-and-recovery)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Infrastructure Requirements

- Linux server with Docker Engine 20.10+ and Docker Compose v2+
- CPU: 2+ cores
- RAM: 4GB minimum (8GB recommended)
- Disk: 20GB free space (additional for data growth)
- Outbound network access for image pulls

### Software

**Option A: Docker**

```bash
# Install Docker (Ubuntu/Debian example)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Verify installation
docker --version
docker compose version
```

**Option B: Podman**

```bash
# Install Podman (Ubuntu/Debian example)
sudo apt-get install -y podman podman-compose

# Verify installation
podman --version
podman-compose --version
```

> **Note:** Podman runs daemonless and rootless by default. All `docker` commands in this guide can be replaced with `podman`, and `docker compose` with `podman-compose`.

---

## Production Setup

### 1. Clone the Repository

```bash
git clone https://github.com/fumkob/ezqrin-server.git
cd ezqrin-server
```

### 2. Build the Production Image

The production image is built from the `Dockerfile` at the repository root using a
multi-stage build (builder: `golang:1.25.5-alpine`, runtime: `alpine:3.21`).

```bash
# Docker
docker build -t ezqrin-server:latest .

# Podman
podman build -t ezqrin-server:latest .
```

### 3. Configure Secrets

Copy the secrets example and fill in production values:

```bash
cp .env.secrets.example .env.secrets
chmod 600 .env.secrets
```

Edit `.env.secrets` with real production values:

```bash
# Generate secure values
openssl rand -base64 32   # For DB_PASSWORD
openssl rand -base64 48   # For JWT_SECRET (longer for production)
openssl rand -base64 32   # For QR_HMAC_SECRET
openssl rand -base64 24   # For REDIS_PASSWORD
```

Required values in `.env.secrets`:

```bash
DB_USER=ezqrin_prod
DB_PASSWORD=<strong-random-password>
DB_NAME=ezqrin_db
JWT_SECRET=<strong-random-secret-min-32-chars>
QR_HMAC_SECRET=<strong-random-secret-min-32-chars>
REDIS_PASSWORD=<strong-random-password>
```

**CRITICAL:** Never commit `.env.secrets` to version control. Verify it is listed in `.gitignore`.

### 4. Start Production Services

```bash
# Docker
docker compose -f docker-compose.prod.yml up -d

# Podman
podman-compose -f docker-compose.prod.yml up -d
```

Check that all services are running:

```bash
# Docker
docker compose -f docker-compose.prod.yml ps

# Podman
podman-compose -f docker-compose.prod.yml ps
```

Expected output:

```
NAME                    STATUS          PORTS
ezqrin-api-prod         Up (healthy)    0.0.0.0:8080->8080/tcp
ezqrin-postgres-prod    Up (healthy)
ezqrin-redis-prod       Up (healthy)
```

### 5. Run Database Migrations

Migrations must be applied after the first deployment and after any update that includes
schema changes. The production image includes a compiled migration binary (`ezqrin-migrate`):

```bash
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up

# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up
```

To check the current migration version before running:

```bash
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate version

# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate version
```

### 6. Verify the Deployment

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:

```json
{ "status": "ok" }
```

---

## Environment Variables

### Secret Variables (`.env.secrets`)

These must never be committed to version control.

| Variable         | Description                                              | Required |
| ---------------- | -------------------------------------------------------- | -------- |
| `DB_USER`        | PostgreSQL username                                      | Yes      |
| `DB_PASSWORD`    | PostgreSQL password (generate with `openssl rand -base64 32`) | Yes |
| `DB_NAME`        | PostgreSQL database name                                 | Yes      |
| `JWT_SECRET`     | JWT signing secret (minimum 32 characters, use 48+ for production) | Yes |
| `QR_HMAC_SECRET` | HMAC-SHA256 secret for QR code signing (minimum 32 characters) | Yes |
| `REDIS_PASSWORD` | Redis password (leave empty to disable Redis auth)       | No       |

### Non-Secret Variables (set in `docker-compose.prod.yml` or override)

| Variable              | Default        | Description                                         |
| --------------------- | -------------- | --------------------------------------------------- |
| `SERVER_ENV`          | `production`   | Application environment                             |
| `SERVER_PORT`         | `8080`         | HTTP server port                                    |
| `DB_HOST`             | `postgres`     | PostgreSQL hostname (Docker service name)           |
| `DB_PORT`             | `5432`         | PostgreSQL port                                     |
| `DB_SSL_MODE`         | `require`      | SSL mode (`require` for production)                 |
| `REDIS_HOST`          | `redis`        | Redis hostname (Docker service name)                |
| `REDIS_PORT`          | `6379`         | Redis port                                          |
| `LOG_LEVEL`           | `info`         | Log verbosity (`debug`, `info`, `warn`, `error`)    |
| `LOG_FORMAT`          | `json`         | Log format (`json` for production)                  |

### YAML Configuration

Non-secret configuration is managed through YAML files committed to the repository.
Production uses `config/default.yaml` and `config/production.yaml`.

Key production settings in `config/production.yaml`:

```yaml
server:
  environment: production

database:
  ssl_mode: require

logging:
  level: info
  format: json
```

See [`docs/deployment/environment.md`](./docs/deployment/environment.md) for the full
configuration reference.

---

## Running Migrations

The production Docker image includes a pre-compiled `ezqrin-migrate` binary alongside the main
`ezqrin-server` binary. Use this binary to manage database schema migrations.

### Apply All Pending Migrations

```bash
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up

# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up
```

### Check Current Migration Version

```bash
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate version

# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate version
```

### Rollback Last Migration

```bash
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate down

# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate down
```

### Migration Best Practices

- Always take a database backup before running migrations in production.
- Test migrations in a staging environment first.
- Migrations are located in `internal/infrastructure/database/migrations/`.
- Each migration has an `.up.sql` and `.down.sql` file for rollback support.

---

## Health Checks and Monitoring

### Health Check Endpoints

| Endpoint       | Purpose                                           | Expected Response              |
| -------------- | ------------------------------------------------- | ------------------------------ |
| `GET /api/v1/health`       | Basic application health check                   | `200 OK` with `{"status":"ok"}` |
| `GET /api/v1/health/ready` | Readiness probe (DB and Redis connectivity)      | `200 OK` when ready            |
| `GET /api/v1/health/live`  | Liveness probe (process alive)                   | `200 OK` always                |

Use `/health/ready` and `/health/live` for Kubernetes probes or container orchestration.

### Monitor Logs

```bash
# Follow all service logs
# Docker
docker compose -f docker-compose.prod.yml logs -f
# Podman
podman-compose -f docker-compose.prod.yml logs -f

# API server logs only
# Docker
docker compose -f docker-compose.prod.yml logs -f api
# Podman
podman-compose -f docker-compose.prod.yml logs -f api

# Last 100 lines
# Docker
docker compose -f docker-compose.prod.yml logs --tail=100 api
# Podman
podman-compose -f docker-compose.prod.yml logs --tail=100 api
```

Logs are structured JSON in production, suitable for ingestion by log aggregation systems
(Loki, Elasticsearch, CloudWatch, etc.).

### Key Metrics to Monitor

- API response latency (p50, p95, p99)
- HTTP error rate (4xx, 5xx)
- PostgreSQL connection pool utilization
- Redis cache hit rate
- Database query latency

---

## Updating the Application

### Zero-Downtime Update Process

```bash
# 1. Pull latest code
git pull origin main

# 2. Build new image
# Docker
docker build -t ezqrin-server:latest .
# Podman
podman build -t ezqrin-server:latest .

# 3. Apply migrations (if any schema changes)
# Docker
docker compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up
# Podman
podman-compose -f docker-compose.prod.yml exec api \
    ./ezqrin-migrate up

# 4. Restart the API service (other services continue running)
# Docker
docker compose -f docker-compose.prod.yml up -d --no-deps api
# Podman
podman-compose -f docker-compose.prod.yml up -d --no-deps api
```

### Verify Update

```bash
curl http://localhost:8080/api/v1/health
# Docker
docker compose -f docker-compose.prod.yml logs --tail=50 api
# Podman
podman-compose -f docker-compose.prod.yml logs --tail=50 api
```

---

## Backup and Recovery

### Database Backup

```bash
# Create a timestamped backup
# Docker
docker compose -f docker-compose.prod.yml exec postgres \
    pg_dump -U ${DB_USER} ${DB_NAME} \
    > backup_$(date +%Y%m%d_%H%M%S).sql
# Podman
podman-compose -f docker-compose.prod.yml exec postgres \
    pg_dump -U ${DB_USER} ${DB_NAME} \
    > backup_$(date +%Y%m%d_%H%M%S).sql

# Compress the backup
gzip backup_$(date +%Y%m%d_%H%M%S).sql
```

Automate with a cron job:

```bash
# Add to crontab (daily at 2:00 AM)
0 2 * * * docker compose -f /path/to/ezqrin-server/docker-compose.prod.yml exec -T postgres \
    pg_dump -U ezqrin_prod ezqrin_db | gzip > /backups/ezqrin_$(date +\%Y\%m\%d).sql.gz
```

### Database Restore

```bash
# Stop the API to prevent writes during restore
# Docker
docker compose -f docker-compose.prod.yml stop api
# Podman
podman-compose -f docker-compose.prod.yml stop api

# Restore from backup
# Docker
cat backup.sql | docker compose -f docker-compose.prod.yml exec -T postgres \
    psql -U ${DB_USER} ${DB_NAME}
# Podman
cat backup.sql | podman-compose -f docker-compose.prod.yml exec -T postgres \
    psql -U ${DB_USER} ${DB_NAME}

# Restart the API
# Docker
docker compose -f docker-compose.prod.yml start api
# Podman
podman-compose -f docker-compose.prod.yml start api
```

### Volume Backup

Named Docker volumes (`postgres-data`, `redis-data`) store persistent data.
Back up volumes directly:

```bash
# Backup postgres volume
docker run --rm \
    --volumes-from ezqrin-postgres-prod \
    -v $(pwd)/backups:/backup alpine \
    tar czf /backup/postgres-data-$(date +%Y%m%d).tar.gz /var/lib/postgresql/data
```

---

## Troubleshooting

### Service Not Starting

Check container logs for errors:

```bash
# Docker
docker compose -f docker-compose.prod.yml logs api
docker compose -f docker-compose.prod.yml logs postgres
docker compose -f docker-compose.prod.yml logs redis

# Podman
podman-compose -f docker-compose.prod.yml logs api
podman-compose -f docker-compose.prod.yml logs postgres
podman-compose -f docker-compose.prod.yml logs redis
```

Verify all services are healthy:

```bash
# Docker
docker compose -f docker-compose.prod.yml ps

# Podman
podman-compose -f docker-compose.prod.yml ps
```

### API Returns 503 or Connection Errors

The API depends on PostgreSQL and Redis being healthy before it starts. Check dependency health:

```bash
# Check PostgreSQL
# Docker
docker compose -f docker-compose.prod.yml exec postgres \
    pg_isready -U ${DB_USER} -d ${DB_NAME}
# Podman
podman-compose -f docker-compose.prod.yml exec postgres \
    pg_isready -U ${DB_USER} -d ${DB_NAME}

# Check Redis
# Docker
docker compose -f docker-compose.prod.yml exec redis \
    redis-cli -a ${REDIS_PASSWORD} ping
# Podman
podman-compose -f docker-compose.prod.yml exec redis \
    redis-cli -a ${REDIS_PASSWORD} ping
```

### Database Connection Refused

Verify `.env.secrets` contains correct credentials:

```bash
# Print non-sensitive env vars (do not log DB_PASSWORD or JWT_SECRET)
# Docker
docker compose -f docker-compose.prod.yml exec api env | grep DB_HOST
docker compose -f docker-compose.prod.yml exec api env | grep DB_PORT
# Podman
podman-compose -f docker-compose.prod.yml exec api env | grep DB_HOST
podman-compose -f docker-compose.prod.yml exec api env | grep DB_PORT
```

Confirm `DB_SSL_MODE=require` is set for production and that the PostgreSQL instance
accepts SSL connections.

### JWT Authentication Errors

- Verify `JWT_SECRET` is set and identical across all API instances.
- Confirm `JWT_SECRET` is at least 32 characters.
- Check token expiry settings (`JWT_ACCESS_TOKEN_EXPIRY`).

### Port Already in Use

If port 8080 is occupied:

```bash
# Find the process using port 8080
lsof -i :8080

# Or change the port mapping in docker-compose.prod.yml
ports:
  - "9090:8080"  # Use 9090 on the host instead
```

### Out of Disk Space

Check volume sizes:

```bash
# Docker
docker system df
docker volume ls

# Podman
podman system df
podman volume ls
```

Clean up unused Docker resources (be careful in production):

```bash
# Remove stopped containers and dangling images (safe)
docker container prune
docker image prune

# Remove unused volumes (DANGEROUS: verify before running)
docker volume prune
```

---

## Security Checklist

Before going live, verify the following:

- `.env.secrets` is not committed to version control
- All secrets use strong random values (`openssl rand`)
- `DB_SSL_MODE=require` is set for PostgreSQL connections
- Redis has a strong password configured
- The API is not directly exposed to the internet without a reverse proxy (nginx, Caddy, etc.)
- TLS termination is handled at the load balancer or reverse proxy level
- Log output does not contain secrets or PII
- Regular database backups are scheduled and tested

---

## Related Documentation

- [Configuration Reference](./docs/deployment/environment.md)
- [DevContainer Guide](./docs/deployment/docker.md)
- [Architecture Overview](./docs/architecture/overview.md)
- [Database Schema](./docs/architecture/database.md)
- [Security Design](./docs/architecture/security.md)
- [API Reference](./api/openapi.yaml)
