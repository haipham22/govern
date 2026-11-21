package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/haipham22/govern/http/middleware"
)

// Error is a typed non-2xx response. 5xx and 429 are retried; every other
// status is a terminal business error.
type Error struct {
	StatusCode int
	Status     string
	Body       []byte
	RequestID  string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("http client: unexpected status %d (%s)", e.StatusCode, e.Status)
}

// DecodeError is a typed JSON decode failure; Body holds the raw payload.
type DecodeError struct {
	Body []byte
	Err  error
}

// Error implements the error interface.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("http client: decode response body: %v", e.Err)
}

// Unwrap returns the underlying JSON error.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

var (
	// Only http/https allowed, matching the commons-golang schema check.
	errInvalidScheme = errors.New("http client: only http and https schemes are allowed")
	// Redirects are never followed; a redirect reply surfaces as its own status.
	httpClient = &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
)

// requestIDFromCtx reuses the govern middleware request id when present,
// else generates a uuid.
func requestIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(middleware.RequestIDKeyVal).(string); ok && id != "" {
		return id
	}
	return uuid.New().String()
}

// isRetryable allows another attempt for transport errors, 5xx and 429 only.
// Build errors (bad URL, invalid method) are not transport failures and
// terminate immediately.
func isRetryable(err error) bool {
	var respErr *Error
	if errors.As(err, &respErr) {
		return respErr.StatusCode >= 500 || respErr.StatusCode == http.StatusTooManyRequests
	}
	var uerr *url.Error
	return errors.As(err, &uerr)
}

// unwrapFinal strips retry-internal wrappers so callers receive typed errors.
func unwrapFinal(err error) error {
	var respErr *Error
	if errors.As(err, &respErr) {
		return respErr
	}
	return err
}
