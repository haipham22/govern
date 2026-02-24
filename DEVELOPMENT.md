# Development Guide

## Quick Start

```bash
# Clone the repository
git clone https://github.com/haipham22/govern.git
cd govern

# Install development tools using mise (recommended)
mise install

# Or install Go dependencies manually
go mod download

# Set up pre-commit hooks (optional)
pre-commit install
```

## Requirements

- Go 1.25+
- make (optional, for makefiles)
- golangci-lint (for linting)
- pre-commit (optional, for git hooks)

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run tests for specific package
go test ./config/...

# Run tests with verbose output
go test -v ./config/...
```

## Running Linters

```bash
# Run all linters with auto-fix
golangci-lint run --fix

# Run without auto-fix
golangci-lint run

# Run for specific package
golangci-lint run ./config/...

# Run with specific linters only
golangci-lint run --disable-all --enable=gofmt,goimports,vet ./...
```

## Pre-Commit Hooks

The project uses pre-commit hooks to ensure code quality. Install them with:

```bash
pre-commit install
```

The pre-commit configuration (`.pre-commit-config.yaml`) runs:
- **golangci-lint**: Automatically fixes and checks Go code

To skip pre-commit hooks (not recommended):
```bash
git commit --no-verify -m "message"
```

## Code Standards

- Follow Go best practices from [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write tests for all new features
- Aim for >80% test coverage
- Use table-driven tests for multiple cases
- Add godoc comments to exported functions

## Project Structure

```
govern/
├── config/           # Configuration loading
├── database/         # Database connections (postgres, redis)
├── docs/             # Project documentation
├── errors/           # Error handling
├── graceful/         # Graceful shutdown
├── healthcheck/      # Health check utilities
├── http/             # HTTP server and middleware
│   ├── echo/        # Echo framework integration
│   ├── jwt/         # JWT authentication
│   └── middleware/  # HTTP middleware
├── log/              # Logging utilities
├── metrics/          # Prometheus metrics
├── retry/            # Retry logic
└── plans/            # Implementation plans
```

## Adding New Features

1. Create a plan in `plans/` directory
2. Follow the existing code patterns
3. Write tests before implementation
4. Run linters and fix issues
5. Update documentation
6. Submit PR with clear description

## Testing Strategy

- **Unit tests**: Test individual functions
- **Table-driven tests**: Test multiple scenarios
- **Race detector**: Run with `-race` flag
- **Integration tests**: Test package interactions
- **Example tests**: Verify documentation examples work

## Common Commands

```bash
# Format code
gofmt -w .
goimports -w .

# Run vet
go vet ./...

# Build
go build ./...

# Install dependencies
go mod tidy
go mod download

# Update dependencies
go get -u ./...
go mod tidy
```

## Useful Links

- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [golangci-lint Config](https://golangci-lint.run/usage/configuration/)
