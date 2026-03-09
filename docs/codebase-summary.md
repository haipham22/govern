# Codebase Summary

## Overview

The Govern codebase consists of **102 Go files** organized into **21 directories**, providing a comprehensive production-ready HTTP service library. The codebase follows clean architecture principles with clear separation of concerns and extensive test coverage.

## Project Structure

```
govern/
├── config/           # Configuration loading with YAML, .env, and ENV variables
├── cron/             # Cron scheduler with gocron v2, graceful lifecycle
├── database/         # Database connections (PostgreSQL + Redis)
│   ├── postgres/    # PostgreSQL with GORM
│   └── redis/       # Redis client with connection pooling
├── docs/             # Project documentation
├── errors/           # Structured error handling with error codes
├── graceful/         # Graceful shutdown for all services
├── healthcheck/      # Health check registry for liveness/readiness probes
├── http/             # HTTP server and middleware
│   ├── echo/        # Echo framework integration
│   ├── jwt/         # JWT authentication
│   └── middleware/  # HTTP middleware (logging, recovery, CORS, etc.)
├── log/              # Zap-based structured logging
├── metrics/          # Prometheus metrics
├── mq/               # Message queue
│   └── asynq/       # Asynq implementation
└── retry/            # Retry logic with backoff strategies
```

## Core Components Analysis

### 1. Configuration Module (`config/`)

**Files:** 8 files (including tests)
**Key Components:**
- `load.go`: Core configuration loading with Viper
- `load_env.go`: Environment variable handling with dot-to-underscore mapping
- `options.go`: Configuration options using builder pattern
- `config_test.go`, `example_test.go`: Comprehensive testing

**Architecture:**
```go
// Configuration priority: ENV vars > .env file > YAML values
type Config struct {
    Server     ServerConfig     `mapstructure:"server"`
    Database   DatabaseConfig   `mapstructure:"database"`
    Redis      RedisConfig      `mapstructure:"redis"`
    JWT        JWTConfig        `mapstructure:"jwt"`
    Log        LogConfig        `mapstructure:"log"`
    Metrics    MetricsConfig    `mapstructure:"metrics"`
    // ... other configurations
}
```

**Key Features:**
- Struct validation using go-playground/validator/v10
- Environment variable override capability
- YAML file loading with Viper
- Type-safe configuration with builder pattern

### 2. HTTP Server Module (`http/`)

**Files:** 22 files (including tests)
**Key Components:**
- `server.go`: Main HTTP server with graceful shutdown
- `echo/`: Echo framework utilities (JWT, Swagger, middleware)
- `jwt/`: JWT authentication with multiple extractors
- `middleware/`: Common HTTP middleware

**Architecture:**
```go
// Main server with graceful shutdown
type Server struct {
    echo        *echo.Echo
    httpServer  *http.Server
    graceful    *graceful.Manager
    // ... other fields
}

// JWT middleware with multiple extractor strategies
func JWTMiddleware(config JWTConfig) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Extract JWT from header, query, or cookie
            token := extractToken(c, config)
            // Validate and process JWT
            return next(c)
        }
    }
}
```

**Key Features:**
- Echo v4 framework integration
- Graceful shutdown with configurable timeouts
- JWT authentication with flexible extractors
- Middleware chain (logging, recovery, CORS, request ID, trimming)
- Swagger documentation support
- Configurable TLS and timeouts

### 3. Database Module (`database/`)

#### PostgreSQL (`database/postgres/`)

**Files:** 4 files
**Key Components:**
- `postgres.go`: GORM integration with connection pooling
- `postgres_test.go`: Connection testing

**Architecture:**
```go
// PostgreSQL factory with connection pooling
func NewPostgres(cfg DatabaseConfig) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

    return db, nil
}
```

**Features:**
- GORM v1.31.1 ORM integration
- Connection pooling: MaxOpenConns 100, MaxIdleConns 25, ConnMaxLifetime 5min
- Optimized settings: PreferSimpleProtocol, SkipDefaultTransaction
- Factory pattern with cleanup functions

#### Redis (`database/redis/`)

**Files:** 7 files
**Key Components:**
- `redis.go`: go-redis client with connection pooling
- `dsn.go`: DSN parsing and validation
- `options.go`: Configuration options

**Architecture:**
```go
// Redis universal client with connection pooling
func NewRedis(cfg RedisConfig) (*redis.Client, error) {
    return redis.NewClient(&redis.Options{
        Addr:         cfg.Addr,
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,
        MinIdleConns: cfg.MinIdleConns,
        // ... other options
    }), nil
}
```

**Features:**
- go-redis/v9 v9.17.3 UniversalClient
- Connection pooling: PoolSize 100, MinIdleConns 10
- DSN support: `redis://[:password@]host[:port][/db][?options]`
- Cluster and single-node support

### 4. Message Queue Module (`mq/asynq/`)

**Files:** 14 files
**Key Components:**
- `client.go`: Asynq client with task enqueueing
- `server.go`: Asynq server with graceful shutdown
- `mux.go`: Handler router with middleware
- `middleware.go`: Task processing middleware

**Architecture:**
```go
// Asynq client with task enqueueing
type Client struct {
    client    *asynq.Client
    config    Config
    // ... other fields
}

// Task enqueueing with priority, retry, scheduling
func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.Result, error) {
    return c.client.Enqueue(task, opts...)
}
```

**Features:**
- Asynq v0.26.0 with Redis broker
- Task enqueueing with priority, retry, scheduling, deduplication
- Server with graceful shutdown
- Handler router (TaskMux) with middleware chain
- JSON serialization via bytedance/sonic

### 5. Error Handling Module (`errors/`)

**Files:** 4 files
**Key Components:**
- `errors.go`: Structured error codes and ErrorWithCode struct

**Architecture:**
```go
// Structured error codes
type ErrorCode string

const (
    ErrorCodeInternal    ErrorCode = "INTERNAL"
    ErrorCodeInvalid     ErrorCode = "INVALID"
    ErrorCodeNotFound    ErrorCode = "NOT_FOUND"
    ErrorCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
    ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
    ErrorCodeForbidden   ErrorCode = "FORBIDDEN"
    ErrorCodeConflict   ErrorCode = "CONFLICT"
    ErrorCodeRateLimit  ErrorCode = "RATE_LIMIT"
)

// ErrorWithCode for wrapping errors with context
type ErrorWithCode struct {
    Code    ErrorCode
    Message string
    Err     error
}
```

**Features:**
- 8 predefined error codes for common scenarios
- ErrorWithCode struct for context-aware error wrapping
- Compatibility with Go standard errors and pkg/errors
- Structured error handling patterns

### 6. Graceful Shutdown Module (`graceful/`)

**Files:** 9 files
**Key Components:**
- `manager.go`: Three-phase shutdown manager
- `service.go`: Graceful service abstraction
- `runner.go`: Worker group management

**Architecture:**
```go
// Three-phase shutdown manager
type Manager struct {
    timeout     time.Duration
    failFast    bool
    cancel      context.CancelFunc
    shutdownWG  sync.WaitGroup
    cleanupHooks []CleanupHook
    // ... other fields
}

// Three-phase lifecycle
func (m *Manager) Shutdown() {
    // Phase 1: Cancel context
    // Phase 2: Wait for goroutines
    // Phase 3: Cleanup hooks (LIFO)
}
```

**Features:**
- Three-phase shutdown: cancel context → wait for goroutines → cleanup hooks (LIFO)
- Signal-aware (SIGINT, SIGTERM)
- Fail-fast mode for goroutine errors
- Configurable timeout and cleanup hooks

### 7. Health Check Module (`healthcheck/`)

**Files:** 6 files
**Key Components:**
- `registry.go`: Health check registry
- `handler.go`: HTTP handlers for probes
- `types.go`: Health check types and status

**Architecture:**
```go
// Health check registry
type Registry struct {
    checks    map[string]CheckFunc
    mutex     sync.RWMutex
    // ... other fields
}

// Health check with status levels
type HealthCheck struct {
    Name     string
    Status   StatusLevel
    Message  string
    Duration time.Duration
    // ... other fields
}
```

**Features:**
- Status levels: passing, failing, warning
- Concurrent health check execution
- Timeout-based with panic recovery
- HTTP handlers for liveness/readiness probes

### 8. Logging Module (`log/`)

**Files:** 5 files
**Key Components:**
- `logger.go`: Zap-based global logger
- `config.go`: Logging configuration
- `helpers.go`: Logging utilities

**Architecture:**
```go
// Global Zap logger with thread safety
var (
    logger     *zap.Logger
    loggerOnce sync.Once
)

// Configurable logging levels and outputs
type Config struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"` // json or console
    Output string `mapstructure:"output"` // stdout, stderr, or file path
}
```

**Features:**
- Zap-based structured logging
- Configurable levels, encoding (JSON/console), output
- Global logger with thread safety
- Colorized console output for development

### 9. Metrics Module (`metrics/`)

**Files:** 9 files
**Key Components:**
- `registry.go`: Metrics registry
- `middleware.go`: Prometheus HTTP middleware
- `metrics-types.go`: Custom metric types

**Architecture:**
```go
// Prometheus metrics with HTTP middleware
type Middleware struct {
    registry *prometheus.Registry
    // ... other fields
}

// Standard metrics collection
type Metrics struct {
    HTTPRequestsTotal      *prometheus.CounterVec
    HTTPRequestDuration    *prometheus.HistogramVec
    HTTPResponseSize       *prometheus.HistogramVec
    // ... other metrics
}
```

**Features:**
- Prometheus client_golang v1.23.2
- HTTP middleware: http_requests_total, http_request_duration_seconds, http_response_size_bytes
- Counter, Gauge, Histogram, Summary support
- Custom label support

### 10. Cron Scheduler Module (`cron/`)

**Files:** 10 files
**Key Components:**
- `scheduler.go`: Main scheduler with gocron v2
- `handler.go`: Handler interface for jobs
- `session.go`: Session interface for job lifecycle

**Architecture:**
```go
// Three-phase job lifecycle
type Handler interface {
    Setup(ctx context.Context) error
    Execute(ctx context.Context, job Job) error
    Cleanup(ctx context.Context) error
}

// Scheduler with graceful shutdown
type Scheduler struct {
    scheduler *gocron.Scheduler
    handler   Handler
    session   Session
    // ... other fields
}
```

**Features:**
- gocron v2 with graceful shutdown
- Three-phase job lifecycle: Setup → Execute → Cleanup
- Inspired by sarama.ConsumerGroupHandler pattern
- Duration-based scheduling

### 11. Retry Module (`retry/`)

**Files:** 5 files
**Key Components:**
- `policy.go`: Retry policy configuration
- `backoff.go`: Backoff strategy implementations
- `predicate.go`: Retry predicate logic

**Architecture:**
```go
// Retry policy with backoff strategies
type Policy struct {
    MaxAttempts     int           `json:"max_attempts"`
    MaxDuration     time.Duration `json:"max_duration"`
    InitialDelay   time.Duration `json:"initial_delay"`
    MaxDelay       time.Duration `json:"max_delay"`
    Multiplier     float64       `json:"multiplier"`
    Jitter         float64       `json:"jitter"`
    Strategy       Strategy      `json:"strategy"`
    // ... other fields
}

// Exponential backoff with jitter
func Exponential(delay, maxDelay time.Duration, multiplier, jitter float64) BackoffFunc {
    return func(attempt int, err error) time.Duration {
        // Exponential backoff calculation with jitter
    }
}
```

**Features:**
- Default: 3 attempts, 1 minute max duration
- Backoff strategies: exponential (default), linear, constant
- Exponential: base 100ms, max 30s, 2x multiplier, 25% jitter
- Retry predicates for error-specific logic

## Technology Stack

### Core Dependencies
- **Go 1.25.8**: Downgraded from 1.26 due to golangci-lint compatibility
- **Echo v4**: High-performance HTTP framework
- **GORM v1.31.1**: PostgreSQL ORM (maintained for stability)
- **go-redis/v9**: Redis client with excellent performance
- **Asynq v0.26.0**: Redis-based message queue
- **Zap**: High-performance structured logging
- **Prometheus**: Metrics collection and monitoring
- **gocron v2**: Cron scheduler with graceful lifecycle
- **Viper**: Configuration management
- **JWT golang-jwt/jwt/v5**: Authentication

### Development Dependencies
- **Go testing**: Standard library testing framework
- **golangci-lint**: Code quality and linting (v2)
- **testify**: Testing utilities
- **mock**: Mock generation for testing

## Design Patterns Implementation

### 1. Builder Pattern
Used extensively for configuration options:
```go
// Configuration with builder pattern
type ServerConfig struct {
    Port         int    `mapstructure:"port"`
    ReadTimeout  int    `mapstructure:"read_timeout"`
    WriteTimeout int    `mapstructure:"write_timeout"`
    TLS          TLSConfig `mapstructure:"tls"`
}

// Builder-style configuration
func WithPort(port int) Option {
    return func(c *Config) {
        c.Server.Port = port
    }
}
```

### 2. Observer Pattern
Middleware chain implementation:
```go
// Middleware chain as observer
func Chain(middlewares ...echo.MiddlewareFunc) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}
```

### 3. Strategy Pattern
JWT extractors and CORS strategies:
```go
// JWT extractor strategy
type TokenExtractor func(c echo.Context) (string, error)

// Extract from header
func FromHeader(echo.Header) TokenExtractor {
    return func(c echo.Context) (string, error) {
        return c.Request().Header.Get(echo.HeaderAuthorization), nil
    }
}

// Extract from query parameter
func FromQuery(param string) TokenExtractor {
    return func(c echo.Context) (string, error) {
        return c.QueryParam(param), nil
    }
}
```

### 4. Chain of Responsibility
Middleware pipeline:
```go
// Chain of responsibility middleware
func LoggingMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Log request
            logger.Info("Request received", zap.String("method", c.Request().Method))

            // Call next middleware
            err := next(c)

            // Log response
            logger.Info("Request processed", zap.Error(err))

            return err
        }
    }
}
```

### 5. Repository Pattern
Health check registry:
```go
// Repository pattern for health checks
type Registry struct {
    checks map[string]CheckFunc
    mutex  sync.RWMutex
}

// Register health check
func (r *Registry) Register(name string, check CheckFunc) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    r.checks[name] = check
}
```

### 6. Factory Pattern
Database connections:
```go
// Factory pattern for database connections
func NewPostgres(cfg DatabaseConfig) (*gorm.DB, error) {
    // Create and configure database connection
    return db, nil
}

// Factory pattern for Redis connections
func NewRedis(cfg RedisConfig) (*redis.Client, error) {
    // Create and configure Redis connection
    return client, nil
}
```

### 7. Producer-Consumer
Message queue implementation:
```go
// Producer: Task enqueueing
func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.Result, error) {
    return c.client.Enqueue(task, opts...)
}

// Consumer: Task processing
func (s *Server) Start() {
    s.mux.HandleFunc(taskType, s.processTask)
    s.server.Run(s.client)
}
```

## Test Coverage Analysis

### Test Distribution by Module
- **HTTP Server**: Comprehensive test coverage with integration tests
- **Database**: Connection testing and configuration validation
- **Message Queue**: Server and client functionality tests
- **Authentication**: JWT middleware and validation tests
- **Configuration**: Loading and validation tests
- **Graceful Shutdown**: Signal handling and cleanup tests
- **Health Check**: Registry and handler tests
- **Metrics**: Middleware and registry tests

### Test Quality Standards
- Unit tests for all public APIs
- Integration tests for component interactions
- Mock generation for external dependencies
- Test helpers and utilities for consistent testing
- Benchmark tests for performance-critical components

### Code Quality Metrics
- **Go Report Card**: A+ grade maintained
- **Linting**: golangci-lint v2 with comprehensive rules
- **Formatting**: Standard Go formatting with gofmt
- **Vetting**: go vet for static analysis

## Performance Characteristics

### Memory Efficiency
- Connection pooling reduces memory footprint
- JSON serialization with sonic for better performance
- Minimal object allocation in hot paths
- Configurable buffers and timeouts

### Concurrency Design
- Goroutine pools with proper lifecycle management
- Concurrent health check execution
- Thread-safe metrics collection
- Context-based cancellation throughout

### Optimization Highlights
- Connection pooling for database and Redis
- Efficient JSON serialization with sonic
- Minimal overhead middleware design
- Configurable timeouts for all network operations

## Security Considerations

### Authentication & Authorization
- JWT authentication with configurable policies
- Multiple token extraction strategies
- Configurable CORS protection
- Input validation and sanitization

### Security Best Practices
- Secure default configurations
- Proper error handling to avoid information leakage
- HTTPS/TLS support for production
- Secure password handling in Redis connections

## Integration Points

### Component Interactions
```
HTTP Server → Database (PostgreSQL + Redis)
HTTP Server → Message Queue (Asynq)
HTTP Server → Scheduler (gocron)
All Components → Graceful Shutdown
All Components → Logging (Zap)
All Components → Metrics (Prometheus)
```

### External Integration Points
- **PostgreSQL**: Primary database for persistent data
- **Redis**: Caching, session storage, message broker
- **Prometheus**: Metrics collection and monitoring
- **Health Check Endpoints**: Kubernetes and container orchestration

## Documentation Quality

### Documentation Coverage
- 100% of public APIs documented
- Comprehensive examples for all components
- Architecture diagrams and flowcharts
- Best practices and usage guidelines

### Documentation Types
- README files in each module
- Inline code documentation
- Architecture and design documentation
- Quick start guides and tutorials

## Maintainability Features

### Code Organization
- Clear module boundaries
- Consistent naming conventions
- Comprehensive error handling
- Extensive test coverage

### Extensibility
- Interface-based design for all major components
- Plugin architecture for middleware
- Configuration-driven behavior
- Easy integration with additional services

### Operational Excellence
- Comprehensive observability
- Graceful shutdown for all components
- Health check endpoints
- Configurable logging and metrics

## Future Extensibility

### Planned Enhancements
- Circuit breaker pattern
- Additional framework integrations (gin, fiber)
- Advanced monitoring and tracing
- Distributed transaction support

### Scaling Considerations
- Horizontal scaling support
- Load balancing integration
- Multi-tenancy patterns
- Microservices architecture compatibility

---

*This summary covers the complete codebase architecture, components, and key implementation details. The codebase follows Go best practices with comprehensive testing, documentation, and clear separation of concerns.*
