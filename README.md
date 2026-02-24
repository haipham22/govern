# Govern

A lightweight, production-ready Go library for building robust HTTP services.

## Installation

```bash
go get github.com/haipham22/govern
```

## Packages

| Package | Description |
|---------|-------------|
| [`graceful`](./graceful/) | HTTP server with graceful shutdown and background process management |
| [`database/postgres`](./database/postgres/) | PostgreSQL connection management with GORM |
| [`errors`](./errors/) | Structured error handling with error codes |
| [`log`](./log/) | Zap-based structured logging |
| [`metrics`](./metrics/) | Prometheus metrics with HTTP middleware |
| [`healthcheck`](./healthcheck/) | Health check registry for liveness/readiness probes |
| [`retry`](./retry/) | Flexible retry logic with backoff strategies |

## Roadmap

### Phase 1: Core Functionality ✅
- [x] Graceful server shutdown
- [x] PostgreSQL connection management
- [x] Structured error handling
- [x] Zap-based logging package

### Phase 2: Enhancement ✅
- [x] Metrics integration (Prometheus)
- [x] Health check utilities
- [x] Retry logic

### Phase 3: Ecosystem
- [ ] Circuit breaker pattern
- [ ] Docker compose examples
- [ ] Kubernetes deployment guides
- [ ] Integration with popular frameworks (gin, echo, fiber)

## Requirements

- Go 1.25.5+

## License

TBD

---

**Repository**: https://github.com/haipham22/govern
