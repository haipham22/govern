package graceful

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/haipham22/govern/log"
	"go.uber.org/zap"
)

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithShutdownTimeout sets the maximum time to wait for connections to close.
func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

// WithLogger sets a custom logger.
func WithServerLogger(logger *zap.SugaredLogger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithServerOptions sets additional http.Server options.
func WithServerOptions(opts ...func(*http.Server)) ServerOption {
	return func(s *Server) {
		for _, opt := range opts {
			opt(s.server)
		}
	}
}

// Server wraps an http.Server with graceful shutdown capabilities.
type Server struct {
	server          *http.Server
	shutdownTimeout time.Duration
	logger          *zap.SugaredLogger
	manager         *Manager
}

// NewServer creates a new Server with graceful shutdown support.
func NewServer(addr string, handler http.Handler, opts ...ServerOption) *Server {
	s := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		shutdownTimeout: 30 * time.Second,
		logger:          log.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.manager = NewManager(context.Background())

	return s
}

// Start begins serving and blocks until the server is shut down gracefully.
func (s *Server) Start() error {
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
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
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
