# Govern

![CI](https://github.com/haipham22/govern/workflows/CI/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/haipham22/govern)](https://goreportcard.com/report/github.com/haipham22/govern)

A lightweight, production-ready Go library for building robust HTTP services with graceful shutdown, database management, JWT authentication, logging, metrics, and health checks.

## Installation

```bash
go get github.com/haipham22/govern
```

## Features

| Package                                     | Description                                                  |
|---------------------------------------------|--------------------------------------------------------------|
| [`config`](./config/)                       | Configuration loading with YAML and ENV variables            |
| [`http`](./http/)                           | HTTP server with graceful shutdown and middleware            |
| [`http/echo`](./http/echo/)                 | Echo framework integration with Govern's HTTP server         |
| [`http/jwt`](./http/jwt/)                   | JWT authentication middleware and token management           |
| [`http/middleware`](./http/middleware/)     | Common HTTP middleware (logging, recovery, CORS, request ID) |
| [`graceful`](./graceful/)                   | Goroutine and process management with graceful shutdown      |
| [`database/postgres`](./database/postgres/) | PostgreSQL connection management with GORM                   |
| [`database/redis`](./database/redis/)       | Redis connection management with connection pooling          |
| [`errors`](./errors/)                       | Structured error handling with error codes                   |
| [`log`](./log/)                             | Zap-based structured logging                                 |
| [`metrics`](./metrics/)                     | Prometheus metrics with HTTP middleware                      |
| [`healthcheck`](./healthcheck/)             | Health check registry for liveness/readiness probes          |
| [`retry`](./retry/)                         | Flexible retry logic with backoff strategies                 |

## Quick Start

See [QUICKSTART.md](./QUICKSTART.md) for installation and usage examples.

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
- [x] HTTP server with middleware

### Phase 3: Ecosystem ✅
- [x] Echo framework integration
- [x] JWT authentication
- [ ] Circuit breaker pattern
- [ ] Additional framework integrations (gin, fiber)

## Requirements

- Go 1.25+

## Development

See [DEVELOPMENT.md](./DEVELOPMENT.md) for development setup, testing, and contribution guidelines.

## License

TBD

---

**Repository**: https://github.com/haipham22/govern
