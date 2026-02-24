# Govern Deployment Guide

## Overview

This guide covers deployment strategies and best practices for applications using the Govern library. Govern is designed to be production-ready with minimal configuration required for common deployment scenarios.

## Prerequisites

### Development Environment
- Go 1.25 or higher
- Git
- Access to PostgreSQL 12+ (if using database/postgres package)
- Access to Redis 6+ (if using database/redis package)

### Production Environment
- Linux or macOS server (Windows supported but less common)
- PostgreSQL database server (if applicable)
- Redis server (if applicable)
- Reverse proxy (nginx, HAProxy, or Traefik) recommended

## Installation

### Using Go Modules

```bash
# Initialize a new project
go mod init myservice

# Add Govern to your project
go get github.com/haipham22/govern

# Download dependencies
go mod download
```

### Importing Packages

```go
import (
    governhttp "github.com/haipham22/govern/http"
    httpEcho "github.com/haipham22/govern/http/echo"
    "github.com/haipham22/govern/database/postgres"
    "github.com/haipham22/govern/database/redis"
    "github.com/haipham22/govern/log"
)
```

## Configuration

### Environment Variables

Govern recommends using environment variables for configuration:

```bash
# Server
export SERVER_PORT=8080
export SERVER_SHUTDOWN_TIMEOUT=30s

# PostgreSQL
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=mydb
export DB_USER=user
export DB_PASSWORD=secret

# Redis
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

# JWT
export JWT_SECRET=your-secret-key-here
export JWT_ACCESS_DURATION=15m
export JWT_REFRESH_DURATION=168h

# Logging
export LOG_LEVEL=info
export LOG_ENCODING=json  # or "console"
```

## Build Process

### Local Development

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run the application
go run cmd/server/main.go
```

### Production Build

```bash
# Build for current platform
go build -o server cmd/server/main.go

# Build for Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o server-linux-amd64 cmd/server/main.go

# Build for macOS (ARM64/Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o server-darwin-arm64 cmd/server/main.go

# Build with optimizations
go build -ldflags="-s -w" -o server cmd/server/main.go
```

## Docker Deployment

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run application
CMD ["./server"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=mydb
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=secret
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

## Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: db-host
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: jwt-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

## Health Checks

Govern's healthcheck package integrates with Kubernetes probes:

```go
// Setup health checks
registry := healthcheck.New()

registry.Register("db", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

server.GET("/health", func(c echo.Context) error {
    resp := registry.Run(c.Request().Context())
    return c.JSON(http.StatusOK, resp)
})
```

## Monitoring

### Prometheus Metrics

```go
// Metrics endpoint
server.GET("/metrics", echo.WrapHandler(metrics.HandlerDefault()))

// Use metrics middleware
server.Use(metrics.HTTPMiddleware)
```

## Graceful Shutdown

Govern handles graceful shutdown automatically. The server will:

1. Stop accepting new connections
2. Wait for in-flight requests to complete
3. Run cleanup hooks (database, redis, etc.)
4. Exit cleanly

## Security Best Practices

### JWT Secret Management

```bash
# Generate secure secret
openssl rand -base64 32
```

### Database Credentials

Use secrets management:
- Kubernetes Secrets
- AWS Secrets Manager
- HashiCorp Vault
- Environment variables (never hardcode)

## Troubleshooting

### Common Issues

**1. Database connection refused**
- Check database is running
- Verify host/port configuration
- Check firewall rules

**2. High memory usage**
- Reduce connection pool sizes
- Check for goroutine leaks

**3. Slow requests**
- Check database query performance
- Verify network latency

### Debug Mode

```go
logger := log.New(
    log.WithLevel(zapcore.DebugLevel),
)
```

---

**Last Updated**: 2026-02-24
**Maintainer**: @haipham22
**Status**: Active Development
