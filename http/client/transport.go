package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// attempt executes a single call: per-attempt timeout, request id, headers.
func (c *Client) attempt(ctx context.Context, req *Request, body []byte) (*Response, error) {
	// Per-attempt timeout; a caller-set deadline wins.
	if c.timeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.timeout)
			defer cancel()
		}
	}

	id := requestIDFromCtx(ctx)
	httpReq, err := c.newHTTPRequest(ctx, req, body, id)
	if err != nil {
		return nil, err
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		c.warn("http client: request failed",
			"method", req.Method, "url", req.URL, "error", err)
		return nil, err
	}
	defer func() {
		// Drain so the connection stays reusable.
		_, _ = io.Copy(io.Discard, httpResp.Body)
		_ = httpResp.Body.Close()
	}()

	payload, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		c.warn("http client: non-2xx",
			"method", req.Method, "url", req.URL, "status", httpResp.StatusCode, "request_id", id)
		return nil, &Error{StatusCode: httpResp.StatusCode, Status: httpResp.Status, Body: payload, RequestID: id}
	}
	return &Response{StatusCode: httpResp.StatusCode, Header: httpResp.Header, Body: payload}, nil
}

// newHTTPRequest merges default headers, request headers, JSON content-type
// and the request id into a net/http request.
func (c *Client) newHTTPRequest(ctx context.Context, req *Request, body []byte, requestID string) (*http.Request, error) {
	u, err := c.resolve(req.URL)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", contentTypeJSON)
	}
	if httpReq.Header.Get("Accept") == "" {
		httpReq.Header.Set("Accept", contentTypeJSON)
	}
	httpReq.Header.Set(requestIDHeader, requestID)
	return httpReq, nil
}

// resolve joins a relative URL against baseURL and enforces http/https only.
func (c *Client) resolve(rawURL string) (*url.URL, error) {
	ref, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("http client: invalid url %q: %w", rawURL, err)
	}
	if !ref.IsAbs() && c.baseURL != "" {
		base, berr := url.Parse(c.baseURL)
		if berr != nil {
			return nil, fmt.Errorf("http client: base url %q: %w", c.baseURL, berr)
		}
		ref = base.ResolveReference(ref)
	}
	if ref.Scheme != "http" && ref.Scheme != "https" {
		return nil, errInvalidScheme
	}
	return ref, nil
}

// mergeHeaders merges header maps; later maps win.
func mergeHeaders(headers ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, h := range headers {
		for k, v := range h {
			merged[k] = v
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// marshalBody JSON-encodes Body; nil encodes to no payload.
func marshalBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("http client: encode request body: %w", err)
	}
	return b, nil
}

// warn logs at Warn level; no-op without a logger. Bodies and headers are
// never logged, so tokens never reach logs.
func (c *Client) warn(args ...any) {
	if c.logger != nil {
		c.logger.Warn(args...)
	}
}
