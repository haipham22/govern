# Quick Start Guide

Get up and running with Govern in minutes.

## Installation

```bash
go get github.com/haipham22/govern
```

## Configuration Loading

### Loading from YAML

Load configuration from YAML files with environment variable override:

**config.yaml**:
```yaml
server:
  host: "0.0.0.0"
  port: 8080

postgres:
  host: "localhost"
  port: 5432
  database: "myapp"
  user: "dbuser"
```

**main.go**:
```go
package main

import (
    "fmt"
    "github.com/haipham22/govern/config"
)

type Config struct {
    Server struct {
        Host string `validate:"required"`
        Port int    `validate:"min=1,max=65535"`
    } `validate:"required"`
    Postgres struct {
        Host     string `validate:"required"`
        Port     int    `validate:"required"`
        Database string `validate:"required"`
        User     string `validate:"required"`
    } `validate:"required"`
}

func main() {
    // Load config from YAML (ENV vars override YAML values)
    cfg, err := config.Load[Config]("./config.yaml")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
}
```

**ENV override**:
```bash
# Override server port
SERVER_PORT=9090 ./app

# Override postgres host
POSTGRES_HOST=prod.db.local ./app
```

### Loading from .env File

Load configuration directly from .env files:

**.env**:
```bash
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

**main.go**:
```go
type Config struct {
    ServerHost string `mapstructure:"SERVER_HOST" validate:"required"`
    ServerPort int    `mapstructure:"SERVER_PORT" validate:"required,min=1,max=65535"`
    DatabaseHost string `mapstructure:"DATABASE_HOST" validate:"required"`
    DatabasePort int    `mapstructure:"DATABASE_PORT" validate:"required,min=1,max=65535"`
}

func main() {
    cfg, err := config.LoadFromEnv[Config](".env")
    if err != nil {
        panic(err)
    }
}
```

### Combining YAML + .env

```go
cfg, err := config.LoadWithOptions[Config](
    "config.yaml",
    config.WithEnvFile(".env"),
)
```

**Priority**: System ENV > .env file > YAML values

## Echo Server with JWT

Create an HTTP server with JWT authentication using Echo framework:

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

    // Public route
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

## PostgreSQL Connection

Connect to PostgreSQL with connection pooling:

```go
package main

import (
    "github.com/haipham22/govern/database/postgres"
)

func main() {
    db, cleanup, err := postgres.New(&postgres.Config{
        Host:         "localhost",
        Port:         5432,
        Database:     "mydb",
        User:         "user",
        Password:     "password",
        MaxIdleConns: 25,
        MaxOpenConns: 100,
    })
    if err != nil {
        panic(err)
    }
    defer cleanup()

    // Use db with GORM
    // db.AutoMigrate(&User{})
}
```

## Complete Example

Putting it all together:

```go
package main

import (
    "github.com/haipham22/govern/config"
    "github.com/haipham22/govern/database/postgres"
    httpEcho "github.com/haipham22/govern/http/echo"
)

type Config struct {
    Server struct {
        Port int `validate:"min=1,max=65535"`
    } `validate:"required"`
    Postgres struct {
        Host     string `validate:"required"`
        Port     int    `validate:"required"`
        Database string `validate:"required"`
        User     string `validate:"required"`
        Password string `validate:"required"`
    } `validate:"required"`
}

func main() {
    // 1. Load configuration
    cfg, err := config.Load[Config]("./config.yaml")
    if err != nil {
        panic(err)
    }

    // 2. Connect to database
    db, cleanup, err := postgres.New(&postgres.Config{
        Host:     cfg.Postgres.Host,
        Port:     cfg.Postgres.Port,
        Database: cfg.Postgres.Database,
        User:     cfg.Postgres.User,
        Password: cfg.Postgres.Password,
    })
    if err != nil {
        panic(err)
    }
    defer cleanup()

    // 3. Start HTTP server
    server := httpEcho.NewServer(":8080")
    server.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{"status": "ok"})
    })
    server.Start()
}
```

## Next Steps

- Read [package documentation](./README.md#features) for all available packages
- Check [DEVELOPMENT.md](./DEVELOPMENT.md) for contribution guidelines
- See [docs/](./docs/) for detailed architecture and design

## Examples

See the [examples directory](./examples/) for complete working examples:
- Basic HTTP server
- JWT authentication
- Database integration
- Configuration management
