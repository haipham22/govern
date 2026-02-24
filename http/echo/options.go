package echo

import (
	"time"

	"go.uber.org/zap"

	governhttp "github.com/haipham22/govern/http"
	"github.com/haipham22/govern/http/jwt"
)

// WithTimeout sets server timeouts.
func WithTimeout(read, write, idle time.Duration) ServerOption {
	return func(s *Server) {
		opt := governhttp.WithTimeout(read, write, idle)
		opt(s.Server)
	}
}

// WithLogger sets custom logger.
func WithLogger(logger *zap.SugaredLogger) ServerOption {
	return func(s *Server) {
		opt := governhttp.WithLogger(logger)
		opt(s.Server)
	}
}

// WithJWT adds JWT authentication middleware.
func WithJWT(config *jwt.MiddlewareConfig) ServerOption {
	return func(s *Server) {
		s.Use(JWTMiddleware(config))
	}
}
