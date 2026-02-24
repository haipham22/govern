package echo

import (
	"github.com/haipham22/govern/http/jwt"
)

// ExampleWithSwaggerAuth_bearerToken shows how to configure Swagger UI
// with Bearer token authentication (JWT).
//
// In your main.go, add these annotations for swag:
//
//	// @securityDefinitions.apikey Bearer
//	// @in header
//	// @name Authorization
//	// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"
func ExampleWithSwaggerAuth_bearerToken() {
	jwtConfig := &jwt.MiddlewareConfig{
		Config: &jwt.Config{
			Secret: "your-secret-key",
		},
		SkipPaths: []string{"/health", "/swagger/*"},
	}

	server := NewServer(":8080",
		WithJWT(jwtConfig),
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerAuth(&SwaggerAuth{
				Type:        "Bearer",
				Description: "JWT token",
				Name:        "Authorization",
				In:          "header",
			}),
		),
	)

	_ = server
}

// ExampleWithSwaggerAuth_apiKey shows API key authentication.
func ExampleWithSwaggerAuth_apiKey() {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerAuth(&SwaggerAuth{
				Type:        "ApiKey",
				Description: "API key authentication",
				Name:        "X-API-Key",
				In:          "header",
			}),
		),
	)

	_ = server
}

// ExampleWithSwaggerAuth_basicAuth shows Basic authentication.
func ExampleWithSwaggerAuth_basicAuth() {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerAuth(&SwaggerAuth{
				Type:        "Basic",
				Description: "Basic authentication",
			}),
		),
	)

	_ = server
}

// ExampleWithSwaggerAuth_oauth2 shows OAuth2 authentication.
func ExampleWithSwaggerAuth_oauth2() {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerAuth(&SwaggerAuth{
				Type:             "OAuth2",
				Flow:             "accessCode",
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
				Scopes: map[string]string{
					"read":  "Read access",
					"write": "Write access",
				},
			}),
		),
	)

	_ = server
}
