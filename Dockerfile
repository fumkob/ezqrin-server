# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /build

# Download dependencies first (layer cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build binaries
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" -o ezqrin-server ./cmd/api/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" -o ezqrin-migrate ./cmd/migrate/main.go

# Production stage
FROM alpine:3.21 AS production

# curl for health checks, ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates curl

WORKDIR /app

# Copy binaries
COPY --from=builder /build/ezqrin-server .
COPY --from=builder /build/ezqrin-migrate .

# Copy runtime config (YAML only, no .go or _test.go files)
COPY config/default.yaml ./config/
COPY config/production.yaml ./config/

# Copy migration SQL files
COPY internal/infrastructure/database/migrations/ ./internal/infrastructure/database/migrations/

EXPOSE 8080

# Run as non-root
USER nobody

CMD ["./ezqrin-server"]
