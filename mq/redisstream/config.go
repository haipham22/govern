package redisstream

import (
	"time"
)

const (
	DefaultNamespace         = "redisstream"
	DefaultMaxLen            = 1000
	DefaultBatchSize         = 10
	DefaultBlockTimeout      = 5 * time.Second
	DefaultMaxProcessingTime = 30 * time.Second
	DefaultBufferSize        = 100
	DefaultShutdownTimeout   = 10 * time.Second
)

type Config struct {
	// URL, when set, IS the connection: Addr, Username, Password and DB below
	// are ignored ("redis://user:pass@host:6379/0", "rediss://..."). It is the
	// shape a managed platform hands you, and the shape a service that only
	// borrows this Redis (a session store, say) usually already has.
	//
	// The timeouts and PoolSize below still apply — they are client tuning, not
	// part of the address.
	URL string `yaml:"url" mapstructure:"url" json:"-"`

	Addr         string        `yaml:"addr" mapstructure:"addr" default:"localhost:6379"`
	Username     string        `yaml:"username" mapstructure:"username" json:"-"`
	Password     string        `yaml:"password" mapstructure:"password" json:"-"`
	DB           int           `yaml:"db" mapstructure:"db"`
	Namespace    string        `yaml:"namespace" mapstructure:"namespace" default:"redisstream"`
	DialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout" default:"5s"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout" default:"10s"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout" default:"3s"`
	PoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`
}

func (config *Config) applyDefaults() {
	// A URL answers for the address, so the localhost fallback must not fire —
	// filling Addr here would leave two addresses in one config and only the
	// reader of New would know which won.
	if config.Addr == "" && config.URL == "" {
		config.Addr = "localhost:6379"
	}
	if config.Namespace == "" {
		config.Namespace = DefaultNamespace
	}
	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 3 * time.Second
	}
}

func (config *Config) Validate() error {
	if config.Addr == "" && config.URL == "" {
		return ErrMissingAddr
	}
	return nil
}

type StreamConfig struct {
	Name string `yaml:"name" mapstructure:"name"`

	MaxLen int64 `yaml:"max_len" mapstructure:"max_len" default:"1000"`
}

func (config *StreamConfig) applyDefaults() {
	if config.MaxLen == 0 {
		config.MaxLen = DefaultMaxLen
	}
}

func (config *StreamConfig) Validate() error {
	if config.Name == "" {
		return ErrMissingStreamName
	}
	return nil
}

type ConsumerConfig struct {
	Stream            string        `yaml:"stream" mapstructure:"stream"`
	Group             string        `yaml:"group" mapstructure:"group"`
	ConsumerID        string        `yaml:"consumer_id" mapstructure:"consumer_id"`
	BatchSize         int64         `yaml:"batch_size" mapstructure:"batch_size" default:"10"`
	BlockTimeout      time.Duration `yaml:"block_timeout" mapstructure:"block_timeout" default:"5s"`
	MaxProcessingTime time.Duration `yaml:"max_processing_time" mapstructure:"max_processing_time" default:"30s"`
	BufferSize        int           `yaml:"buffer_size" mapstructure:"buffer_size" default:"100"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout" default:"10s"`
}

func (config *ConsumerConfig) applyDefaults() {
	if config.ConsumerID == "" {
		config.ConsumerID = newConsumerID()
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.BlockTimeout <= 0 {
		config.BlockTimeout = DefaultBlockTimeout
	}
	if config.MaxProcessingTime <= 0 {
		config.MaxProcessingTime = DefaultMaxProcessingTime
	}
	if config.BufferSize <= 0 {
		config.BufferSize = DefaultBufferSize
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = DefaultShutdownTimeout
	}
}

func (config *ConsumerConfig) Validate() error {
	switch {
	case config.Stream == "":
		return ErrMissingStreamName
	case config.Group == "":
		return ErrMissingGroup
	}
	return nil
}
