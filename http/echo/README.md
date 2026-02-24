# Echo Framework Integration

The `echo` package provides integration between [Echo](https://echo.labstack.com/) framework and Govern's HTTP server, combining Echo's powerful routing with Govern's graceful shutdown and server management.

## Features

- **Echo Server Wrapper**: Full Echo API access through `Server.Echo()`
- **Graceful Shutdown**: Integrated with Govern's graceful shutdown mechanism
- **JWT Authentication**: Echo-specific JWT middleware using `http/jwt`
- **Route Methods**: All HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
- **Route Groups**: Support for route grouping with middleware
- **Static Files**: Built-in static file serving
- **Handler Wrapping**: Reuse standard `http.Handler` with Echo via `WrapHandler()`

## Installation

```bash
go get github.com/haipham22/govern/http/echo
```

## Quick Start

```go
package main

import (
    httpEcho "github.com/haipham22/govern/http/echo"
    "github.com/labstack/echo/v4"
    "net/http"
)

func main() {
    server := httpEcho.NewServer(":8080")

    server.GET("/", func(c echo.Context) error {
        return c.String(http.StatusOK, "Hello, World!")
    })

    server.POST("/users", func(c echo.Context) error {
        return c.JSON(http.StatusCreated, map[string]string{
            "message": "User created",
        })
    })

    server.Start()
}
```

## JWT Authentication

```go
package main

import (
    httpEcho "github.com/haipham22/govern/http/echo"
    "github.com/haipham22/govern/http/jwt"
    "github.com/labstack/echo/v4"
    "net/http"
)

func main() {
    server := httpEcho.NewServer(":8080")

    // Public routes
    server.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    // JWT configuration
    jwtConfig := &jwt.MiddlewareConfig{
        Config:         jwt.DefaultConfig(),
        TokenExtractor: jwt.DefaultTokenExtractor,
        SkipPaths:      []string{"/health", "/login"},
    }
    jwtConfig.Config.Secret = "your-secret-key"

    // Protected routes
    api := server.Group("/api", httpEcho.WithJWT(jwtConfig))
    api.GET("/profile", func(c echo.Context) error {
        claims := httpEcho.MustGetCurrentUser(c)
        return c.JSON(http.StatusOK, map[string]interface{}{
            "user_id":  claims.UserID,
            "username": claims.Username,
        })
    })

    server.Start()
}
```

## Middleware

### IMPORTANT: Do NOT Mix Middleware Types

Echo middleware (`echo.MiddlewareFunc`) and Govern middleware (`http.Middleware`) are **not compatible** due to:

1. Different error handling flows (Echo returns errors, http middleware doesn't)
2. Different context types (`echo.Context` vs `http.Request`)
3. Response writer differences (`echo.Response` vs `http.ResponseWriter`)
4. Complex conversion introduces bugs and performance overhead

**Instead:**
- Use Echo middleware for Echo routes
- Use Govern middleware for `http.Handler` routes
- Use `WrapHandler()` to reuse `http.Handler` with Echo if needed

### Using Echo Middleware

```go
import (
    "github.com/labstack/echo/v4/middleware"
    httpEcho "github.com/haipham22/govern/http/echo"
)

server := httpEcho.NewServer(":8080")

// Use Echo's built-in middleware
server.Use(middleware.Logger())
server.Use(middleware.Recover())
server.Use(middleware.CORS())
```

### Wrapping http.Handler

```go
import (
    "net/http"
    httpEcho "github.com/haipham22/govern/http/echo"
)

// Reuse standard http.Handler with Echo
var myHandler http.Handler = // ...

server.GET("/legacy", httpEcho.WrapHandler(myHandler))
```

## Configuration

### Server Options

```go
import (
    "time"
    httpEcho "github.com/haipham22/govern/http/echo"
    "github.com/haipham22/govern/log"
)

server := httpEcho.NewServer(":8080",
    httpEcho.WithTimeout(5*time.Second, 10*time.Second, 60*time.Second),
    httpEcho.WithLogger(log.Default()),
)
```

## Context Helpers

### Getting Current User

```go
func handler(c echo.Context) error {
    // Returns claims and true if found, nil and false otherwise
    claims, ok := httpEcho.GetCurrentUser(c)
    if !ok {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    // Panics if user not found (use after JWT middleware)
    claims := httpEcho.MustGetCurrentUser(c)

    // Get individual fields
    userID, ok := httpEcho.GetUserID(c)
    username, ok := httpEcho.GetUsername(c)

    return c.JSON(http.StatusOK, claims)
}
```

## Testing

All tests run with race detector:

```bash
go test -race ./http/echo/...
```

## Implementation Details

### Server Wrapper

The `Server` type embeds both `*http.Server` and `*echo.Echo`:

- `*http.Server`: Provides Govern's graceful shutdown and HTTP server management
- `*echo.Echo`: Provides full Echo API access

### Skip Path Matching

JWT middleware uses proper prefix matching to avoid false positives:

- `/api` matches `/api/users` but NOT `/apifoo`
- `/api/` matches `/api/users`
- Exact match for `/health` matches `/health` only

## Best Practices

1. **Use Echo middleware** for Echo routes (logging, recovery, CORS)
2. **Use WrapHandler()** to reuse standard `http.Handler` with Echo
3. **Always pass context** as first parameter to downstream operations
4. **Test with race detector**: `go test -race ./http/echo/...`
5. **Run golangci-lint**: `golangci-lint run ./http/echo/...`

## Examples

See `example_test.go` for more usage examples.
