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
- **`docker-compose.yaml` environment** - For DevContainer
- **Environment variables** - For production deployments

**Non-secrets** (timeouts, connection limits, URLs) are in YAML files and committed to the repository.

---

## Environment File Setup

### For DevContainer Users

If using DevContainer (`.devcontainer/`), secrets are automatically set via `docker-compose.yaml`. **No additional setup required.**

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

The following 21 environment variables are required for MVP operation:

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

### QR Code Configuration

#### QR_HMAC_SECRET

**Description:** Secret key for HMAC-SHA256 signing of QR code tokens **Type:** String (sensitive) **Required:** Yes
**Security:** Must be strong, randomly generated, minimum 32 characters

```bash
QR_HMAC_SECRET=your_qr_hmac_secret_key_at_least_32_characters_long
```

**Purpose:** QR code tokens are signed with this secret to prevent forgery. The format is:
`evt_{event_id}_prt_{participant_id}_{random}.{hmac_signature}`

Check-in validation rejects tokens whose signature does not match, protecting against forged QR codes.

**Generate Secure Secret:**

```bash
# Linux/macOS
openssl rand -base64 48
```

---

### Server Configuration

#### PORT

**Description:** HTTP server port **Type:** Integer **Default:** `8080`

```bash
PORT=8080
```

#### ENV

**Description:** Application environment **Type:** Enum **Options:** `development`, `production`,
`test` **Default:** `development`

```bash
ENV=development
```

**Environment Behaviors:**

- `development`: Verbose logging, debug endpoints, relaxed security
- `production`: Minimal logging, strict security, optimizations
- `test`: Test environment for automated testing

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

ezQRin supports two email backends selected via `EMAIL_BACKEND`.

#### EMAIL_BACKEND

**Description:** Email sending backend
**Type:** Enum
**Options:** `smtp`, `gmail`
**Default:** `smtp`

```bash
EMAIL_BACKEND=smtp
```

#### EMAIL_FROM_ADDRESS

**Description:** Sender email address shown in the "From" header
**Type:** String (email)
**Default:** `noreply@ezqrin.local`

```bash
EMAIL_FROM_ADDRESS=noreply@ezqrin.local
```

#### EMAIL_FROM_NAME

**Description:** Display name shown alongside the from address
**Type:** String
**Default:** `ezQRin`

```bash
EMAIL_FROM_NAME=ezQRin
```

#### EMAIL_PLAIN_TEXT_ONLY

**Description:** Send plain-text emails only (no HTML)
**Type:** Boolean
**Default:** `false`

```bash
EMAIL_PLAIN_TEXT_ONLY=false
```

---

#### SMTP Settings (`EMAIL_BACKEND=smtp`)

| Variable | Description | Default |
|----------|-------------|---------|
| `EMAIL_SMTP_HOST` | SMTP server hostname | `localhost` |
| `EMAIL_SMTP_PORT` | SMTP server port | `1025` |
| `EMAIL_SMTP_USER` | SMTP authentication username | *(empty)* |
| `EMAIL_SMTP_PASSWORD` | SMTP authentication password *(sensitive)* | *(empty)* |
| `EMAIL_SMTP_TLS` | Enable STARTTLS (`true`/`false`) | `false` |

**Development (MailHog):**

```bash
EMAIL_SMTP_HOST=localhost
EMAIL_SMTP_PORT=1025
EMAIL_SMTP_TLS=false
```

**Production (e.g., SendGrid / Amazon SES):**

```bash
EMAIL_SMTP_HOST=smtp.sendgrid.net
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USER=apikey
EMAIL_SMTP_PASSWORD=<your-api-key>
EMAIL_SMTP_TLS=true
```

---

#### Gmail API Settings (`EMAIL_BACKEND=gmail`)

| Variable | Description |
|----------|-------------|
| `EMAIL_GMAIL_CLIENT_ID` | OAuth2 client ID *(sensitive)* |
| `EMAIL_GMAIL_CLIENT_SECRET` | OAuth2 client secret *(sensitive)* |
| `EMAIL_GMAIL_REFRESH_TOKEN` | OAuth2 refresh token *(sensitive)* |

**Setup Guide:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/) and create or select a project.
2. Enable the **Gmail API** (`APIs & Services → Library → Gmail API → Enable`).
3. Create an OAuth2 credential (`APIs & Services → Credentials → Create Credentials → OAuth client ID`).
   - Application type: **Desktop app**
4. Download the client secret JSON and note `client_id` and `client_secret`.
5. Grant the `https://www.googleapis.com/auth/gmail.send` scope and obtain a refresh token using the OAuth2 authorization flow (e.g., with `oauth2l` or a small helper script).
6. Set the environment variables:

```bash
EMAIL_GMAIL_CLIENT_ID=123456789-xxxx.apps.googleusercontent.com
EMAIL_GMAIL_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxx
EMAIL_GMAIL_REFRESH_TOKEN=1//xxxxxxxxxxxxxxxxx
```

---

### Telemetry / OpenTelemetry Configuration

ezQRin exports traces, metrics, and logs via OpenTelemetry. All telemetry settings are optional
with safe defaults for local development.

#### OTEL_ENABLED

**Description:** Master switch that enables or disables all OpenTelemetry instrumentation.
When `false`, all OTel providers are replaced with no-op implementations and no Collector
connection is attempted.
**Type:** Boolean
**Default:** `true`
**Required:** No

```bash
OTEL_ENABLED=true
```

#### OTEL_SERVICE_NAME

**Description:** The service name attached to all traces, metrics, and log records. This is the
identifier shown in the Jaeger service dropdown and Prometheus label `service_name`.
**Type:** String
**Default:** `ezqrin-server`
**Required:** No

```bash
OTEL_SERVICE_NAME=ezqrin-server
```

#### OTEL_EXPORTER_OTLP_ENDPOINT

**Description:** The gRPC endpoint of the OTel Collector that the application sends telemetry to.
All three signals (traces, metrics, logs) share this endpoint by default.
**Type:** String (host:port)
**Default:** `localhost:4317`
**Required:** No

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

#### OTEL_EXPORTER_OTLP_INSECURE

**Description:** Disables TLS verification on the gRPC connection to the Collector. Set to `true`
in local development (where the Collector runs over plain HTTP). Set to `false` in production
environments where TLS is required.
**Type:** Boolean
**Default:** `true`
**Required:** No

```bash
OTEL_EXPORTER_OTLP_INSECURE=true
```

#### OTEL_TRACES_SAMPLER

**Description:** Controls the trace sampling strategy. Determines which requests generate a trace.
**Type:** Enum
**Options:** `always_on`, `always_off`, `traceidratio`
**Default:** `always_on`
**Required:** No

| Value | Behavior |
|-------|----------|
| `always_on` | Every request is traced. Use in local development. |
| `always_off` | No traces are created. Disables tracing while keeping metrics and logs active. |
| `traceidratio` | A fraction of requests are traced, determined by `OTEL_TRACES_SAMPLER_ARG`. |

```bash
OTEL_TRACES_SAMPLER=always_on
```

#### OTEL_TRACES_SAMPLER_ARG

**Description:** The sampling ratio when `OTEL_TRACES_SAMPLER=traceidratio`. A value of `1.0`
traces every request; `0.1` traces approximately 10% of requests.
**Type:** Float (0.0–1.0)
**Default:** `1.0`
**Required:** No (only relevant when sampler is `traceidratio`)

```bash
OTEL_TRACES_SAMPLER_ARG=0.1
```

#### OTEL_LOGS_EXPORTER

**Description:** Controls where structured log records are exported. When set to `otlp`, logs are
sent to the Collector alongside traces and metrics (and forwarded to Loki). When set to `none`,
log export is disabled while traces and metrics continue to be exported. Stdout logging is always
active regardless of this setting.
**Type:** Enum
**Options:** `otlp`, `none`
**Default:** `otlp`
**Required:** No

```bash
OTEL_LOGS_EXPORTER=otlp
```

---

**Sampling strategy examples:**

```bash
# Local development — trace everything
OTEL_ENABLED=true
OTEL_TRACES_SAMPLER=always_on
OTEL_TRACES_SAMPLER_ARG=1.0
OTEL_LOGS_EXPORTER=otlp

# Production-like — 10% sampling, TLS enabled
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.internal:4317
OTEL_EXPORTER_OTLP_INSECURE=false
OTEL_TRACES_SAMPLER=traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
OTEL_LOGS_EXPORTER=otlp
```

For the full local stack setup, UI access, and troubleshooting, see
[Observability Operations Guide](./observability.md).

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

# QR Code
QR_HMAC_SECRET=dev_qr_hmac_secret_do_not_use_in_production_min_32_chars

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

# QR HMAC Secret (48+ characters)
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
    "QR_HMAC_SECRET"
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

1. `.env` file is in same directory as `docker-compose.yaml`
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
