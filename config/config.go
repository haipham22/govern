package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Load reads YAML file, overrides with ENV variables, validates struct.
// Uses generic type T - user defines their own config struct.
//
// Example:
//
//	type Config struct {
//	    Server struct {
//	        Port int `validate:"required,min=1,max=65535"`
//	    } `validate:"required"`
//	}
//
//	cfg, err := config.Load[Config]("./config.yaml")
func Load[T any](path string) (*T, error) {
	return LoadWithOptions[T](path)
}

// LoadWithOptions reads YAML file, overrides with ENV, validates struct
// with custom options.
func LoadWithOptions[T any](path string, opts ...Option) (*T, error) {
	l := &loader{
		v:        viper.New(),
		logger:   zap.NewNop(),
		validate: validator.New(),
	}

	for _, opt := range opts {
		opt(l)
	}

	// Read YAML file
	l.v.SetConfigFile(path)
	if err := l.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Bind ENV vars (flatten: POSTGRES_HOST -> postgres.host)
	if l.envPrefix != "" {
		l.v.SetEnvPrefix(l.envPrefix)
	}
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	l.v.AutomaticEnv()

	// Unmarshal
	var cfg T
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Validate
	if err := l.validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return &cfg, nil
}

type loader struct {
	v         *viper.Viper
	logger    *zap.Logger
	validate  *validator.Validate
	envPrefix string
}

// Option configures the loader.
type Option func(*loader)

// WithENVPrefix sets ENV variable prefix.
// For example, WithENVPrefix("APP") will look for APP_SERVER_PORT
// instead of SERVER_PORT.
func WithENVPrefix(prefix string) Option {
	return func(l *loader) {
		l.envPrefix = prefix
	}
}

// WithLogger sets custom logger for debug output.
func WithLogger(logger *zap.Logger) Option {
	return func(l *loader) {
		l.logger = logger
	}
}
