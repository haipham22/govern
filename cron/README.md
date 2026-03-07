# cron

Cron scheduler integration with [gocron v2](https://github.com/go-co-op/gocron) and graceful lifecycle management.

## Features

- ✅ Implements `graceful.Service` interface for clean lifecycle management
- ✅ Thread-safe scheduler start/stop using atomic operations
- ✅ Functional options pattern for configuration
- ✅ Integration with govern's logging (Zap)
- ✅ Multiple job types support (Duration, Cron, Daily, Weekly, Monthly, OneTime, RandomDuration)
- ✅ Configurable location/timezone support
- ✅ Graceful shutdown with timeout

## Installation

```bash
go get github.com/haipham22/govern/cron
```

## Quick Start

```go
package main

import (
    "context"
    "time"

    "github.com/haipham22/govern/cron"
    "github.com/haipham22/govern/graceful"
    "github.com/haipham22/govern/log"
)

func main() {
    logger := log.New()

    // Create cron scheduler
    scheduler, cleanup, _ := cron.New(
        cron.WithLogger(logger),
    )
    defer cleanup()

    // Add a duration-based job (runs every 5 minutes)
    _, _ = scheduler.DurationJob(5*time.Minute, func() {
        logger.Info("Running cleanup job")
        // Your job logic here
    })

    // Run with graceful shutdown
    graceful.Run(
        context.Background(),
        logger,
        30*time.Second,
        scheduler,
    )
}
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithLogger(logger)` | Set Zap logger | `log.Default()` |
| `WithLocation(loc)` | Set timezone | `time.Local` |
| `WithStopTimeout(d)` | Shutdown timeout | `30s` |

## Job Types

### Duration Job
Runs at fixed intervals:
```go
scheduler.DurationJob(10*time.Second, myFunc, arg1, arg2)
```

### Cron Job
Runs using cron expression:
```go
scheduler.CronJob("0 * * * *", true, myFunc) // Every hour
```

### Daily Job
Runs daily at specific times:
```go
import gocronv2 "github.com/go-co-op/gocron/v2"

scheduler.DailyJob(1, gocronv2.NewAtTimes(gocronv2.NewAtTime(9, 0, 0)), myFunc)
```

### More Job Types
- `WeeklyJob` - Weekly on specific weekdays
- `MonthlyJob` - Monthly on specific days
- `OneTimeJob` - Run once at specific time
- `RandomDurationJob` - Random interval between min/max

## Graceful Integration

```go
package main

import (
    "context"
    "time"

    "github.com/haipham22/govern/cron"
    "github.com/haipham22/govern/graceful"
    "github.com/haipham22/govern/http"
    "github.com/haipham22/govern/log"
)

func main() {
    logger := log.New()

    // Create HTTP server
    server, _ := http.NewServer(":8080")

    // Create cron scheduler
    scheduler, cleanup, _ := cron.New(cron.WithLogger(logger))
    defer cleanup()

    // Add jobs
    _, _ = scheduler.DurationJob(time.Hour, cleanupJob)

    // Run both services with graceful shutdown
    graceful.Run(
        context.Background(),
        logger,
        30*time.Second,
        server,
        scheduler,
    )
}
```

## Testing

```bash
go test ./cron/...
```

## API Documentation

See [GoDoc](https://pkg.go.dev/github.com/haipham22/govern/cron) for full API reference.

## Roadmap

- [ ] Phase 2: Job Registration (all 7 job types)
- [ ] Phase 3: Distributed Locker (Redis implementation)
- [ ] Phase 4: Prometheus Metrics
- [ ] Phase 5: Singleton scheduler (`cron.Default()`)

## License

TBD
