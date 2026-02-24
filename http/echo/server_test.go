package echo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	httpEcho "github.com/haipham22/govern/http/echo"
)

func TestNewServer(t *testing.T) {
	server := httpEcho.NewServer(":0")
	assert.NotNil(t, server)
	assert.NotNil(t, server.Echo())
}

func TestEchoRoutes(t *testing.T) {
	// Use Echo's test server for proper testing
	server := httpEcho.NewServer(":0")
	e := server.Echo()

	server.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Use httptest for request testing
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestRouteGrouping(t *testing.T) {
	server := httpEcho.NewServer(":0")
	e := server.Echo()

	api := server.Group("/api")
	api.GET("/users", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []string{"user1", "user2"})
	})

	req := httptest.NewRequest("GET", "/api/users", http.NoBody)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "user1")
}
