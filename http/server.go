package http

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/haipham22/govern/graceful"
	"github.com/haipham22/govern/log"
)

// Middleware function type
type Middleware func(http.Handler) http.Handler

// Server wraps an http.Server with graceful shutdown capabilities.
type Server struct {
	server          *http.Server
	shutdownTimeout time.Duration
	logger          *zap.SugaredLogger
	manager         *graceful.Manager
	middlewares     []Middleware
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// NewServer creates a new Server with graceful shutdown support.
func NewServer(addr string, handler http.Handler, opts ...ServerOption) *Server {
	s := &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
		},
		shutdownTimeout: 30 * time.Second,
		logger:          log.Default(),
		middlewares:     []Middleware{},
	}

	for _, opt := range opts {
		opt(s)
	}

	s.manager = graceful.NewManager(context.Background())

	return s
}

// Use adds middleware to the server.
func (s *Server) Use(middleware ...Middleware) {
	s.middlewares = append(s.middlewares, middleware...)
}

// buildHandler builds the final handler with middleware chain.
func (s *Server) buildHandler() http.Handler {
	handler := s.server.Handler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}
	return handler
}

// Start begins serving and blocks until the server is shut down gracefully.
func (s *Server) Start() error {
	// Build handler with middleware
	s.server.Handler = s.buildHandler()

	// Register server shutdown as cleanup
	s.manager.Defer(func(ctx context.Context) error {
		s.logger.Info("Shutting down HTTP server")
		return s.server.Shutdown(ctx)
	})

	// Start server in managed goroutine
	s.manager.Go(func(ctx context.Context) error {
		s.logger.Infow("Server starting", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorw("Server error", "error", err)
			return err
		}
		return nil
	})

	// Wait for shutdown signal
	if err := s.manager.Wait(); err != nil {
		return err
	}

	return s.manager.Shutdown(s.shutdownTimeout)
}

// Shutdown triggers a graceful shutdown programmatically.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Server returns the underlying http.Server for direct access if needed.
func (s *Server) Server() *http.Server {
	return s.server
}

// Listen returns the listener for testing purposes.
func (s *Server) Listen() (net.Listener, error) {
	return net.Listen("tcp", s.server.Addr)
}
