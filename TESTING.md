# Testing Guidelines

This document provides testing guidelines and best practices for the govern project, based on table-driven test patterns.

## Table of Contents

- [Core Principles](#core-principles)
- [Table-Driven Test Patterns](#table-driven-test-patterns)
- [Best Practices](#best-practices)
- [Common Helpers](#common-helpers)
- [Examples](#examples)

## Core Principles

1. **Use Table-Driven Tests** for multiple similar test cases
2. **Separate Test Data from Test Logic** - Keep table cells declarative
3. **Use `t.Helper()`** in all helper functions
4. **Prefer Testify** over manual assertions
5. **Keep Tests Independent** - No test dependencies
6. **Make Tests Readable** - Test intent should be immediately clear

## Table-Driven Test Patterns

### Pattern 1: Function Factory

**Use when:** Simple tests with minimal setup, retry logic, or error scenarios

```go
func TestRetry(t *testing.T) {
    tests := []struct {
        name      string
        fnFactory func(*int) func() error
        wantErr   bool
        wantCalls int
    }{
        {
            name: "success after retries",
            fnFactory: func(calls *int) func() error {
                return func() error {
                    (*calls)++
                    if *calls < 2 {
                        return errors.New("temporary error")
                    }
                    return nil
                }
            },
            wantErr:   false,
            wantCalls: 2,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            calls := 0
            fn := tt.fnFactory(&calls)
            err := retry.Do(fn)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            assert.Equal(t, tt.wantCalls, calls)
        })
    }
}
```

### Pattern 2: Test Function

**Use when:** Complex tests with context, goroutines, channels, or multiple steps

```go
func TestWithContext(t *testing.T) {
    type TestParams struct {
        wantErr   bool
        wantCalls int
    }

    tests := []struct {
        name     string
        testFunc func(*testing.T, context.Context, *TestParams)
    }{
        {
            name: "context cancellation",
            testFunc: func(t *testing.T, ctx context.Context, params *TestParams) {
                cancel := make(chan struct{})
                close(cancel)

                select {
                case <-ctx.Done():
                    assert.True(t, true)
                case <-time.After(time.Second):
                    t.Fatal("context not cancelled")
                }
            },
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx, cancel := context.WithCancel(context.Background())
            defer cancel()

            params := &TestParams{wantErr: true}
            tt.testFunc(t, ctx, params)
        })
    }
}
```

### Pattern 3: Helper Types

**Use when:** Multiple configuration options or type-safe parameters

```go
type serverConfig struct {
    Host string
    Port int
}

func TestConfigLoad(t *testing.T) {
    tests := []struct {
        name        string
        configType  string
        configValue interface{}
        wantErr     bool
    }{
        {
            name: "server config",
            configType: "server",
            configValue: serverConfig{
                Host: "localhost",
                Port: 8080,
            },
            wantErr: false,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var cfg interface{}
            switch tt.configType {
            case "server":
                cfg = tt.configValue.(serverConfig)
            }

            err := config.Load(cfg)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Pattern 4: Validation Functions

**Use when:** Multiple assertion patterns across test cases

```go
type validationFunc func(*testing.T, Result)

func validateSuccess(t *testing.T, result Result) {
    t.Helper()
    assert.True(t, result.Success)
    assert.NoError(t, result.Error)
}

func validateError(t *testing.T, result Result) {
    t.Helper()
    assert.False(t, result.Success)
    assert.Error(t, result.Error)
}

func TestHealthCheck(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(*Registry)
        validate validationFunc
    }{
        {
            name: "all checks passing",
            setup: func(r *Registry) {
                r.Register("db", func() error { return nil })
            },
            validate: func(t *testing.T, resp *Response) {
                t.Helper()
                assert.Equal(t, StatusPassing, resp.Status)
            },
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            registry := NewRegistry()
            tt.setup(registry)

            result := registry.Run(context.Background())
            tt.validate(t, result)
        })
    }
}
```

### Pattern 5: Unified Metric Testing

**Use when:** Testing multiple types with similar operations

```go
func TestMetrics(t *testing.T) {
    tests := []struct {
        name      string
        metric    string
        setup     func(*prometheus.Registry)
        assertion string
    }{
        {
            name:   "counter increments",
            metric: "counter",
            setup: func(r *prometheus.Registry) {
                counter := NewCounter(r, "test", "help")
                counter.Inc()
            },
            assertion: "test_total 1",
        },
        {
            name:   "gauge sets value",
            metric: "gauge",
            setup: func(r *prometheus.Registry) {
                gauge := NewGauge(r, "test", "help")
                gauge.Set(42)
            },
            assertion: "test 42",
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            registry := prometheus.NewRegistry()
            tt.setup(registry)

            // Test logic here
            metrics := gatherMetrics(registry)
            assert.Contains(t, metrics, tt.assertion)
        })
    }
}
```

### Pattern 6: Test Helpers

**Use when:** Repeated validation logic

```go
func assertErrorCode(t *testing.T, err error, wantCode string) {
    t.Helper()
    var ec *ErrorWithCode
    assert.True(t, errors.As(err, &ec))
    assert.Equal(t, wantCode, ec.Code)
}

func assertNoCode(t *testing.T, err error) {
    t.Helper()
    var ec *ErrorWithCode
    assert.False(t, errors.As(err, &ec))
}

func TestErrors(t *testing.T) {
    tests := []struct {
        name string
        err  error
        code string
    }{
        {
            name: "error with code",
            err:  WrapWithCode(errors.New("test"), "ERR001"),
            code: "ERR001",
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.code != "" {
                assertErrorCode(t, tt.err, tt.code)
            } else {
                assertNoCode(t, tt.err)
            }
        })
    }
}
```

### Pattern 7: Option Validation

**Use when:** Testing configuration options

```go
func TestLoggerOptions(t *testing.T) {
    tests := []struct {
        name     string
        option   Option
        validate func(*testing.T, *zap.SugaredLogger)
    }{
        {
            name:   "with debug level",
            option: WithLevel(zap.DebugLevel),
            validate: func(t *testing.T, logger *zap.SugaredLogger) {
                require.NotNil(t, logger)
            },
        },
        {
            name:   "with json encoding",
            option: WithEncoding("json"),
            validate: func(t *testing.T, logger *zap.SugaredLogger) {
                assert.NotNil(t, logger)
            },
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            logger := New(tt.option)
            tt.validate(t, logger)
        })
    }
}
```

## Best Practices

### ✅ Do

1. **Use `t.Helper()`** in all helper functions
   ```go
   func assertNotNil(t *testing.T, val interface{}) {
       t.Helper()  // Marks this as a helper function
       assert.NotNil(t, val)
   }
   ```

2. **Extract setup functions** to reduce table cell complexity
   ```go
   func createTempFile(t *testing.T, content string) string {
       t.Helper()
       // ... implementation
   }
   ```

3. **Define types at test scope** (not in table cells)
   ```go
   type testConfig struct {
       Field1 string
       Field2 int
   }
   // Use in table, don't define inline
   ```

4. **Use Testify** for assertions
   ```go
   // Good
   assert.Equal(t, want, got)
   require.NotNil(t, val)

   // Bad
   if got != want {
       t.Errorf("got %v, want %v", got, want)
   }
   ```

5. **Keep table cells declarative**
   ```go
   // Good - describes what
   {
       name: "test case",
       input: 42,
       want: "result",
   }

   // Bad - describes how
   {
       name: "test case",
       setup: func() {
           // 20 lines of setup logic
       },
   }
   ```

6. **Use descriptive test names**
   ```go
   // Good
   "success after retries"
   "max attempts exceeded"
   "context cancellation during retry"

   // Bad
   "test1"
   "test2"
   "retry test"
   ```

### ❌ Avoid

1. **Conditional logic based on test names**
   ```go
   // Bad
   if tt.name == "test1" {
       // setup for test1
   } else if tt.name == "test2" {
       // setup for test2
   }

   // Good - use function factory or test function
   ```

2. **Complex function definitions in table cells**
   ```go
   // Bad
   {
       name: "test",
       fn: func() error {
           // 20 lines of complex logic
       },
   }

   // Good - extract to function factory
   {
       name: "test",
       fnFactory: func() func() error {
           return func() error { /* implementation */ }
       },
   }
   ```

3. **Inline file operations**
   ```go
   // Bad
   {
       name: "test",
       setup: func() {
           // Create temp file
               f, _ := os.CreateTemp("", "test")
               f.WriteString(content)
               f.Close()
           // ...
       },
   }

   // Good - use helper
   {
       name: "test",
       file: createTempFile(t, "content"),
   }
   ```

4. **Type definitions in table cells**
   ```go
   // Bad
   {
       name: "test",
       config: struct {
           Field1 string
           Field2 int
       }{"value", 42},
   }

   // Good - define type at test scope
   type testConfig struct {
       Field1 string
       Field2 int
   }
   // Then use in table
   ```

5. **Manual assertions**
   ```go
   // Bad
   if got != want {
       t.Errorf("got %v, want %v", got, want)
   }

   // Good
   assert.Equal(t, want, got)
   ```

## Common Helpers

### File Operations

```go
// Create temporary file with content
func createTempFile(t *testing.T, content string) string {
    t.Helper()
    f, err := os.CreateTemp("", "*.yaml")
    require.NoError(t, err)
    _, err = f.WriteString(content)
    require.NoError(t, err)
    require.NoError(t, f.Close())
    return f.Name()
}

// Cleanup temp file
func cleanupFile(t *testing.T, path string) {
    t.Helper()
    if path != "" {
        os.Remove(path)
    }
}
```

### Environment Variables

```go
// Set environment variables with cleanup
func setEnvVars(t *testing.T, vars []string) func() {
    t.Helper()
    oldVals := make(map[string]string)
    for _, v := range vars {
        parts := strings.SplitN(v, "=", 2)
        oldVals[parts[0]] = os.Getenv(parts[0])
        os.Setenv(parts[0], parts[1])
    }
    return func() {
        for k, v := range oldVals {
            if v == "" {
                os.Unsetenv(k)
            } else {
                os.Setenv(k, v)
            }
        }
    }
}
```

### Assertions

```go
// Assert error has specific code
func assertErrorCode(t *testing.T, err error, code string) {
    t.Helper()
    var ec *ErrorWithCode
    assert.True(t, errors.As(err, &ec))
    assert.Equal(t, code, ec.Code)
}

// Assert value is not nil
func assertNotNil[T any](t *testing.T, val T) {
    t.Helper()
    assert.NotNil(t, val)
}

// Assert slices equal
func assertSlicesEqual[T comparable](t *testing.T, want, got []T) {
    t.Helper()
    assert.Equal(t, want, got)
}
```

## Examples

See the following files for complete examples:

- `retry/retry_test.go` - Function Factory pattern
- `config/config_test.go` - Helper Types pattern
- `healthcheck/registry_test.go` - Validation Functions pattern
- `metrics/metrics_test.go` - Unified Metric Testing pattern
- `errors/error_code_test.go` - Test Helpers pattern
- `log/logger_test.go` - Option Validation pattern

## Testing Checklist

Before submitting tests, ensure:

- [ ] All tests pass (`go test ./...`)
- [ ] Tests run with race detector (`go test -race ./...`)
- [ ] Code coverage is sufficient (aim for >80%)
- [ ] No test dependencies (tests can run independently)
- [ ] Descriptive test names
- [ ] Used `t.Helper()` in helper functions
- [ ] Used Testify for assertions
- [ ] Table cells are declarative (what, not how)
- [ ] No complex logic in table cells
- [ ] Types defined at test scope, not in tables

## Resources

- [Table-Driven Tests in Go](https://dev.to/boncheff/table-driven-unit-tests-in-go-407b)
- [Testing Tips for Go](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table-Driven Tests (Go Blog)](https://go.dev/wiki/TableDrivenTests)

## Questions?

If you have questions about testing patterns or need help applying them, refer to the completed refactoring examples in the codebase or ask in team discussions.
