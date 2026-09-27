# otlp

OpenTelemetry setup helper — wires TracerProvider, MeterProvider, and LoggerProvider with OTLP exporters (gRPC or HTTP) from a single `Config`, installs them as globals, and returns a `Shutdown` func for graceful flush.

## Features

- One-call setup for **traces, metrics, and logs**.
- OTLP over **gRPC** (default) or **HTTP**.
- Composite propagator (**W3C TraceContext + Baggage**) installed on setup.
- Standard resource attributes: `service.name`, `service.version`, `deployment.environment`, plus env / host / process / SDK autodetection.
- `Shutdown` is a plain func to defer for a clean flush.
- `NewSlogLogger(name)` returns an `*slog.Logger` bridged into the OTel logs pipeline.

## Installation

```bash
go get github.com/haipham22/govern/otlp
```

## Usage

### Setup + graceful shutdown

```go
import (
    "context"
    "log/slog"
    "time"

    "github.com/haipham22/govern/otlp"
)

func main() {
    ctx := context.Background()

    otelShutdown, err := otlp.Setup(ctx, otlp.Config{
        Enabled:        true,
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        Endpoint:       "otel-collector:4317",
        Protocol:       otlp.ProtocolGRPC,
        Insecure:       true, // in-cluster plaintext
        Traces:         true,
        Metrics:        true,
        Logs:           true,
        SampleRatio:    1.0,
    })
    if err != nil {
        panic(err)
    }

    // Use OTel-backed slog for the app
    logger := otlp.NewSlogLogger("my-service")
    slog.SetDefault(logger)

    defer otelShutdown(ctx) // flush telemetry during shutdown

    // ... start your components (httpSrv.Start(), grpcSrv.Start(), ...)
}
```

### Disable at runtime

Leave `Enabled: false` (the default) — `Setup` returns a no-op shutdown and installs nothing. Same code path in dev and prod:

```go
cfg := otlp.Config{Enabled: os.Getenv("OTEL_ENABLED") == "true", /* ... */ }
shutdown, err := otlp.Setup(ctx, cfg)
```

### HTTP transport

```go
otlp.Config{
    Endpoint: "otel-collector:4318",
    Protocol: otlp.ProtocolHTTP,
    Insecure: false,
    Headers: map[string]string{
        "authorization": "Bearer " + token,
    },
    // ...
}
```

### Authenticated collector

```go
otlp.Config{
    Endpoint: "ingest.honeycomb.io:443",
    Protocol: otlp.ProtocolGRPC,
    Headers: map[string]string{
        "x-honeycomb-team": apiKey,
    },
    // ...
}
```

## Config

| Field            | Type                 | Default          | Notes                                                       |
| ---------------- | -------------------- | ---------------- | ----------------------------------------------------------- |
| `Enabled`        | `bool`               | `false`          | When false, `Setup` is a no-op.                             |
| `ServiceName`    | `string`             | — (required)     | `service.name` resource attr.                               |
| `ServiceVersion` | `string`             | ""               | `service.version`.                                          |
| `Environment`    | `string`             | ""               | `deployment.environment`.                                   |
| `Endpoint`       | `string`             | — (required)     | `host:4317` (gRPC) or `host:4318` (HTTP), no scheme.        |
| `Protocol`       | `otlp.Protocol`      | `ProtocolGRPC`   | `ProtocolGRPC` or `ProtocolHTTP`.                           |
| `Insecure`       | `bool`               | `false`          | Disables TLS.                                               |
| `Headers`        | `map[string]string`  | `nil`            | Sent with every OTLP request.                               |
| `SampleRatio`    | `float64`            | `1.0`            | Trace sampler (ParentBased ratio).                          |
| `ExportTimeout`  | `time.Duration`      | `10s`            | Bounds each export operation.                               |
| `Traces`         | `bool`               | `false`          | Install TracerProvider.                                     |
| `Metrics`        | `bool`               | `false`          | Install MeterProvider.                                      |
| `Logs`           | `bool`               | `false`          | Install LoggerProvider.                                     |
| `ResourceAttrs`  | `map[string]string`  | `nil`            | Extra `resource.KeyValue`s merged with defaults.            |

## What Setup installs

- `otel.SetTracerProvider(...)` when `cfg.Traces`.
- `otel.SetMeterProvider(...)` when `cfg.Metrics`.
- `global.SetLoggerProvider(...)` when `cfg.Logs`.
- `otel.SetTextMapPropagator(...)` — `TraceContext` + `Baggage` — whenever any signal is enabled.

Environment variables recognized by the OTel SDK (e.g. `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_RESOURCE_ATTRIBUTES`) still take effect via `resource.WithFromEnv()` and the exporter defaults.
