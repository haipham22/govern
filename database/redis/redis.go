package redis

import (
	"github.com/redis/go-redis/v9"
)

const (
	DefaultPoolSize = 100
	DefaultMinIdle  = 10
)

// New creates a new Redis client.
// It returns the client, a cleanup function to close the connection, and an error.
func New(addr string, opts ...Option) (*redis.Client, func(), error) {
	return NewClient(addr, opts...)
}

// NewClient creates a new Redis client with an explicit address.
// Use NewFromDSN for DSN string parsing.
func NewClient(addr string, opts ...Option) (*redis.Client, func(), error) {
	cfg := defaultConfig()
	cfg.Addr = addr

	for _, opt := range opts {
		opt(cfg)
	}

	client := redis.NewClient(&redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxRetries:      cfg.MaxRetries,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		MinRetryBackoff: cfg.MinRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})

	cleanup := func() {
		_ = client.Close()
	}

	return client, cleanup, nil
}

// NewFromDSN creates a new Redis client from a DSN string.
// DSN format: redis://[:password@]host[:port][/db][?options]
func NewFromDSN(dsn string, opts ...Option) (*redis.Client, func(), error) {
	addr, parsedOpts, err := ParseDSN(dsn)
	if err != nil {
		return nil, nil, err
	}

	// Explicit opts override parsed opts
	parsedOpts = append(parsedOpts, opts...)
	return NewClient(addr, parsedOpts...)
}
