package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoad(t *testing.T) {
	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  host: "0.0.0.0"
  port: 8080
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Host string `validate:"required"`
			Port int    `validate:"required,min=1,max=65535"`
		} `validate:"required"`
	}

	cfg, err := Load[Config](yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadWithENVOverride(t *testing.T) {
	// Set ENV var
	os.Setenv("SERVER_PORT", "9090")
	defer os.Unsetenv("SERVER_PORT")

	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  port: 8080
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Port int `validate:"required,min=1,max=65535"`
		} `validate:"required"`
	}

	cfg, err := Load[Config](yamlPath)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Server.Port, "ENV should override YAML value")
}

func TestLoadWithENVPrefix(t *testing.T) {
	// Set ENV var with prefix
	os.Setenv("APP_SERVER_PORT", "9090")
	defer os.Unsetenv("APP_SERVER_PORT")

	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  port: 8080
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Port int `validate:"required"`
		} `validate:"required"`
	}

	cfg, err := LoadWithOptions[Config](yamlPath, WithENVPrefix("APP"))
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Server.Port)
}

func TestLoadValidationRequired(t *testing.T) {
	// Create test YAML file with missing required field
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  host: "localhost"
  # port is missing - required field
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Host string `validate:"required"`
			Port int    `validate:"required,min=1,max=65535"`
		} `validate:"required"`
	}

	_, err := Load[Config](yamlPath)
	assert.Error(t, err, "Should fail validation for missing required field")
	assert.Contains(t, err.Error(), "validate")
}

func TestLoadValidationMinMax(t *testing.T) {
	// Create test YAML file with port out of range
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  port: 99999
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Port int `validate:"required,min=1,max=65535"`
		} `validate:"required"`
	}

	_, err := Load[Config](yamlPath)
	assert.Error(t, err, "Should fail validation for port > 65535")
	assert.Contains(t, err.Error(), "validate")
}

func TestLoadFileNotFound(t *testing.T) {
	yamlPath := "/nonexistent/config.yaml"

	type Config struct {
		Port int `validate:"required"`
	}

	_, err := Load[Config](yamlPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadInvalidYAML(t *testing.T) {
	// Create test YAML file with invalid syntax
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  port: [invalid
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Port int
		}
	}

	_, err := Load[Config](yamlPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadWithLogger(t *testing.T) {
	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: 8080`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Port int `validate:"required"`
	}

	logger := zap.NewNop()
	cfg, err := LoadWithOptions[Config](yamlPath, WithLogger(logger))
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestLoadNestedConfig(t *testing.T) {
	// Create test YAML file with nested structure
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  host: "localhost"
  port: 8080
postgres:
  host: "db.local"
  port: 5432
  database: "mydb"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Host string `validate:"required"`
			Port int    `validate:"required"`
		} `validate:"required"`
		Postgres struct {
			Host     string `validate:"required"`
			Port     int    `validate:"required"`
			Database string `validate:"required"`
		} `validate:"required"`
	}

	cfg, err := Load[Config](yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "db.local", cfg.Postgres.Host)
	assert.Equal(t, 5432, cfg.Postgres.Port)
	assert.Equal(t, "mydb", cfg.Postgres.Database)
}

func TestLoadENVOverrideNested(t *testing.T) {
	// Set ENV vars for nested config
	os.Setenv("POSTGRES_HOST", "prod.db.local")
	os.Setenv("POSTGRES_PORT", "5433")
	defer os.Unsetenv("POSTGRES_HOST")
	defer os.Unsetenv("POSTGRES_PORT")

	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `postgres:
  host: "localhost"
  port: 5432
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Postgres struct {
			Host string `validate:"required"`
			Port int    `validate:"required"`
		} `validate:"required"`
	}

	cfg, err := Load[Config](yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "prod.db.local", cfg.Postgres.Host, "ENV should override nested host")
	assert.Equal(t, 5433, cfg.Postgres.Port, "ENV should override nested port")
}

func TestLoadRace(t *testing.T) {
	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: 8080`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Port int `validate:"required"`
	}

	// Run multiple loads concurrently to test for race conditions
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = Load[Config](yamlPath)
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		cfg, err := Load[Config](yamlPath)
		require.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port)
	}

	<-done
}

func ExampleLoad() {
	// Define your config structure
	type Config struct {
		Server struct {
			Host string `validate:"required"`
			Port int    `validate:"min=1,max=65535"`
		} `validate:"required"`
	}

	// Load from YAML file (with ENV override)
	cfg, err := Load[Config]("./config.yaml")
	if err != nil {
		panic(err)
	}

	// Use the config
	fmt.Println(cfg.Server.Host, cfg.Server.Port)
}
