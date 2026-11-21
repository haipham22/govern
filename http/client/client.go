// Package client provides a minimal net/http outbound client with retry,
// request-id propagation and JSON helpers — the govern counterpart of the
// commons-golang resty client, without the resty dependency.
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/haipham22/govern/retry"
)

const (
	contentTypeJSON = "application/json"
	requestIDHeader = "X-Request-ID"
)

// Client is a minimal outbound HTTP client with retry, request-id
// propagation and JSON helpers. Create with New; the zero value is not usable.
type Client struct {
	timeout        time.Duration
	maxRetries     int
	policy         *retry.Policy
	baseURL        string
	defaultHeaders map[string]string
	logger         *zap.SugaredLogger
}

// Request describes one outbound call. Body is JSON-encoded when non-nil;
// pass json.RawMessage to send raw payloads unchanged.
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    any
}

// Response is a 2xx reply.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// New creates a client with defaults: 10s per-attempt timeout, 3 max attempts,
// retry.NewPolicy exponential backoff.
func New(opts ...Option) *Client {
	c := &Client{
		timeout:        10 * time.Second,
		maxRetries:     3,
		defaultHeaders: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// policy returns the configured policy or a default one built from maxRetries.
func (c *Client) retryPolicy() *retry.Policy {
	if c.policy != nil {
		return c.policy
	}
	return retry.NewPolicy(retry.MaxAttempts(c.maxRetries))
}

// Do executes req, retrying only transport errors, 5xx and 429.
// Non-2xx replies surface as *Error; JSON decode failures as *DecodeError.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	body, err := marshalBody(req.Body)
	// Encode errors are programmer errors: never retry them.
	if err != nil {
		return nil, err
	}

	var resp *Response
	err = c.retryPolicy().DoWithContext(ctx, retry.RetryIf(func(ctx context.Context) error {
		r, aerr := c.attempt(ctx, req, body)
		if aerr == nil {
			resp = r
		}
		return aerr
	}, isRetryable))
	if err != nil {
		return nil, unwrapFinal(err)
	}
	return resp, nil
}

// GetJSON performs a GET and decodes the JSON body into out.
func (c *Client) GetJSON(ctx context.Context, url string, out any, headers ...map[string]string) error {
	return c.doJSON(ctx, http.MethodGet, url, nil, out, headers...)
}

// PostJSON performs a POST with a JSON body and decodes the reply into out.
func (c *Client) PostJSON(ctx context.Context, url string, in, out any, headers ...map[string]string) error {
	return c.doJSON(ctx, http.MethodPost, url, in, out, headers...)
}

// doJSON runs Do and decodes a non-empty 2xx body into out.
func (c *Client) doJSON(ctx context.Context, method, url string, in, out any, headers ...map[string]string) error {
	resp, err := c.Do(ctx, &Request{
		Method:  method,
		URL:     url,
		Headers: mergeHeaders(headers...),
		Body:    in,
	})
	if err != nil {
		return err
	}
	// 204 / empty body: nothing to decode.
	if out == nil || len(resp.Body) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return &DecodeError{Body: resp.Body, Err: err}
	}
	return nil
}
