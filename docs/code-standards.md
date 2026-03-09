# Code Standards and Conventions

## Overview

This document outlines the coding standards and conventions used throughout the Govern project. These standards ensure code quality, consistency, maintainability, and performance across the entire codebase.

## Table of Contents

1. [General Guidelines](#general-guidelines)
2. [Project Structure](#project-structure)
3. [Naming Conventions](#naming-conventions)
4. [Code Organization](#code-organization)
5. [Error Handling](#error-handling)
6. [Testing Standards](#testing-standards)
7. [Documentation Standards](#documentation-standards)
8. [Performance Considerations](#performance-considerations)
9. [Security Practices](#security-practices)
10. [Go Best Practices](#go-best-practices)
11. [Tooling and Linting](#tooling-and-linting)

## General Guidelines

### 1. Code Quality Principles

- **Readability**: Code should be easy to understand and maintain
- **Consistency**: Follow established patterns and conventions
- **Simplicity**: Prefer simple, clear solutions over complex ones
- **Performance**: Optimize for production performance while maintaining readability
- **Testability**: Design code to be easily testable and mockable

### 2. Development Philosophy

- **Minimal Dependencies**: Only include dependencies that provide clear value
- **Standard Library First**: Prefer Go standard library over third-party packages when possible
- **Interface Segregation**: Create focused, single-purpose interfaces
- **Composition Over Inheritance**: Use composition and interfaces for code reuse
- **Fail Fast**: Validate inputs and fail early when possible

## Project Structure

### 1. Directory Organization

```
govern/
├── config/           # Configuration management
├── database/         # Database connections (postgres/, redis/)
├── cron/             # Scheduling and background jobs
├── errors/           # Error handling and codes
├── graceful/         # Graceful shutdown utilities
├── healthcheck/      # Health monitoring
├── http/             # HTTP server and middleware
├── log/              # Logging utilities
├── metrics/          # Metrics collection
├── mq/               # Message queue
└── retry/            # Retry mechanisms
```

### 2. File Naming Conventions

- **Package files**: Use lowercase names (e.g., `server.go`, `logger.go`)
- **Interface files**: Name after the interface (e.g., `handler.go`, `session.go`)
- **Configuration files**: Use `config.go` or specific name (e.g., `options.go`)
- **Test files**: Append `_test.go` to the source file name
- **Example files**: Append `_example.go` to the source file name
- **Documentation files**: Use `README.md` in each module

### 3. Package Organization

- **Single Responsibility**: Each package should have a single, clear purpose
- **Loose Coupling**: Minimize dependencies between packages
- **High Cohesion**: Related functionality should be in the same package
- **Public/Private APIs**: Use capitalization to control API visibility

## Naming Conventions

### 1. Go Naming Standards

#### Package Names
- Use lowercase, descriptive names
- Avoid underscores and mixed case
- Keep names concise but meaningful
- Example: `http`, `database`, `config`

#### Variable and Function Names
- Use `camelCase` for variables and functions
- Use descriptive names that explain the purpose
- Avoid single-letter variables except in loops
- Example: `httpClient`, `databaseConfig`, `startServer`

#### Type Names
- Use `PascalCase` for types, structs, and interfaces
- Use descriptive names that indicate the purpose
- Example: `HTTPServer`, `DatabaseConfig`, `JWTMiddleware`

#### Constants
- Use `PascalCase` for exported constants
- Use `camelCase` for unexported constants
- Use consistent naming for related constants
- Example: `MaxRetries`, `DefaultTimeout`, `ErrorCodeNotFound`

#### Interface Names
- Use `-er` suffix for simple interfaces
- Use descriptive names for complex interfaces
- Example: `Handler`, `Session`, `Logger`

### 2. Specific Naming Patterns

#### Configuration Structures
```go
// Use descriptive field names with clear types
type ServerConfig struct {
    Port         int           `mapstructure:"port" validate:"required,min=1,max=65535"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout" validate:"required,min=1s"`
    WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"required,min=1s"`
    TLS          TLSConfig     `mapstructure:"tls"`
}
```

#### Error Types
```go
// Use consistent error naming with Error suffix
type ValidationError struct {
    Field   string
    Message string
}

type AuthenticationError struct {
    Reason string
}
```

#### Middleware Functions
```go
// Use descriptive middleware names
func LoggingMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Middleware logic
            return next(c)
        }
    }
}
```

## Code Organization

### 1. File Structure

Each Go file should follow this structure:

```go
// Package declaration
package [package_name]

// Imports
import (
    "context"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/go-redis/redis/v9"

    "github.com/haipham22/govern/errors"
)

// Constants
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

// Interfaces
type InterfaceName interface {
    MethodName() error
}

// Struct definitions
type StructName struct {
    Field1 type1
    Field2 type2
}

// Public functions
func FunctionName() error {
    // Implementation
}

// Private functions
func privateFunction() error {
    // Implementation
}

// Main package initialization
func init() {
    // Package initialization
}
```

### 2. Import Organization

#### Standard Library First
```go
import (
    "context"
    "fmt"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/go-redis/redis/v9"

    "github.com/haipham22/govern/config"
    "github.com/haipham22/govern/http"
)
```

#### Group Related Imports
```go
import (
    // Standard library
    "context"
    "time"
    "sync"

    // Third-party packages
    "github.com/labstack/echo/v4"
    "github.com/go-redis/redis/v9"
    "github.com/asynq/asynq/v4"

    // Local packages
    "github.com/haipham22/govern/config"
    "github.com/haipham22/govern/database"
    "github.com/haipham22/govern/errors"
)
```

#### Avoid Wildcard Imports
```go
// Bad
import . "github.com/labstack/echo/v4"

// Good
import "github.com/labstack/echo/v4"
```

### 3. Function Organization

#### Public Functions First
```go
// Package starts with public functions
func NewServer(config config.ServerConfig) (*Server, error) {
    // Implementation
}

func (s *Server) Start() error {
    // Implementation
}

// Private functions follow
func (s *Server) setupRoutes() {
    // Implementation
}

func (s *Server) shutdown() error {
    // Implementation
}
```

#### Function Length
- Keep functions under 50 lines when possible
- Break complex functions into smaller, focused functions
- Use early returns for error handling
- Maintain a single responsibility per function

### 4. Struct Organization

#### Field Organization
```go
type Server struct {
    // Configuration fields first
    config     config.ServerConfig
    logger     *zap.Logger

    // Core components
    echo       *echo.Echo
    httpServer *http.Server

    // Dependencies
    database   *gorm.DB
    redis      *redis.Client

    // State
    ctx        context.Context
    cancel     context.CancelFunc

    // Synchronization
    mutex      sync.RWMutex
}
```

#### Interface Implementation
```go
// Implement interfaces explicitly
func (h *Handler) Setup(ctx context.Context) error {
    // Implementation
}

func (h *Handler) Execute(ctx context.Context, job Job) error {
    // Implementation
}

func (h *Handler) Cleanup(ctx context.Context) error {
    // Implementation
}
```

## Error Handling

### 1. Error Code Standards

Use the predefined error codes from the `errors` package:

```go
import "github.com/haipham22/govern/errors"

// Predefined error codes
errors.ErrorCodeInternal      // For unexpected internal errors
errors.ErrorCodeInvalid       // For invalid input or configuration
errors.ErrorCodeNotFound      // For missing resources
errors.ErrorCodeAlreadyExists // For duplicate resources
errors.ErrorCodeUnauthorized   // For authentication failures
errors.ErrorCodeForbidden     // For authorization failures
errors.ErrorCodeConflict     // For conflicting operations
errors.ErrorCodeRateLimit    // For rate limiting
```

### 2. Error Handling Patterns

#### Error Wrapping
```go
// Use ErrorWithCode for context-aware error wrapping
func validateConfig(config Config) error {
    if config.Port == 0 {
        return errors.NewErrorWithCode(
            errors.ErrorCodeInvalid,
            "port is required",
            fmt.Errorf("port validation failed"),
        )
    }
    return nil
}
```

#### Error Handling in Functions
```go
func startServer(config config.ServerConfig) (*Server, error) {
    // Validate input
    if err := validateConfig(config); err != nil {
        return nil, err
    }

    // Create dependencies
    logger, err := log.New(config.Log)
    if err != nil {
        return nil, errors.NewErrorWithCode(
            errors.ErrorCodeInternal,
            "failed to create logger",
            err,
        )
    }

    // Initialize server
    server := &Server{
        config:  config,
        logger:  logger,
    }

    return server, nil
}
```

#### Error Logging
```go
func (s *Server) handleRequest(c echo.Context) error {
    defer func() {
        if r := recover(); r != nil {
            s.logger.Error("Recovered from panic",
                zap.Any("panic", r),
                zap.Stack("stack"),
            )
            c.Error(errors.NewErrorWithCode(
                errors.ErrorCodeInternal,
                "internal server error",
                fmt.Errorf("panic recovered: %v", r),
            ))
        }
    }()

    // Handle request
    return nil
}
```

### 3. Error Best Practices

- **Don't ignore errors**: Always handle errors appropriately
- **Provide context**: Wrap errors with meaningful context
- **Use appropriate error codes**: Choose the right error code for the situation
- **Log errors appropriately**: Use structured logging with context
- **Return errors, not panic**: Use panic only for unrecoverable errors

## Testing Standards

### 1. Test File Organization

```go
// Package with tests
package server_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/haipham22/govern/config"
    "github.com/haipham22/govern/http"
)

// Test functions with clear naming
func TestNewServer_ValidConfig(t *testing.T) {
    // Test implementation
}

func TestServer_Start_InvalidPort(t *testing.T) {
    // Test implementation
}

func TestServer_Shutdown_Graceful(t *testing.T) {
    // Test implementation
}
```

### 2. Testing Patterns

#### Unit Testing
```go
func TestJWTMiddleware_ValidToken(t *testing.T) {
    // Setup
    config := http.JWTConfig{
        Secret:     "test-secret",
        Algorithm:  "HS256",
        Extractor:  http.FromHeader(echo.HeaderAuthorization),
    }

    middleware := http.JWTMiddleware(config)
    echoServer := echo.New()
    echoServer.Use(middleware)

    // Test implementation
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")

    resp := httptest.NewRecorder()

    // Verify
    assert.Equal(t, http.StatusOK, resp.Code)
}
```

#### Integration Testing
```go
func TestServer_Integration_FullLifecycle(t *testing.T) {
    // Integration test for complete server lifecycle
    config := config.ServerConfig{
        Port: 8080,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    server, err := http.NewServer(config)
    require.NoError(t, err)

    // Start server
    go func() {
        err := server.Start()
        assert.NoError(t, err)
    }()

    // Wait for server to start
    time.Sleep(100 * time.Millisecond)

    // Test endpoints
    // ...

    // Shutdown server
    err = server.Shutdown()
    assert.NoError(t, err)
}
```

#### Test Utilities
```go
// Common test helpers
func createTestConfig() config.Config {
    return config.Config{
        Server: config.ServerConfig{
            Port:         8080,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
        },
        Database: config.DatabaseConfig{
            Host:     "localhost",
            Port:     5432,
            User:     "test",
            Password: "test",
            Database: "test",
        },
    }
}

func createTestServer() (*http.Server, error) {
    config := createTestConfig()
    return http.NewServer(config.Server)
}
```

### 3. Testing Best Practices

- **Use testify/assert and testify/require**: For better test assertions
- **Test both success and failure cases**: Ensure comprehensive coverage
- **Mock external dependencies**: Use interfaces for testability
- **Test edge cases**: Handle boundary conditions and error cases
- **Write integration tests**: Test component interactions
- **Benchmark critical functions**: Use testing.B for performance testing

## Documentation Standards

### 1. Code Documentation

#### Package Documentation
```go
// Package http provides HTTP server functionality for Govern
//
// This package includes:
//   - HTTP server with graceful shutdown
//   - Echo framework integration
//   - JWT authentication middleware
//   - Common HTTP middleware (logging, recovery, CORS)
//   - Swagger documentation support
package http
```

#### Function Documentation
```go
// NewServer creates a new HTTP server with the given configuration
//
// Parameters:
//   - config: Server configuration including port, timeouts, and TLS settings
//
// Returns:
//   - *Server: Configured HTTP server instance
//   - error: If configuration is invalid or server initialization fails
//
// Example:
//   config := config.ServerConfig{Port: 8080}
//   server, err := NewServer(config)
//   if err != nil {
//       log.Fatal(err)
//   }
func NewServer(config config.ServerConfig) (*Server, error)
```

#### Struct Documentation
```go
// Server represents a HTTP server with graceful shutdown capabilities
//
// The server includes:
//   - Echo web framework integration
//   - Graceful shutdown handling
//   - Middleware chain support
//   - Configuration management
//   - Logging and metrics integration
type Server struct {
    config     config.ServerConfig  // Server configuration
    logger     *zap.Logger          // Structured logger
    echo       *echo.Echo           // Echo web framework
    httpServer *http.Server         // HTTP server instance
    graceful   *graceful.Manager    // Graceful shutdown manager
}
```

### 2. Example Code

#### Usage Examples
```go
// Example of server configuration and startup
func ExampleServer_Usage() {
    // Configure server
    config := config.ServerConfig{
        Port:         8080,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        TLS: config.TLSConfig{
            Enabled:  false,
            CertFile: "",
            KeyFile:  "",
        },
    }

    // Create and start server
    server, err := NewServer(config)
    if err != nil {
        log.Fatal(err)
    }

    // Add routes
    server.echo.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{
            "status": "ok",
        })
    })

    // Start server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }

    // Graceful shutdown
    server.Shutdown()
}
```

### 3. Documentation Best Practices

- **Keep documentation up to date**: Update when code changes
- **Provide clear examples**: Include working examples for complex features
- **Document error cases**: Document error conditions and handling
- **Use consistent formatting**: Follow Go documentation conventions
- **Include performance notes**: Document performance characteristics
- **Document thread safety**: Note when types are safe for concurrent use

## Performance Considerations

### 1. Memory Management

#### Connection Pooling
```go
// Configure connection pooling for optimal performance
func createDatabaseConnection(config config.DatabaseConfig) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })

    if err != nil {
        return nil, err
    }

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(100)      // Maximum open connections
    sqlDB.SetMaxIdleConns(25)      // Maximum idle connections
    sqlDB.SetConnMaxLifetime(5 * time.Minute) // Connection lifetime

    return db, nil
}
```

#### Object Allocation
```go
// Minimize object allocation in hot paths
func (s *Server) handleRequest(c echo.Context) error {
    // Reuse buffers and objects where possible
    // Avoid allocations in request handling
    return c.JSON(http.StatusOK, response)
}
```

### 2. Concurrency Patterns

#### Goroutine Management
```go
// Use worker pools for concurrent operations
func (s *Server) startWorkerPool() {
    s.workerWg.Add(1)
    go func() {
        defer s.workerWg.Done()

        // Worker pool with fixed size
        workerCount := 10
        jobs := make(chan Job, 100)

        for i := 0; i < workerCount; i++ {
            go s.worker(jobs)
        }

        // Process jobs
        for job := range jobs {
            s.processJob(job)
        }
    }()
}
```

#### Channel Usage
```go
// Use buffered channels for performance
func (s *Server) startMetricsCollector() {
    metricsChan := make(chan Metric, 100) // Buffered channel

    go func() {
        for metric := range metricsChan {
            s.processMetric(metric)
        }
    }()

    // Send metrics to channel
    metricsChan <- Metric{Name: "requests", Value: 1}
}
```

### 3. Performance Optimization

#### Efficient JSON Processing
```go
// Use sonic for faster JSON serialization
import "github.com/bytedance/sonic"

// Fast JSON encoding
func encodeJSON(v interface{}) ([]byte, error) {
    return sonic.Marshal(v)
}

// Fast JSON decoding
func decodeJSON(data []byte, v interface{}) error {
    return sonic.Unmarshal(data, v)
}
```

#### Proper Time Handling
```go
// Use time.Time for better performance than string timestamps
func (s *Server) logRequest(c echo.Context, start time.Time) {
    duration := time.Since(start)
    s.logger.Info("Request processed",
        zap.String("method", c.Request().Method),
        zap.String("path", c.Request().URL.Path),
        zap.Duration("duration", duration),
    )
}
```

## Security Practices

### 1. Input Validation

#### Struct Validation
```go
import "github.com/go-playground/validator/v10"

type UserRequest struct {
    Username string `json:"username" validate:"required,min=3,max=30"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

func validateUserRequest(req UserRequest) error {
    validate := validator.New()
    return validate.Struct(req)
}
```

#### Sanitization
```go
import "html"

func sanitizeInput(input string) string {
    // Prevent XSS attacks by escaping HTML
    return html.EscapeString(input)
}

func validateInput(input string) error {
    // Check for SQL injection patterns
    if strings.Contains(input, "'") || strings.Contains(input, ";") {
        return errors.NewErrorWithCode(
            errors.ErrorCodeInvalid,
            "invalid input characters",
            fmt.Errorf("input contains invalid characters"),
        )
    }
    return nil
}
```

### 2. Security Configuration

#### JWT Security
```go
type JWTConfig struct {
    Secret        string `json:"secret" validate:"required"`
    Algorithm     string `json:"algorithm" validate:"one_of=HS256 RS256"`
    Expiry        time.Duration `json:"expiry" validate:"required"`
    RefreshExpiry time.Duration `json:"refresh_expiry"`
    Issuer        string `json:"issuer"`
    Audience      string `json:"audience"`
}
```

#### TLS Configuration
```go
type TLSConfig struct {
    Enabled  bool   `json:"enabled"`
    CertFile string `json:"cert_file"`
    KeyFile  string `json:"key_file"`

    // Validate TLS configuration
    func (c TLSConfig) Validate() error {
        if c.Enabled {
            if c.CertFile == "" || c.KeyFile == "" {
                return errors.NewErrorWithCode(
                    errors.ErrorCodeInvalid,
                    "TLS enabled but certificate files not specified",
                    nil,
                )
            }
        }
        return nil
    }
}
```

### 3. Security Best Practices

- **Use HTTPS**: Always use TLS for production deployments
- **Validate input**: Validate all external inputs
- **Sanitize output**: Escape output to prevent injection attacks
- **Secure configuration**: Use secure defaults and configuration validation
- **Handle secrets securely**: Use environment variables for sensitive data
- **Regular security updates**: Keep dependencies updated

## Go Best Practices

### 1. Error Handling

#### Early Returns
```go
func processRequest(c echo.Context) error {
    // Validate input
    if err := validateRequest(c); err != nil {
        return err
    }

    // Process request
    data, err := processData(c)
    if err != nil {
        return err
    }

    // Return response
    return c.JSON(http.StatusOK, data)
}
```

#### Error Wrapping
```go
func externalCall() error {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return fmt.Errorf("failed to fetch data: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return nil
}
```

### 2. Context Usage

#### Context Propagation
```go
func (s *Server) handleRequest(c echo.Context) error {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
    defer cancel()

    // Propagate context through the call chain
    result, err := s.service.Process(ctx, c.Request())
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, result)
}
```

#### Context Values
```go
type contextKey string

const (
    userKey contextKey = "user"
    requestIDKey contextKey = "request_id"
)

// Set user in context
func WithUser(ctx context.Context, user User) context.Context {
    return context.WithValue(ctx, userKey, user)
}

// Get user from context
func UserFromContext(ctx context.Context) (User, bool) {
    user, ok := ctx.Value(userKey).(User)
    return user, ok
}
```

### 3. Interface Usage

#### Interface Segregation
```go
// Small, focused interfaces
type Reader interface {
    Read() ([]byte, error)
}

type Writer interface {
    Write([]byte) error
}

// Larger interfaces for complex operations
type Processor interface {
    Process(ctx context.Context, input []byte) ([]byte, error)
    Cleanup(ctx context.Context) error
}
```

#### Interface Implementation
```go
// Explicit interface implementation
type FileProcessor struct{}

func (f *FileProcessor) Process(ctx context.Context, input []byte) ([]byte, error) {
    // Implementation
    return input, nil
}

func (f *FileProcessor) Cleanup(ctx context.Context) error {
    // Implementation
    return nil
}
```

## Tooling and Linting

### 1. Code Quality Tools

#### golangci-lint Configuration
```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - govet
    - golint
    - staticcheck
    - ineffassign
    - misspell
    - structcheck
    - varcheck
    - unconvert
    - unparam
    - deadcode
    - interfacer
    - testpackage
    - goconst
    - gocyclo
    - maligned
    - scopelint
    - gosec
    - gosimple
    - typecheck
    - unused
    - gocritic

linters-settings:
  gocyclo:
    min-complexity: 15
  gosec:
    # Excludes specific rules
    excludes:
      - G104
  goconst:
    min-len: 3
    min-occurrences: 3
```

### 2. Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint
        args: [--timeout=5m]
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-merge-conflict
```

### 3. Build and Test Commands

#### Makefile
```makefile
.PHONY: test lint build clean

# Run tests
test:
	go test -v -race -timeout=30s ./...

# Run linting
lint:
	golangci-lint run --timeout=5m

# Build application
build:
	go build -o bin/govern ./cmd/govern

# Clean build artifacts
clean:
	rm -rf bin/
	go clean -cache

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all checks
check: lint test build
```

### 4. Continuous Integration

#### GitHub Actions Workflow
```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.23'
      - run: make deps
      - run: make lint
      - run: make test
      - run: make build

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.23'
      - run: go install github.com/securecodewarrior/go-analysis@latest
      - run: go-analysis -enable-all
```

## Summary

These coding standards ensure that the Govern project maintains high code quality, consistency, and performance across the entire codebase. By following these conventions, we can:

- Ensure maintainability and readability of the codebase
- Provide consistent APIs and interfaces
- Maintain high test coverage and code quality
- Follow Go best practices and community standards
- Ensure security and performance considerations are addressed
- Enable efficient collaboration among developers

All contributors should follow these standards when working on the Govern project.

---

*Last Updated: March 9, 2026*
*Version: 1.0.0*