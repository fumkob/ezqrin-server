# Development container for ezQRin Server
FROM golang:1.25.5-alpine

# Install system dependencies
RUN apk add --no-cache \
    git \
    make \
    curl \
    bash \
    postgresql-client \
    redis \
    github-cli \
    nodejs \
    npm \
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community

# Install Delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Install Air for hot reload
RUN go install github.com/air-verse/air@latest

# Install golangci-lint (supports Go 1.25+)
RUN curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2

# Install oapi-codegen for API code generation
RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1

# Install swagger-cli for OpenAPI bundling
RUN npm install -g @apidevtools/swagger-cli@4.0.4

# Create non-root user for VS Code
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN addgroup -g $USER_GID $USERNAME \
    && adduser -D -u $USER_UID -G $USERNAME $USERNAME \
    && mkdir -p /home/$USERNAME/.cache \
    && chown -R $USERNAME:$USERNAME /home/$USERNAME \
    && chown -R $USERNAME:$USERNAME /go

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER $USERNAME

# Expose ports
# 8080: API server
# 2345: Delve debugger
EXPOSE 8080 2345

# Default command (will be overridden by devcontainer.json)
CMD ["sleep", "infinity"]
