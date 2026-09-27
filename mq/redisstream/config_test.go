package redisstream

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestConfig_ApplyDefaultsFillsZeroValues(t *testing.T) {
	config := Config{}
	config.applyDefaults()

	assert.Equal(t, "localhost:6379", config.Addr)
	assert.Equal(t, DefaultNamespace, config.Namespace)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 10*time.Second, config.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.WriteTimeout)
}

func TestConfig_ApplyDefaultsKeepsExplicitValues(t *testing.T) {
	config := Config{Addr: "redis:6380", Namespace: "billing", ReadTimeout: time.Minute}
	config.applyDefaults()

	assert.Equal(t, "redis:6380", config.Addr)
	assert.Equal(t, "billing", config.Namespace)
	assert.Equal(t, time.Minute, config.ReadTimeout)
}

func TestConfig_ValidateRequiresAddr(t *testing.T) {
	assert.ErrorIs(t, (&Config{}).Validate(), ErrMissingAddr)
	assert.NoError(t, (&Config{Addr: "localhost:6379"}).Validate())
}

func TestStreamConfig_ApplyDefaultsAndValidate(t *testing.T) {
	config := StreamConfig{Name: "orders"}
	config.applyDefaults()

	assert.Equal(t, int64(DefaultMaxLen), config.MaxLen)
	assert.NoError(t, config.Validate())

	assert.ErrorIs(t, (&StreamConfig{}).Validate(), ErrMissingStreamName)
}

func TestConsumerConfig_ApplyDefaultsFillsZeroValues(t *testing.T) {
	config := ConsumerConfig{Stream: "orders", Group: "g1"}
	config.applyDefaults()

	assert.NotEmpty(t, config.ConsumerID)
	assert.Equal(t, int64(DefaultBatchSize), config.BatchSize)
	assert.Equal(t, DefaultBlockTimeout, config.BlockTimeout)
	assert.Equal(t, DefaultMaxProcessingTime, config.MaxProcessingTime)
	assert.Equal(t, DefaultBufferSize, config.BufferSize)
	assert.Equal(t, DefaultShutdownTimeout, config.ShutdownTimeout)
}

func TestConsumerConfig_ApplyDefaultsKeepsExplicitValues(t *testing.T) {
	config := ConsumerConfig{
		Stream:       "orders",
		Group:        "g1",
		ConsumerID:   "fixed-id",
		BatchSize:    50,
		BlockTimeout: time.Second,
		BufferSize:   7,
	}
	config.applyDefaults()

	assert.Equal(t, "fixed-id", config.ConsumerID)
	assert.Equal(t, int64(50), config.BatchSize)
	assert.Equal(t, time.Second, config.BlockTimeout)
	assert.Equal(t, 7, config.BufferSize)
}

func TestConsumerConfig_ValidateReportsMissingFields(t *testing.T) {
	assert.ErrorIs(t, (&ConsumerConfig{Group: "g1"}).Validate(), ErrMissingStreamName)
	assert.ErrorIs(t, (&ConsumerConfig{Stream: "orders"}).Validate(), ErrMissingGroup)
	assert.NoError(t, (&ConsumerConfig{Stream: "orders", Group: "g1"}).Validate())
}

// A URL answers for the address, so the localhost fallback must not fire: two
// addresses in one config and only New would know which one won.
func TestConfig_ApplyDefaultsLeavesAddrEmptyWhenURLSet(t *testing.T) {
	config := Config{URL: "redis://cache:6379/2"}
	config.applyDefaults()

	assert.Empty(t, config.Addr)
	assert.Equal(t, DefaultNamespace, config.Namespace)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
}

func TestConfig_ValidateAcceptsURLAlone(t *testing.T) {
	config := Config{URL: "redis://cache:6379/2"}
	assert.NoError(t, config.Validate())
}

func TestConfig_ValidateRejectsNeitherAddrNorURL(t *testing.T) {
	config := Config{}
	assert.ErrorIs(t, config.Validate(), ErrMissingAddr)
}

// The URL decides scheme, credentials and db index; the timeouts stay ours,
// because they tune the client rather than name the server.
func TestNewRedisClient_URLDecidesAddressTuningStaysConfig(t *testing.T) {
	config := Config{URL: "redis://user:pw@cache:6380/3", ReadTimeout: 7 * time.Second, PoolSize: 42}
	config.applyDefaults()

	client, err := newRedisClient(config)
	assert.NoError(t, err)

	options := client.(*redis.Client).Options()
	assert.Equal(t, "cache:6380", options.Addr)
	assert.Equal(t, "user", options.Username)
	assert.Equal(t, "pw", options.Password)
	assert.Equal(t, 3, options.DB)
	assert.Equal(t, 7*time.Second, options.ReadTimeout)
	assert.Equal(t, 42, options.PoolSize)
}

func TestNewRedisClient_RejectsUnparseableURL(t *testing.T) {
	_, err := newRedisClient(Config{URL: "://not a url"})
	assert.Error(t, err)
}
