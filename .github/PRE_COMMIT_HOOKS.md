# Pre-Commit Hooks

This repository uses [pre-commit](https://pre-commit.com/) to maintain code quality.

## Installation

```bash
# Install pre-commit
brew install pre-commit  # macOS
# or: pip install pre-commit

# Install hooks
pre-commit install

# Run manually on all files
pre-commit run --all-files
```

## Hooks

### Standard Hooks (from pre-commit/pre-commit-hooks)

- **trailing-whitespace** - Trim trailing whitespace
- **end-of-file-fixer** - Ensure files end with newline
- **check-yaml** - Check YAML syntax
- **check-added-large-files** - Prevent large files from being committed
- **check-case-conflict** - Check for merge conflict markers
- **check-merge-conflict** - Check for unresolved merge conflicts
- **check-json** - Check JSON syntax
- **pretty-format-json** - Format JSON files
- **mixed-line-ending** - Fix mixed line endings
- **detect-private-key** - Detect private keys
- **fix-byte-order-marker** - Fix UTF-8 byte order marker

### Go-Specific Hooks

- **golangci-lint** - Run Go linters with auto-fix
- **gofmt** - Format Go code
- **goimports** - Fix import order
- **go-test** - Run all tests
- **go-build** - Verify code builds
- **go-mod-tidy** - Tidy go.mod and go.sum

## Skipping Hooks

To skip hooks for a commit (not recommended):

```bash
git commit --no-verify -m "message"
```

To skip a specific hook:

```bash
SKIP=golangci-lint git commit -m "message"
```

## Updating Hooks

```bash
# Auto-update all hooks
pre-commit autoupdate

# Update specific repo
pre-commit autoupdate --repo https://github.com/pre-commit/pre-commit-hooks
```
