package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haipham22/govern/http/middleware"
	"github.com/haipham22/govern/retry"
)

// fastPolicy keeps retry tests quick while still exercising multiple attempts.
func fastPolicy() *retry.Policy {
	return retry.NewPolicy(
		retry.MaxAttempts(3),
		retry.MaxDuration(2*time.Second),
		retry.Backoff(retry.NewConstantBackoff(time.Millisecond)),
	)
}

// newFastClient is a client tuned for tests: fast backoff, tight timeout.
func newFastClient(opts ...Option) *Client {
	return New(append([]Option{WithRetryPolicy(fastPolicy()), WithTimeout(2 * time.Second)}, opts...)...)
}

func TestDo_RetrySemantics(t *testing.T) {
	type reply struct {
		status int
		body   string
	}
	tests := []struct {
		name        string
		replies     []reply
		handler     func(calls int32, w http.ResponseWriter)
		wantCalls   int32
		wantStatus  int
		wantErrType any // *Error or nil
		description string
	}{
		{
			name:      "500 then 200 is retried and succeeds",
			replies:   []reply{{500, `{"error":"boom"}`}, {200, `{"ok":true}`}},
			wantCalls: 2, wantStatus: 200,
			description: "5xx is retryable",
		},
		{
			name:      "429 then 200 is retried and succeeds",
			replies:   []reply{{429, "slow down"}, {200, `{"ok":true}`}},
			wantCalls: 2, wantStatus: 200,
			description: "429 is retryable",
		},
		{
			name:      "400 is never retried",
			replies:   []reply{{400, `{"error":"bad request"}`}},
			wantCalls: 1, wantErrType: &Error{}, wantStatus: 400,
			description: "4xx business error terminates immediately",
		},
		{
			name:      "404 is never retried",
			replies:   []reply{{404, "missing"}},
			wantCalls: 1, wantErrType: &Error{}, wantStatus: 404,
			description: "any non-5xx/429 status terminates immediately",
		},
		{
			name:      "500 on every attempt exhausts max attempts",
			replies:   []reply{{500, "boom"}, {500, "boom"}, {500, "boom"}},
			wantCalls: 3, wantErrType: &Error{}, wantStatus: 500,
			description: "exhaustion surfaces the last typed status error",
		},
		{
			name: "transport error is retried",
			handler: func(calls int32, w http.ResponseWriter) {
				if calls == 1 {
					// Drop the connection without a response: transport error.
					hj, ok := w.(http.Hijacker)
					require.True(t, ok)
					conn, _, err := hj.Hijack()
					require.NoError(t, err)
					_ = conn.Close()
					return
				}
				w.WriteHeader(http.StatusOK)
			},
			wantCalls: 2, wantStatus: 200,
			description: "connection drop counts as transport error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := atomic.AddInt32(&calls, 1)
				if tt.handler != nil {
					tt.handler(n, w)
					return
				}
				i := int(n) - 1
				if i >= len(tt.replies) {
					i = len(tt.replies) - 1
				}
				w.WriteHeader(tt.replies[i].status)
				_, _ = w.Write([]byte(tt.replies[i].body))
			}))
			defer srv.Close()

			c := newFastClient()
			resp, err := c.Do(context.Background(), &Request{Method: http.MethodGet, URL: srv.URL})

			assert.Equal(t, tt.wantCalls, atomic.LoadInt32(&calls), tt.description)
			if tt.wantErrType != nil {
				require.Error(t, err)
				var statusErr *Error
				require.True(t, errors.As(err, &statusErr), "want *Error, got %T: %v", err, err)
				assert.Equal(t, tt.wantStatus, statusErr.StatusCode)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestJSONHelpers(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	tests := []struct {
		name        string
		run         func(t *testing.T, c *Client, srv *httptest.Server, seen *atomic.Pointer[http.Request])
		description string
	}{
		{
			name: "GetJSON decodes the response body",
			run: func(t *testing.T, c *Client, srv *httptest.Server, _ *atomic.Pointer[http.Request]) {
				var got payload
				err := c.GetJSON(context.Background(), srv.URL+"/thing", &got)
				require.NoError(t, err)
				assert.Equal(t, payload{Name: "thing", Count: 1}, got)
			},
		},
		{
			name: "PostJSON sends a JSON body and decodes the reply",
			run: func(t *testing.T, c *Client, srv *httptest.Server, seen *atomic.Pointer[http.Request]) {
				var got payload
				err := c.PostJSON(context.Background(), srv.URL+"/things", map[string]any{"name": "x"}, &got)
				require.NoError(t, err)
				assert.Equal(t, payload{Name: "created", Count: 2}, got)

				req := seen.Load()
				require.NotNil(t, req)
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
			},
		},
		{
			name: "204 empty body is not a decode error",
			run: func(t *testing.T, c *Client, srv *httptest.Server, _ *atomic.Pointer[http.Request]) {
				var got payload
				err := c.GetJSON(context.Background(), srv.URL+"/empty", &got)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seenReq atomic.Pointer[http.Request]
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenReq.Store(r)
				switch r.URL.Path {
				case "/empty":
					w.WriteHeader(http.StatusNoContent)
				default:
					w.WriteHeader(http.StatusOK)
					if r.Method == http.MethodPost {
						// Consume the sent body so the test asserts delivery.
						var in map[string]any
						_ = json.NewDecoder(r.Body).Decode(&in)
						_ = json.NewEncoder(w).Encode(payload{Name: "created", Count: 2})
						return
					}
					_ = json.NewEncoder(w).Encode(payload{Name: "thing", Count: 1})
				}
			}))
			defer srv.Close()

			tt.run(t, newFastClient(), srv, &seenReq)
		})
	}
}

func TestDecodeError_IsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	var out map[string]any
	err := newFastClient().GetJSON(context.Background(), srv.URL, &out)
	require.Error(t, err)

	var decErr *DecodeError
	require.True(t, errors.As(err, &decErr), "want *DecodeError, got %T: %v", err, err)
	assert.Equal(t, "not json at all", string(decErr.Body))
	assert.ErrorIs(t, err, decErr.Err)
}

func TestRequestID_PropagationAndGeneration(t *testing.T) {
	tests := []struct {
		name        string
		ctxValue    string
		wantID      string
		generated   bool
		description string
	}{
		{
			name:        "propagates the middleware request id from context",
			ctxValue:    "req-abc-123",
			wantID:      "req-abc-123",
			description: "X-Request-ID mirrors the middleware ctx value",
		},
		{
			name:        "generates a uuid without a ctx value",
			generated:   true,
			description: "absent ctx value yields a parseable uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID atomic.Pointer[string]
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id := r.Header.Get("X-Request-ID")
				gotID.Store(&id)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			ctx := context.Background()
			if tt.ctxValue != "" {
				ctx = context.WithValue(ctx, middleware.RequestIDKeyVal, tt.ctxValue)
			}

			c := newFastClient(WithBaseURL(srv.URL))
			_, err := c.Do(ctx, &Request{Method: http.MethodGet, URL: "/ping"})
			require.NoError(t, err)
			if tt.generated {
				_, err := uuid.Parse(*gotID.Load())
				assert.NoError(t, err, "generated id must be a uuid")
			} else {
				assert.Equal(t, tt.wantID, *gotID.Load(), tt.description)
			}
		})
	}
}

func TestError_CarriesRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	ctx := context.WithValue(context.Background(), middleware.RequestIDKeyVal, "propagated-1")
	_, err := newFastClient().Do(ctx, &Request{Method: http.MethodGet, URL: srv.URL})
	require.Error(t, err)

	var statusErr *Error
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, 400, statusErr.StatusCode)
	assert.Equal(t, "propagated-1", statusErr.RequestID)
}

func TestDo_ContextDeadline(t *testing.T) {
	tests := []struct {
		name        string
		clientOpts  []Option
		parentAfter time.Duration
		wantCalls   int32
		description string
	}{
		{
			name:        "per-attempt timeout fires without retries",
			clientOpts:  []Option{WithTimeout(50 * time.Millisecond), WithRetryPolicy(retry.NewPolicy(retry.MaxAttempts(1)))},
			wantCalls:   1,
			description: "slow handler exceeds the per-attempt timeout",
		},
		{
			name:        "parent deadline wins over client timeout",
			clientOpts:  []Option{WithTimeout(10 * time.Second)},
			parentAfter: 50 * time.Millisecond,
			wantCalls:   1,
			description: "caller-set deadline is never shortened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&calls, 1)
				time.Sleep(300 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			ctx := context.Background()
			if tt.parentAfter > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.parentAfter)
				defer cancel()
			}

			start := time.Now()
			_, err := newFastClient(tt.clientOpts...).Do(ctx, &Request{Method: http.MethodGet, URL: srv.URL})
			elapsed := time.Since(start)

			require.Error(t, err)
			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Equal(t, tt.wantCalls, atomic.LoadInt32(&calls), tt.description)
			assert.Less(t, elapsed, 2*time.Second, "deadline must not wait for the handler")
		})
	}
}

func TestOptions(t *testing.T) {
	tests := []struct {
		name  string
		opts  func(srvURL string) []Option
		check func(t *testing.T, req *http.Request)
	}{
		{
			name: "base URL joins a relative path",
			opts: func(srvURL string) []Option { return []Option{WithBaseURL(srvURL)} },
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "/joined/path", req.URL.Path)
			},
		},
		{
			name: "default headers apply and per-request headers win",
			opts: func(srvURL string) []Option {
				return []Option{
					WithBaseURL(srvURL),
					WithDefaultHeaders(map[string]string{"X-Shared": "default", "X-Only": "default"}),
				}
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "default", req.Header.Get("X-Shared"))
				assert.Equal(t, "request", req.Header.Get("X-Only"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seenReq atomic.Pointer[http.Request]
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenReq.Store(r)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			c := newFastClient(tt.opts(srv.URL)...)
			headers := map[string]string(nil)
			if tt.name == "default headers apply and per-request headers win" {
				headers = map[string]string{"X-Only": "request"}
			}

			_, err := c.Do(context.Background(), &Request{Method: http.MethodGet, URL: "/joined/path", Headers: headers})
			require.NoError(t, err)
			require.NotNil(t, seenReq.Load())
			tt.check(t, seenReq.Load())
		})
	}
}

func TestDo_RejectsUnsupportedScheme(t *testing.T) {
	c := newFastClient()
	_, err := c.Do(context.Background(), &Request{Method: http.MethodGet, URL: "ftp://example.test/file"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}
