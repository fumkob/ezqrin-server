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
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community

# Install Delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Install Air for hot reload
RUN go install github.com/air-verse/air@latest

# Install golangci-lint
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.2

# Create non-root user for VS Code
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN addgroup -g $USER_GID $USERNAME \
    && adduser -D -u $USER_UID -G $USERNAME $USERNAME \
    && mkdir -p /home/$USERNAME/.cache \
    && chown -R $USERNAME:$USERNAME /home/$USERNAME

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
