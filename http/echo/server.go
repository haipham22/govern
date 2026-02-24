package echo

import (
	"github.com/labstack/echo/v4"

	"github.com/haipham22/govern/http"
)

// Server wraps Echo with Govern's graceful shutdown.
type Server struct {
	*http.Server
	echo *echo.Echo
}

// ServerOption configures Echo server.
type ServerOption func(*Server)

// NewServer creates Echo server with Govern integration.
func NewServer(addr string, opts ...ServerOption) *Server {
	e := echo.New()

	// Create Govern server with Echo as handler
	governSrv := http.NewServer(addr, e)

	return &Server{
		Server: governSrv,
		echo:   e,
	}
}

// Echo returns the underlying Echo instance.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Use adds Echo middleware.
func (s *Server) Use(middleware ...echo.MiddlewareFunc) {
	s.echo.Use(middleware...)
}

// Pre adds middleware to pre-route.
func (s *Server) Pre(middleware ...echo.MiddlewareFunc) {
	s.echo.Pre(middleware...)
}

// GET registers GET route.
func (s *Server) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.GET(path, h, m...)
}

// POST registers POST route.
func (s *Server) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.POST(path, h, m...)
}

// PUT registers PUT route.
func (s *Server) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.PUT(path, h, m...)
}

// DELETE registers DELETE route.
func (s *Server) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.DELETE(path, h, m...)
}

// PATCH registers PATCH route.
func (s *Server) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.PATCH(path, h, m...)
}

// HEAD registers HEAD route.
func (s *Server) HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.HEAD(path, h, m...)
}

// OPTIONS registers OPTIONS route.
func (s *Server) OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.echo.OPTIONS(path, h, m...)
}

// Group creates route group.
func (s *Server) Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group {
	return s.echo.Group(prefix, m...)
}

// Static serves static files.
func (s *Server) Static(prefix, root string) {
	s.echo.Static(prefix, root)
}
