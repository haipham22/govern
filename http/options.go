package http

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// WithTimeout sets server timeouts.
func WithTimeout(read, write, idle time.Duration) ServerOption {
	return func(s *Server) {
		s.server.ReadTimeout = read
		s.server.WriteTimeout = write
		s.server.IdleTimeout = idle
	}
}

// WithReadHeaderTimeout sets read header timeout.
func WithReadHeaderTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.server.ReadHeaderTimeout = timeout
	}
}

// WithShutdownTimeout sets the maximum time to wait for connections to close.
func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *zap.SugaredLogger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithMaxHeaderBytes sets max header bytes.
func WithMaxHeaderBytes(bytes int) ServerOption {
	return func(s *Server) {
		s.server.MaxHeaderBytes = bytes
	}
}

// WithBaseContext sets base context.
func WithBaseContext(fn func(net.Listener) context.Context) ServerOption {
	return func(s *Server) {
		s.server.BaseContext = fn
	}
}

// WithTLS configures TLS.
func WithTLS(certFile, keyFile string) ServerOption {
	return func(s *Server) {
		s.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		// Note: CertFile and KeyFile are not part of http.Server struct
		// They're passed directly to ListenAndServeTLS
	}
}

// WithHTTPServerOptions sets additional http.Server options.
func WithHTTPServerOptions(opts ...func(*http.Server)) ServerOption {
	return func(s *Server) {
		for _, opt := range opts {
			opt(s.server)
		}
	}
}
