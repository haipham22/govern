package echo_test

import (
	"net/http"

	"github.com/labstack/echo/v4"

	echopackage "github.com/haipham22/govern/http/echo"
	"github.com/haipham22/govern/http/jwt"
)

// Example_basicEcho demonstrates basic Echo server usage.
func Example_basicEcho() {
	server := echopackage.NewServer(":8080")

	server.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	server.POST("/users", func(c echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{
			"message": "User created",
		})
	})

	// In production, call server.Start() to begin serving
}

// Example_echoWithJWT demonstrates Echo with JWT authentication.
func Example_echoWithJWT() {
	server := echopackage.NewServer(":8080")

	// Public routes
	server.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// JWT configuration
	jwtConfig := &jwt.MiddlewareConfig{
		Config:         jwt.DefaultConfig(),
		TokenExtractor: jwt.DefaultTokenExtractor,
		SkipPaths:      []string{"/health", "/login"},
	}

	// Protected routes
	api := server.Group("/api", echopackage.JWTMiddleware(jwtConfig))
	api.GET("/profile", getProfile)

	// In production, call server.Start() to begin serving
}

// Example_echoWithMiddleware demonstrates Echo with custom middleware.
func Example_echoWithMiddleware() {
	server := echopackage.NewServer(":8080")

	// Add custom middleware
	server.Use(echoMiddleware)

	// Routes
	server.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// In production, call server.Start() to begin serving
}

func getProfile(c echo.Context) error {
	claims := echopackage.MustGetCurrentUser(c)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"email":    claims.Email,
	})
}

func echoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Custom middleware logic
		return next(c)
	}
}
