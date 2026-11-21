# http/client

Minimal net/http outbound client with retry, request-id propagation and JSON helpers — no resty dependency.

## Features

- Retries only transport errors, 5xx and 429 — never 4xx business errors
- Per-attempt timeout (default 10s); a caller-set deadline is never shortened
- `X-Request-ID` propagation from the middleware context value, uuid generation otherwise
- Typed errors: `*client.Error` for non-2xx, `*client.DecodeError` for JSON decode failures
- JSON convenience helpers `GetJSON` / `PostJSON`
- Redirects are never followed; only http/https schemes are allowed
- Optional zap logger; the zero-config client works without one

## Usage

```go
import "github.com/haipham22/govern/http/client"

c := client.New(
    client.WithBaseURL("https://api.example.test"),
    client.WithDefaultHeaders(map[string]string{"Authorization": "Bearer " + token}),
)

// Simple JSON round-trip (retries on 5xx/429/transport errors)
var out User
err := c.GetJSON(ctx, "/users/42", &out)

err = c.PostJSON(ctx, "/users", map[string]any{"name": "ha"}, &out)
```

### Raw requests

```go
resp, err := c.Do(ctx, &client.Request{
    Method:  "PATCH",
    URL:     "/users/42",
    Headers: map[string]string{"If-Match": etag},
    Body:    patch, // JSON-encoded; use json.RawMessage to send raw bytes
})
```

### Typed errors

```go
resp, err := c.Do(ctx, req)
var statusErr *client.Error
if errors.As(err, &statusErr) {
    // statusErr.StatusCode, .Body, .RequestID
}

var decErr *client.DecodeError
if errors.As(err, &decErr) {
    // decErr.Body holds the undecodable payload
}
```

### Request ID

`X-Request-ID` is taken from the govern middleware context value
(`middleware.RequestIDKeyVal`) when present, else a uuid is generated.
Every typed error carries the request id it was sent with.

### Retry behavior

| Failure                                        | Retried |
|------------------------------------------------|---------|
| Transport error (conn refused, timeout, EOF)   | yes     |
| 5xx                                            | yes     |
| 429                                            | yes     |
| Any other 4xx                                  | never   |
| Request encode error                           | never   |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithTimeout(d)` | Per-attempt timeout (caller deadline wins) | 10s |
| `WithMaxRetries(n)` | Max attempts (ignored if a full policy is set) | 3 |
| `WithRetryPolicy(p)` | Replace the `retry.Policy` entirely | exponential backoff |
| `WithBaseURL(u)` | Base URL for relative request URLs | "" |
| `WithDefaultHeaders(h)` | Headers sent on every request | empty |
| `WithLogger(l)` | zap logger for failure diagnostics | nil (off) |

## See Also

- [`retry`](../../retry/) - the underlying retry policy and backoff strategies
- [`http/middleware`](../middleware/) - server-side request-id context value

## License

TBD
