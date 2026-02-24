package echo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithEchoSwagger_DefaultDisabled(t *testing.T) {
	server := NewServer(":8080",
		WithEchoSwagger(),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
	assert.NotNil(t, server.echo)
}

func TestWithEchoSwagger_ExplicitlyDisabled(t *testing.T) {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(false),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestWithEchoSwagger_EnabledDefaults(t *testing.T) {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestWithEchoSwagger_CustomPath(t *testing.T) {
	customPath := "/api/docs/*"
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerPath(customPath),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestWithEchoSwagger_WithInfo(t *testing.T) {
	customInfo := &SwaggerInfo{
		Title:       "Custom API",
		Description: "Custom description",
		Version:     "2.0",
	}

	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerInfo(customInfo),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestWithEchoSwagger_WithAuth(t *testing.T) {
	auth := &SwaggerAuth{
		Type:        "Bearer",
		Description: "JWT token",
		Name:        "Authorization",
		In:          "header",
	}

	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerAuth(auth),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestWithEchoSwagger_MultipleOptions(t *testing.T) {
	server := NewServer(":8080",
		WithEchoSwagger(
			WithSwaggerEnabled(true),
			WithSwaggerPath("/docs/*"),
			WithSwaggerInfo(&SwaggerInfo{
				Title:       "Test API",
				Description: "Test",
				Version:     "1.0",
			}),
		),
	)

	// Verify server was created successfully
	assert.NotNil(t, server)
}

func TestDefaultSwaggerInfo_Values(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected interface{}
	}{
		{"Title", "Title", "API Documentation"},
		{"Description", "Description", "This is a sample server."},
		{"Version", "Version", "1.0"},
		{"Host", "Host", "localhost:8080"},
		{"BasePath", "BasePath", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.field {
			case "Title":
				assert.Equal(t, tt.expected, DefaultSwaggerInfo.Title)
			case "Description":
				assert.Equal(t, tt.expected, DefaultSwaggerInfo.Description)
			case "Version":
				assert.Equal(t, tt.expected, DefaultSwaggerInfo.Version)
			case "Host":
				assert.Equal(t, tt.expected, DefaultSwaggerInfo.Host)
			case "BasePath":
				assert.Equal(t, tt.expected, DefaultSwaggerInfo.BasePath)
			}
		})
	}

	// Verify schemes
	assert.Equal(t, []string{"http", "https"}, DefaultSwaggerInfo.Schemes)
}
