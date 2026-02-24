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
// # Swagger UI Integration
//
// This package supports Swagger UI for interactive API documentation.
//
// ## Setup
//
// 1. Install swag CLI:
//
//	go install github.com/swaggo/swag/cmd/swag@latest
//
// 2. Add Swagger annotations to your handlers:
//
//	// @Summary Get user by ID
//	// @Description Retrieve user information
//	// @Tags users
//	// @Accept json
//	// @Produce json
//	// @Param id path int true "User ID"
//	// @Success 200 {object} User
//	// @Failure 404 {object} ErrorResponse
//	// @Router /users/{id} [get]
//	func getUser(c echo.Context) error {
//	    // Handler logic
//	}
//
// 3. Generate Swagger docs:
//
//	swag init -g cmd/api/main.go
//
// 4. Import docs package and use WithEchoSwagger option:
//
//	import _ "myapi/docs"  // Generated docs
//
//	server := echo.NewServer(":8080",
//	    echo.WithEchoSwagger(
//	        echo.WithSwaggerEnabled(true),
//	        echo.WithSwaggerInfo(&echo.SwaggerInfo{
//	            Title:       "My API",
//	            Description: "Sample API server",
//	            Version:     "1.0",
//	        }),
//	    ),
//	)
//
// 5. Access Swagger UI at: http://localhost:8080/swagger/index.html
//
// ## Security
//
// Swagger UI should not be exposed in production without authentication.
// Use WithSwaggerEnabled with environment variables:
//
//	enableSwagger := os.Getenv("GO_ENV") == "development"
//	server := echo.NewServer(":8080",
//	    echo.WithEchoSwagger(
//	        echo.WithSwaggerEnabled(enableSwagger),
//	        echo.WithSwaggerPath("/swagger/*"),
//	    ),
//	)
//
// ## With JWT Authentication
//
// To enable Bearer token authentication in Swagger UI:
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
// Add these annotations to main.go:
//
//	// @securityDefinitions.apikey Bearer
//	// @in header
//	// @name Authorization
//	// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"
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
