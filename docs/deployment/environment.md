# Configuration Reference

## Overview

ezQRin uses **hierarchical YAML configuration** with environment variable overrides for flexible and secure configuration management. This document describes the MVP configuration needed to run the application.

> **Note**: Additional configuration options (production settings, cloud storage, wallet integration, etc.) will be added as features are implemented.

---

## Configuration Structure

### File-Based Configuration

ezQRin uses a layered configuration approach:

1. **`config/default.yaml`** - Base configuration values (committed to repository)
2. **`config/development.yaml`** - Development environment overrides (committed to repository)
3. **`config/production.yaml`** - Production environment overrides (committed to repository)
4. **Environment variables** - Secrets and runtime overrides (highest priority)

### Configuration Priority

Configuration values are loaded in the following order (highest priority first):

```
1. Environment variables (DB_USER, DB_PASSWORD, etc.)
   ↓ overrides
2. Environment-specific YAML (development.yaml or production.yaml)
   ↓ overrides
3. Base YAML (default.yaml)
```

**Example:**
- `config/default.yaml` sets `server.port: 8080`
- `config/development.yaml` overrides with `database.host: postgres`
- Environment variable `SERVER_PORT=9000` overrides both

### Secret Management

**Secrets** (passwords, API keys, tokens) are managed separately:

- **`.env.secrets`** - For local development (gitignored)
- **`docker-compose.yml` environment** - For DevContainer
- **Environment variables** - For production deployments

**Non-secrets** (timeouts, connection limits, URLs) are in YAML files and committed to the repository.

---

## Environment File Setup

### For DevContainer Users

If using DevContainer (`.devcontainer/`), secrets are automatically set via `docker-compose.yml`. **No additional setup required.**

### For Local Development

Create your secrets file:

```bash
cp .env.secrets.example .env.secrets
```

Edit `.env.secrets` with your actual values:

```bash
DB_USER=ezqrin
DB_PASSWORD=your-secure-password
DB_NAME=ezqrin_db
JWT_SECRET=your-jwt-secret-minimum-32-characters
```

**Never commit `.env.secrets` to version control.**

---

## MVP Configuration Variables

The following 20 environment variables are required for MVP operation:

### Database Configuration

#### DB_HOST

**Description:** PostgreSQL server hostname **Type:** String **Default:** `localhost`

```bash
DB_HOST=localhost
```

#### DB_PORT

**Description:** PostgreSQL server port **Type:** Integer **Default:** `5432`

```bash
DB_PORT=5432
```

#### DB_NAME

**Description:** Database name **Type:** String **Default:** `ezqrin`

```bash
DB_NAME=ezqrin_dev
```

#### DB_USER

**Description:** Database username **Type:** String **Default:** `ezqrin`

```bash
DB_USER=ezqrin
```

#### DB_PASSWORD

**Description:** Database password **Type:** String (sensitive) **Required:** Yes

```bash
DB_PASSWORD=your_secure_password_here
```

**Generate Secure Password:**

```bash
# Linux/macOS
openssl rand -base64 32
```

---

### Redis Configuration

#### REDIS_HOST

**Description:** Redis server hostname **Type:** String **Default:** `localhost`

```bash
REDIS_HOST=localhost
```

#### REDIS_PORT

**Description:** Redis server port **Type:** Integer **Default:** `6379`

```bash
REDIS_PORT=6379
```

---

### JWT Authentication

#### JWT_SECRET

**Description:** Secret key for JWT token signing **Type:** String (sensitive) **Required:** Yes
**Security:** Must be strong, randomly generated, minimum 32 characters

```bash
JWT_SECRET=your_jwt_secret_key_at_least_32_characters_long
```

**Generate Secure Secret:**

```bash
# Linux/macOS
openssl rand -base64 48
```

#### JWT_ACCESS_TOKEN_EXPIRY

**Description:** Access token expiration duration **Type:** Duration string **Default:** `15m`
**Examples:** `15m`, `1h`, `30m`

```bash
JWT_ACCESS_TOKEN_EXPIRY=15m
```

---

### Server Configuration

#### PORT

**Description:** HTTP server port **Type:** Integer **Default:** `8080`

```bash
PORT=8080
```

#### ENV

**Description:** Application environment **Type:** Enum **Options:** `development`, `staging`,
`production` **Default:** `development`

```bash
ENV=development
```

**Environment Behaviors:**

- `development`: Verbose logging, debug endpoints, relaxed security
- `staging`: Production-like testing environment
- `production`: Minimal logging, strict security, optimizations

#### LOG_LEVEL

**Description:** Logging verbosity level **Type:** Enum **Options:** `debug`, `info`, `warn`,
`error`, `fatal` **Default:** `info`

```bash
LOG_LEVEL=debug
```

**Level Usage:**

- `debug`: Development debugging (verbose)
- `info`: Standard operation information
- `warn`: Warning messages (non-critical issues)
- `error`: Error messages (operation failures)
- `fatal`: Critical errors (service shutdown)

---

### Email Configuration

#### SMTP_HOST

**Description:** SMTP server hostname **Type:** String **Example:** `smtp.mailtrap.io`
(development), `smtp.gmail.com`, `smtp.sendgrid.net`

```bash
SMTP_HOST=smtp.mailtrap.io
```

#### SMTP_PORT

**Description:** SMTP server port **Type:** Integer **Common Ports:**

- `587`: TLS (recommended)
- `465`: SSL
- `25`: Unencrypted (not recommended)

```bash
SMTP_PORT=587
```

#### SMTP_USERNAME

**Description:** SMTP authentication username **Type:** String

```bash
SMTP_USERNAME=your_smtp_username
```

#### SMTP_PASSWORD

**Description:** SMTP authentication password **Type:** String (sensitive)

```bash
SMTP_PASSWORD=your_smtp_password
```

#### SMTP_FROM_EMAIL

**Description:** Default sender email address **Type:** String (email)

```bash
SMTP_FROM_EMAIL=noreply@ezqrin.local
```

---

### Storage Configuration

#### STORAGE_TYPE

**Description:** File storage backend **Type:** Enum **Options:** `local` (MVP), `s3`, `gcs`
(future) **Default:** `local`

```bash
STORAGE_TYPE=local
```

#### STORAGE_PATH

**Description:** Local storage directory path **Type:** String **Default:** `./storage`

```bash
STORAGE_PATH=./storage
```

---

### Rate Limiting

#### RATE_LIMIT_ENABLED

**Description:** Enable/disable rate limiting **Type:** Boolean **Default:** `true`

```bash
RATE_LIMIT_ENABLED=true
```

---

## Complete Development Configuration

### .env.development Example

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ezqrin_dev
DB_USER=ezqrin
DB_PASSWORD=dev_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=dev_secret_key_do_not_use_in_production_min_32_chars
JWT_ACCESS_TOKEN_EXPIRY=15m

# Server
PORT=8080
ENV=development
LOG_LEVEL=debug

# Email (development - use Mailtrap or similar)
SMTP_HOST=smtp.mailtrap.io
SMTP_PORT=587
SMTP_USERNAME=your_mailtrap_user
SMTP_PASSWORD=your_mailtrap_password
SMTP_FROM_EMAIL=dev@ezqrin.local

# Storage
STORAGE_TYPE=local
STORAGE_PATH=./storage

# Rate Limiting
RATE_LIMIT_ENABLED=true
```

---

## Security Best Practices

### 1. Never Commit Secrets

**Add to .gitignore:**

```gitignore
.env
.env.local
.env.*.local
```

### 2. Use Strong Random Values

**Generate secure values:**

```bash
# JWT Secret (48+ characters)
openssl rand -base64 48

# Database Password (32+ characters)
openssl rand -base64 32
```

### 3. Development vs Production

- **Never** use development credentials in production
- Use separate databases for each environment
- Use different JWT secrets per environment

---

## Validation & Testing

### Verify Configuration

```bash
# Check if all required variables are set
./scripts/check-env.sh

# Test database connection
docker-compose exec api ./ezqrin db test
```

### Example check-env.sh

```bash
#!/bin/bash

required_vars=(
    "DB_HOST"
    "DB_PORT"
    "DB_NAME"
    "DB_USER"
    "DB_PASSWORD"
    "REDIS_HOST"
    "REDIS_PORT"
    "JWT_SECRET"
    "JWT_ACCESS_TOKEN_EXPIRY"
    "PORT"
    "ENV"
    "LOG_LEVEL"
    "SMTP_HOST"
    "SMTP_PORT"
    "SMTP_USERNAME"
    "SMTP_PASSWORD"
    "SMTP_FROM_EMAIL"
    "STORAGE_TYPE"
    "STORAGE_PATH"
    "RATE_LIMIT_ENABLED"
)

missing=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing+=("$var")
    fi
done

if [ ${#missing[@]} -gt 0 ]; then
    echo "❌ Missing required environment variables:"
    printf '  - %s\n' "${missing[@]}"
    exit 1
else
    echo "✅ All required environment variables are set"
    exit 0
fi
```

---

## Troubleshooting

### Environment Variables Not Loading

**Check:**

1. `.env` file exists in project root
2. No syntax errors (no spaces around `=`)
3. No quotes around values (unless intentional)
4. Line endings are LF (not CRLF)

### Docker Compose Not Using .env

**Ensure:**

1. `.env` file is in same directory as `docker-compose.yml`
2. Variables use `${VAR_NAME}` syntax in compose file
3. Restart containers after changing `.env`:

```bash
docker-compose down
docker-compose up -d
```

### JWT Errors

**Check:**

- `JWT_SECRET` is set and matches across instances
- `JWT_SECRET` is sufficiently long (32+ characters)
- Tokens not expired (check `JWT_ACCESS_TOKEN_EXPIRY`)

### Database Connection Errors

**Verify:**

- Database host is reachable: `ping $DB_HOST`
- Port is correct: `telnet $DB_HOST $DB_PORT`
- Credentials are correct

---

## Future Configuration

As features are implemented, additional environment variables will be added for:

- **Production Settings**: SSL configuration, connection pooling, advanced caching
- **Cloud Storage**: S3, GCS integration (see [Future Features](../api/future_features.md))
- **Wallet Integration**: Apple Wallet, Google Wallet (see
  [Future Features](../api/future_features.md))
- **Advanced Rate Limiting**: Granular rate limit configuration
- **CORS Configuration**: Production CORS origins
- **Email Templates**: Additional email service providers

These will be documented as they are implemented.

---

## Related Documentation

- [Docker Deployment](./docker.md)
- [Security Design](../architecture/security.md)
- [System Architecture](../architecture/overview.md)
- [Future Features](../api/future_features.md)
