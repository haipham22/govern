package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveWith builds a request through the TrimStrings middleware and returns
// the body the next handler received.
func serveWith(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var received map[string]any
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if len(raw) > 0 {
			require.NoError(t, sonic.Unmarshal(raw, &received))
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	TrimStrings(next).ServeHTTP(httptest.NewRecorder(), req)
	return received
}

func TestTrimStringsTrimsStringFields(t *testing.T) {
	received := serveWith(t, []byte(`{"username":"  testuser  ","email":"  test@example.com  "}`))

	assert.Equal(t, "testuser", received["username"])
	assert.Equal(t, "test@example.com", received["email"])
}

func TestTrimStringsTrimsNestedStructures(t *testing.T) {
	received := serveWith(t, []byte(`{"user":{"name":"  a  "},"tags":["  x  ","y"]}`))

	user := received["user"].(map[string]any)
	assert.Equal(t, "a", user["name"])
	assert.Equal(t, []any{"x", "y"}, received["tags"])
}

func TestTrimStringsPassesThroughInvalidAndEmptyBodies(t *testing.T) {
	// Invalid JSON passes through untouched.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte("not json"), raw, "body must pass through byte-for-byte")
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	TrimStrings(next).ServeHTTP(httptest.NewRecorder(), req)

	// Empty body reaches the next handler.
	nextCalled := false
	empty := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	emptyReq := httptest.NewRequest(http.MethodPost, "/", nil)
	TrimStrings(empty).ServeHTTP(httptest.NewRecorder(), emptyReq)
	assert.True(t, nextCalled)
}
