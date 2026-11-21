package client

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestError_Message(t *testing.T) {
	err := &Error{StatusCode: http.StatusTooManyRequests, Status: "Too Many Requests", Body: []byte("boom")}
	assert.Equal(t, "http client: unexpected status 429 (Too Many Requests)", err.Error())
}

func TestWithMaxRetries_DisablesRetries(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL), WithMaxRetries(1)).Do(
		t.Context(), &Request{Method: http.MethodGet, URL: "/x"})
	require.Error(t, err)
	assert.Equal(t, int64(1), hits.Load(), "WithMaxRetries(1) means exactly one attempt")
}

func TestWithLogger_StillRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithMaxRetries(1), WithLogger(zap.NewNop().Sugar()))
	resp, err := c.Do(t.Context(), &Request{Method: http.MethodGet, URL: "/x"})
	require.NoError(t, err)
	require.NotNil(t, resp)
}
