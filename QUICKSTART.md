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

## Complete Example

Putting it all together:

```go
package main

import (
    "context"
    "fmt"

    "github.com/haipham22/govern/config"
    governclient "github.com/haipham22/govern/http/client"
)

type Config struct {
    API struct {
        BaseURL string `validate:"required,url"`
    } `validate:"required"`
}

func main() {
    // 1. Load configuration
    cfg, err := config.Load[Config]("./config.yaml")
    if err != nil {
        panic(err)
    }

    // 2. Outbound calls with retries + request-id propagation
    client := governclient.New(governclient.WithBaseURL(cfg.API.BaseURL))
    resp, err := client.Do(context.Background(), &governclient.Request{
        Method: "GET",
        URL:    "/v1/ping",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(resp.StatusCode)
}
```

## Next Steps

- Read [package documentation](./README.md#features) for all available packages
- Check [DEVELOPMENT.md](./DEVELOPMENT.md) for contribution guidelines
- See [docs/](./docs/) for detailed architecture and design

## Examples

See each package's README and doc comments for complete usage examples:
- [`config`](./config/) — YAML, .env, and ENV-variable loading
- [`http/client`](./http/client/) — outbound HTTP with retries and request-id propagation
- [`s3`](./s3/) — object storage upload, download, presigned URLs
