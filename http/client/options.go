package client

import (
	"time"

	"go.uber.org/zap"

	"github.com/haipham22/govern/retry"
)

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the per-attempt timeout. A deadline already present on the
// caller's context is never shortened.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithMaxRetries sets the maximum number of attempts (1 = no retries).
// Ignored when WithRetryPolicy supplied a full policy.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithRetryPolicy replaces the retry policy entirely.
func WithRetryPolicy(p *retry.Policy) Option {
	return func(c *Client) {
		if p != nil {
			c.policy = p
		}
	}
}

// WithBaseURL prepends a base URL to relative request URLs.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithDefaultHeaders sets headers sent with every request.
// Per-request headers take precedence over defaults.
func WithDefaultHeaders(h map[string]string) Option {
	return func(c *Client) {
		for k, v := range h {
			c.defaultHeaders[k] = v
		}
	}
}

// WithLogger sets the logger used for failure diagnostics.
// A nil logger (the default) disables logging.
func WithLogger(l *zap.SugaredLogger) Option {
	return func(c *Client) {
		c.logger = l
	}
}
