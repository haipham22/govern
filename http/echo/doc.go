// Package echo provides integration between the Echo web framework and Govern's HTTP server.
//
// This package allows you to use Echo's powerful routing and middleware while maintaining
// Govern's graceful shutdown and HTTP server management capabilities.
//
// # Basic Usage
//
//	server := echo.NewServer(":8080")
//
//	server.GET("/", func(c echo.Context) error {
//	    return c.String(http.StatusOK, "Hello, World!")
//	})
//
//	server.Start()
//
// # Middleware
//
//	// Add standard Echo middleware
//	server.Use(echoMiddleware)
//
//	// Convert Govern's http.Middleware to Echo
//	server.Use(echo.Middleware(governMiddleware))
//
// # JWT Authentication
//
//	jwtConfig := &jwt.MiddlewareConfig{
//	    Config:         jwtConfig,
//	    TokenExtractor: jwt.DefaultTokenExtractor,
//	    SkipPaths:      []string{"/health", "/login"},
//	}
//
//	server.Use(echo.JWTMiddleware(jwtConfig))
//
// # Graceful Shutdown
//
// The server automatically handles graceful shutdown when interrupted.
// Use context cancellation for clean shutdown of background operations.
//
// # Concurrency
//
// All middleware and handlers are concurrency-safe.
// Use context.Context for cancellation and timeouts in long-running operations.
package echo
