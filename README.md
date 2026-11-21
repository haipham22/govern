# Govern

![CI](https://github.com/haipham22/govern/workflows/CI/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/haipham22/govern)](https://goreportcard.com/report/github.com/haipham22/govern)

A lightweight Go library of production-ready building blocks: config, structured errors, logging, retries, graceful shutdown, health checks, an outbound HTTP client, and S3 storage.

## Installation

```bash
go get github.com/haipham22/govern
```

## Features

| Package                                     | Description                                                  |
|---------------------------------------------|--------------------------------------------------------------|
| [`config`](./config/)                       | Configuration loading with YAML, .env, and ENV variables      |
| [`errors`](./errors/)                       | Structured error handling with error codes                   |
| [`log`](./log/)                             | Zap-based structured logging                                 |
| [`retry`](./retry/)                         | Flexible retry logic with backoff strategies                 |
| [`graceful`](./graceful/)                   | Goroutine and process management with graceful shutdown      |
| [`healthcheck`](./healthcheck/)             | Health check registry for liveness/readiness probes          |
| [`http/client`](./http/client/)             | Outbound HTTP client: retries, request-id propagation, JSON helpers |
| [`s3`](./s3/)                               | S3-compatible object storage (AWS SDK v2): upload, download, presign, remove |

## Removed (2026-11-21)

`mq`, `cron`, `events`, `outbox`, `database`, `mailer`, `metrics`, `otlp`, `http server/jwt/request` — unused by the consumer project; recover from git history if ever needed.

## Quick Start

See [QUICKSTART.md](./QUICKSTART.md) for installation and usage examples.

## Comprehensive Documentation

For detailed documentation, see the `docs/` directory:
- [Project Overview & PDR](./docs/project-overview-pdr.md) - Project overview and requirements
- [Codebase Summary](./docs/codebase-summary.md) - Comprehensive codebase analysis
- [Code Standards](./docs/code-standards.md) - Coding standards and conventions
- [System Architecture](./docs/system-architecture.md) - Architecture and design patterns
- [Deployment Guide](./docs/deployment-guide.md) - Deployment instructions

## Roadmap

### Phase 1: Core Functionality ✅
- [x] Structured error handling
- [x] Zap-based logging package
- [x] Retry logic
- [x] Health check utilities

### Phase 2: Integration ✅
- [x] Outbound HTTP client with retry semantics and request-id propagation
- [x] S3-compatible object storage client
- [ ] Circuit breaker pattern

## Requirements

- Go 1.25 or 1.26

## Development

See [DEVELOPMENT.md](./DEVELOPMENT.md) for development setup, testing, and contribution guidelines.

## License

TBD

---

**Repository**: https://github.com/haipham22/govern
