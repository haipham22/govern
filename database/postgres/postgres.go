package postgres

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DefaultMaxIdleConns    = 25
	DefaultMaxOpenConns    = 100
	DefaultConnMaxLifetime = 5 * time.Minute
)

type Option func(*Config)

type Config struct {
	Debug                bool
	MaxIdleConns         int
	MaxOpenConns         int
	ConnMaxLifetime      time.Duration
	ConnMaxIdleTime      time.Duration
	PreferSimpleProtocol bool
	Logger               logger.Interface
}

func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.Debug = debug
	}
}

func WithMaxIdleConns(n int) Option {
	return func(c *Config) {
		c.MaxIdleConns = n
	}
}

func WithMaxOpenConns(n int) Option {
	return func(c *Config) {
		c.MaxOpenConns = n
	}
}

func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxLifetime = d
	}
}

func WithConnMaxIdleTime(d time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxIdleTime = d
	}
}

func WithPreferSimpleProtocol(v bool) Option {
	return func(c *Config) {
		c.PreferSimpleProtocol = v
	}
}

func WithLogger(log logger.Interface) Option {
	return func(c *Config) {
		c.Logger = log
	}
}

// New creates a new GORM PostgreSQL connection.
// It returns the DB instance, a cleanup function to close the connection, and an error.
func New(dsn string, opts ...Option) (*gorm.DB, func(), error) {
	cfg := &Config{
		MaxIdleConns:         DefaultMaxIdleConns,
		MaxOpenConns:         DefaultMaxOpenConns,
		ConnMaxLifetime:      DefaultConnMaxLifetime,
		ConnMaxIdleTime:      5 * time.Minute,
		PreferSimpleProtocol: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	pgCfg := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: cfg.PreferSimpleProtocol,
	}

	gormCfg := &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 cfg.Logger,
	}

	if cfg.Debug && cfg.Logger == nil {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.New(pgCfg), gormCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	cleanup := func() {
		_ = sqlDB.Close()
	}

	return db, cleanup, nil
}
