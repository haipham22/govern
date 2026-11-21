# s3

Thin S3/MinIO object storage wrapper on AWS SDK v2 — upload, download, stat, remove, presigned GET and idempotent bucket creation.

## Features

- One wrapper for AWS S3 and S3-compatible stores (MinIO, R2, localstack)
- Functional options: endpoint, region, static credentials, path style, secure scheme
- Default bucket: set once, pass `""` per call
- Typed error codes via `govern/errors` (`INTERNAL`, `NOT_FOUND`)
- Presigned GET URLs with expiry
- `EnsureBucket` is idempotent; region becomes the location constraint outside us-east-1
- SDK calls are behind an injectable interface, so unit tests need no server

## Usage

```go
import "github.com/haipham22/govern/s3"

c, err := s3.New(
    s3.WithEndpoint("http://localhost:9000"), // MinIO local
    s3.WithCredentials("access", "secret"),
    s3.WithPathStyle(true),
    s3.WithSecure(false), // http; default true
    s3.WithDefaultBucket("knowledge"),
)
```

### Object operations

```go
// Upload (empty bucket uses the default; empty content type defaults to
// application/octet-stream)
err := c.Upload(ctx, "", "videos/a.mp4", file, size, "video/mp4")

// Download — the caller MUST Close the reader
rc, err := c.Download(ctx, "", "videos/a.mp4")
defer rc.Close()

info, err := c.Stat(ctx, "", "videos/a.mp4") // Size, ContentType, ETag, ...
err = c.Remove(ctx, "", "videos/a.mp4")      // idempotent
```

### Presigned URL

```go
url, err := c.Presign(ctx, "", "videos/a.mp4", 15*time.Minute)
```

### Bucket setup

```go
// Creates the bucket if missing (HEAD first, then CREATE).
// us-east-1 gets no explicit location constraint, other regions do.
err := c.EnsureBucket(ctx, "")
```

### Error handling

```go
_, err := c.Download(ctx, "", "missing")
code, ok := governerrors.GetCode(err) // s3.NOT_FOUND or s3.INTERNAL
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithEndpoint(u)` | Custom endpoint; scheme derived from `WithSecure` when absent | AWS default |
| `WithRegion(r)` | Region | us-east-1 |
| `WithCredentials(access, secret)` | Static keys; omit for the SDK provider chain | chain |
| `WithPathStyle(v)` | Path-style addressing | false |
| `WithSecure(v)` | https (true) or http for the endpoint scheme | true |
| `WithDefaultBucket(b)` | Bucket used when a call passes `""` | "" |
| `WithDebugLogging(v)` | SDK request/response logging | false |

## Testing note

Unit tests cover option mapping, bucket fallback, EnsureBucket branch logic,
error wrapping and offline presigning via an injected fake API. Live ops
against real MinIO/S3 are exercised by consuming projects (e.g. the
affiliate worker's storage adapter) — they are intentionally not part of CI.

## See Also

- [`errors`](../errors/) - error codes carried by this package
- [`retry`](../retry/) - compose with retry for flaky object stores

## License

TBD
