package echo

import (
	"time"

	echoSwagger "github.com/swaggo/echo-swagger"
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

// WithEchoSwagger adds Swagger UI middleware to the Echo server.
//
// Swagger UI provides interactive API documentation at the specified path.
// The documentation is generated from code annotations using the swag tool.
//
// Before using this option:
// 1. Install swag: go install github.com/swaggo/swag/cmd/swag@latest
// 2. Add Swagger annotations to your handlers
// 3. Generate docs: swag init
// 4. Import docs package: import _ "your-project/docs"
//
// Example:
//
//	import _ "myapi/docs"
//
//	server := echo.NewServer(":8080",
//	    echo.WithEchoSwagger(
//	        echo.WithSwaggerEnabled(true),
//	        echo.WithSwaggerPath("/api/docs/*"),
//	        echo.WithSwaggerInfo(&echo.SwaggerInfo{
//	            Title:       "My API",
//	            Description: "Sample API server",
//	            Version:     "1.0",
//	        }),
//	    ),
//	)
//
// Access Swagger UI at: http://localhost:8080/swagger/index.html
//
// With JWT authentication (Bearer token):
//
//	server := echo.NewServer(":8080",
//	    echo.WithJWT(jwtConfig),
//	    echo.WithEchoSwagger(
//	        echo.WithSwaggerEnabled(true),
//	        echo.WithSwaggerAuth(&echo.SwaggerAuth{
//	            Type:        "Bearer",
//	            Description: "JWT token",
//	            Name:        "Authorization",
//	            In:          "header",
//	        }),
//	    ),
//	)
//
// Environment-based enablement:
//
//	enableSwagger := os.Getenv("GO_ENV") == "development"
//	server := echo.NewServer(":8080",
//	    echo.WithEchoSwagger(
//	        echo.WithSwaggerEnabled(enableSwagger),
//	    ),
//	)
//
// Security Warning: Do not expose Swagger UI in production without authentication.
// Always use WithSwaggerEnabled(false) in production or protect with auth middleware.
func WithEchoSwagger(opts ...SwaggerOption) ServerOption {
	return func(s *Server) {
		// Apply settings with defaults
		settings := &swaggerSettings{
			enabled: false, // Default disabled for security
			path:    "/swagger/*",
			info:    DefaultSwaggerInfo,
		}

		for _, opt := range opts {
			opt(settings)
		}

		// Skip if not enabled
		if !settings.enabled {
			return
		}

		// Note: Authentication configuration is handled via swag annotations
		// in your handler code. Use WithSwaggerAuth for documentation purposes
		// and to help generate proper Swagger annotations.
		//
		// Example swag annotations for Bearer auth:
		// // @securityDefinitions.bearer Bearer
		// // @param Authorization header string true "Authorization header"
		//
		// Or in main.go:
		// // @securityDefinitions.apikey Bearer
		// // @in header
		// // @name Authorization
		// // @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"

		// Register Swagger UI route
		s.echo.GET(settings.path, echoSwagger.WrapHandler)
	}
}
