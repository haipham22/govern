# Echo Utilities

The `echo` package provides utilities for using [Echo](https://echo.labstack.com/) framework with Govern, including JWT authentication middleware, Swagger UI integration, and handler wrapping utilities.

## Features

- **JWT Authentication**: Echo-specific JWT middleware using `http/jwt`
- **Swagger UI Integration**: Configuration utilities for Swagger UI with authentication support
- **Handler Wrapping**: Reuse standard `http.Handler` with Echo via `WrapHandler()`
- **Context Helpers**: Utilities for retrieving current user from Echo context

## Installation

```bash
go get github.com/haipham22/govern/http/echo
```

## JWT Authentication

Create JWT authentication middleware for Echo routes:

```go
package main

import (
    httpEcho "github.com/haipham22/govern/http/echo"
    "github.com/haipham22/govern/http/jwt"
    "github.com/labstack/echo/v4"
    "net/http"
)

func main() {
    e := echo.New()

    // JWT configuration
    jwtConfig := &jwt.MiddlewareConfig{
        Config:         jwt.DefaultConfig(),
        TokenExtractor: jwt.DefaultTokenExtractor,
        SkipPaths:      []string{"/health", "/login"},
    }
    jwtConfig.Config.Secret = "your-secret-key"

    // Public routes
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    // Protected routes group
    api := e.Group("/api", httpEcho.JWTMiddleware(jwtConfig))
    api.GET("/profile", func(c echo.Context) error {
        claims := httpEcho.MustGetCurrentUser(c)
        return c.JSON(http.StatusOK, map[string]interface{}{
            "user_id":  claims.UserID,
            "username": claims.Username,
        })
    })

    e.Start(":8080")
}
```

### Context Helpers

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

## Swagger UI Integration

Configure Swagger UI for your Echo application:

### Setup

1. Install swag CLI:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

2. Add Swagger annotations to your handlers:

```go
// @Summary Get user by ID
// @Description Retrieve user information
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} User
// @Failure 404 {object} ErrorResponse
// @Router /users/{id} [get]
func getUser(c echo.Context) error {
    // Handler logic
}
```

3. Generate Swagger docs:

```bash
swag init -g cmd/api/main.go
```

4. Configure Swagger UI:

```go
import _ "myapi/docs"  // Generated docs

func setupSwagger(e *echo.Echo) {
    swaggerOpts := []httpEcho.SwaggerOption{
        httpEcho.WithSwaggerEnabled(true),
        httpEcho.WithSwaggerInfo(&httpEcho.SwaggerInfo{
            Title:       "My API",
            Description: "Sample API server",
            Version:     "1.0",
        }),
    }

    // Apply swagger configuration to Echo instance
    // (Implementation depends on your Echo setup)
}
```

### Authentication

Enable Bearer token authentication in Swagger UI:

```go
swaggerOpts := []httpEcho.SwaggerOption{
    httpEcho.WithSwaggerEnabled(true),
    httpEcho.WithSwaggerAuth(&httpEcho.SwaggerAuth{
        Type:        "Bearer",
        Description: "JWT token",
        Name:        "Authorization",
        In:          "header",
    }),
}
```

Add these annotations to main.go:

```go
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"
```

### Security

Swagger UI should not be exposed in production without authentication. Use environment variables:

```go
enableSwagger := os.Getenv("GO_ENV") == "development"
swaggerOpts := []httpEcho.SwaggerOption{
    httpEcho.WithSwaggerEnabled(enableSwagger),
    httpEcho.WithSwaggerPath("/swagger/*"),
}
```

## Handler Wrapping

Reuse standard `http.Handler` with Echo:

```go
import (
    "net/http"
    httpEcho "github.com/haipham22/govern/http/echo"
)

// Reuse standard http.Handler with Echo
var myHandler http.Handler = // ...

e.GET("/legacy", httpEcho.WrapHandler(myHandler))
```

## Middleware Compatibility

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
)

e := echo.New()

// Use Echo's built-in middleware
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())
```

## Testing

All tests run with race detector:

```bash
go test -race ./http/echo/...
```

## Best Practices

1. **Use Echo middleware** for Echo routes (logging, recovery, CORS)
2. **Use WrapHandler()** to reuse standard `http.Handler` with Echo
3. **Always pass context** as first parameter to downstream operations
4. **Test with race detector**: `go test -race ./http/echo/...`
5. **Run golangci-lint**: `golangci-lint run ./http/echo/...`

## Examples

See `swagger_auth_example.go` for Swagger authentication examples.
