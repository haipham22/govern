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

func TestLoad_ENVOverride(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		yaml           string
		loadOptions    []Option
		expectedPort   int
		wantErr        bool
		setupCleanup   func()
	}{
		{
			name: "ENV override without prefix",
			envVars: map[string]string{
				"SERVER_PORT": "9090",
			},
			yaml: `server:
  port: 8080
`,
			loadOptions:  []Option{},
			expectedPort: 9090,
			wantErr:      false,
		},
		{
			name: "ENV override with prefix",
			envVars: map[string]string{
				"APP_SERVER_PORT": "9090",
			},
			yaml: `server:
  port: 8080
`,
			loadOptions:  []Option{WithENVPrefix("APP")},
			expectedPort: 9090,
			wantErr:      false,
		},
		{
			name: "ENV override nested config",
			envVars: map[string]string{
				"POSTGRES_HOST": "prod.db.local",
				"POSTGRES_PORT": "5433",
			},
			yaml: `postgres:
  host: "localhost"
  port: 5432
`,
			loadOptions:  []Option{},
			expectedPort: 5433,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			// Create test YAML file
			tmpDir := t.TempDir()
			yamlPath := filepath.Join(tmpDir, "config.yaml")
			require.NoError(t, os.WriteFile(yamlPath, []byte(tt.yaml), 0o600))

			// Define config based on test case
			if tt.name == "ENV override nested config" {
				type Config struct {
					Postgres struct {
						Host string `validate:"required"`
						Port int    `validate:"required"`
					} `validate:"required"`
				}

				cfg, err := LoadWithOptions[Config](yamlPath, tt.loadOptions...)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.envVars["POSTGRES_HOST"], cfg.Postgres.Host, "ENV should override nested host")
				assert.Equal(t, tt.expectedPort, cfg.Postgres.Port, "ENV should override nested port")
			} else {
				type Config struct {
					Server struct {
						Port int `validate:"required,min=1,max=65535"`
					} `validate:"required"`
				}

				cfg, err := LoadWithOptions[Config](yamlPath, tt.loadOptions...)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPort, cfg.Server.Port)
			}
		})
	}
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing required field - port",
			yaml: `server:
  host: "localhost"
  # port is missing - required field
`,
			wantErr: true,
			errMsg:  "validate",
		},
		{
			name: "port above maximum value",
			yaml: `server:
  port: 99999
`,
			wantErr: true,
			errMsg:  "validate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			yamlPath := filepath.Join(tmpDir, "config.yaml")
			require.NoError(t, os.WriteFile(yamlPath, []byte(tt.yaml), 0o600))

			type Config struct {
				Server struct {
					Host string `validate:"required"`
					Port int    `validate:"required,min=1,max=65535"`
				} `validate:"required"`
			}

			_, err := Load[Config](yamlPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() string
		wantErr     bool
		expectedErr string
	}{
		{
			name: "file not found",
			setup: func() string {
				return "/nonexistent/config.yaml"
			},
			wantErr:     true,
			expectedErr: "read config",
		},
		{
			name: "invalid YAML syntax",
			setup: func() string {
				tmpDir := t.TempDir()
				yamlPath := filepath.Join(tmpDir, "config.yaml")
				yamlContent := `server:
  port: [invalid
`
				require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))
				return yamlPath
			},
			wantErr:     true,
			expectedErr: "read config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type Config struct {
				Port int `validate:"required"`
			}

			_, err := Load[Config](tt.setup())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestLoadWithLogger(t *testing.T) {
	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `test:
  value: "debug"`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Test struct {
			Value string `validate:"required"`
		} `validate:"required"`
	}

	logger := zap.NewNop()
	cfg, err := LoadWithOptions[Config](yamlPath, WithLogger(logger))
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.Test.Value)
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

func TestLoadRace(t *testing.T) {
	// Create test YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `test:
  value: "debug"`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Test struct {
			Value string `validate:"required"`
		} `validate:"required"`
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
		assert.Equal(t, "debug", cfg.Test.Value)
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

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		envContent     string
		loadOptions    []Option
		wantErr        bool
		errMsg         string
		expectedHost   string
		expectedPort   int
	}{
		{
			name: "successful load from .env",
			envContent: `SERVER_HOST=localhost
SERVER_PORT=8080
`,
			loadOptions:  []Option{},
			wantErr:      false,
			expectedHost: "localhost",
			expectedPort: 8080,
		},
		{
			name: "validation fails for port above maximum",
			envContent: `SERVER_HOST=localhost
SERVER_PORT=99999
`,
			loadOptions:  []Option{},
			wantErr:      true,
			errMsg:       "validate",
		},
		{
			name: "load from .env with prefix",
			envContent: `HOST=localhost
PORT=8080
`,
			loadOptions:  []Option{WithENVPrefix("SERVER")},
			wantErr:      false,
			expectedHost: "localhost",
			expectedPort: 8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envPath := filepath.Join(tmpDir, ".env")
			require.NoError(t, os.WriteFile(envPath, []byte(tt.envContent), 0o600))

			if tt.name == "load from .env with prefix" {
				type PrefixConfig struct {
					Host string `mapstructure:"host" validate:"required"`
					Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
				}

				cfg, err := LoadFromEnvWithOptions[PrefixConfig](envPath, tt.loadOptions...)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMsg != "" {
						assert.Contains(t, err.Error(), tt.errMsg)
					}
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.expectedHost, cfg.Host)
				assert.Equal(t, tt.expectedPort, cfg.Port)
			} else {
				type Config struct {
					ServerHost string `mapstructure:"server_host" validate:"required"`
					ServerPort int    `mapstructure:"server_port" validate:"required,min=1,max=65535"`
				}

				cfg, err := LoadFromEnvWithOptions[Config](envPath, tt.loadOptions...)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMsg != "" {
						assert.Contains(t, err.Error(), tt.errMsg)
					}
					return
				}

				require.NoError(t, err)
				assert.Equal(t, tt.expectedHost, cfg.ServerHost)
				assert.Equal(t, tt.expectedPort, cfg.ServerPort)
			}
		})
	}
}

func TestLoadWithEnvFile(t *testing.T) {
	// Create test .env file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	envContent := `DATABASE_HOST=localhost
DATABASE_PORT=5432
`
	require.NoError(t, os.WriteFile(envPath, []byte(envContent), 0o600))

	// Create test YAML file
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `server:
  host: "0.0.0.0"
  port: 8080
database:
  host: "default-db"
  port: 3306
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o600))

	type Config struct {
		Server struct {
			Host string `validate:"required"`
			Port int    `validate:"required,min=1,max=65535"`
		} `validate:"required"`
		Database struct {
			Host string `validate:"required"`
			Port int    `validate:"required,min=1,max=65535"`
		} `validate:"required"`
	}

	// Load with .env file override
	cfg, err := LoadWithOptions[Config](yamlPath, WithEnvFile(envPath))
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host, "server config from YAML")
	assert.Equal(t, 8080, cfg.Server.Port, "server port from YAML")
	assert.Equal(t, "localhost", cfg.Database.Host, "database host from .env")
	assert.Equal(t, 5432, cfg.Database.Port, "database port from .env")
}
