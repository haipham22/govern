package redisstream

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb    redis.UniversalClient
	config Config
	owned  bool
}

func New(ctx context.Context, config Config) (*Client, error) {
	config.applyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	rdb, err := newRedisClient(config)
	if err != nil {
		return nil, err
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redisstream: ping: %w", err)
	}

	return &Client{rdb: rdb, config: config, owned: true}, nil
}

// newRedisClient builds the client from whichever address shape the config
// carries. A URL goes through redis.ParseURL — that is where scheme (rediss),
// credentials and db index are decoded — and the tuning fields are applied on
// top of it, because they are ours and not part of the address.
func newRedisClient(config Config) (redis.UniversalClient, error) {
	if config.URL != "" {
		options, err := redis.ParseURL(config.URL)
		if err != nil {
			return nil, fmt.Errorf("redisstream: parse url: %w", err)
		}
		options.DialTimeout = config.DialTimeout
		options.ReadTimeout = config.ReadTimeout
		options.WriteTimeout = config.WriteTimeout
		if config.PoolSize > 0 {
			options.PoolSize = config.PoolSize
		}
		return redis.NewClient(options), nil
	}

	return redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        splitAddrs(config.Addr),
		Username:     config.Username,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
	}), nil
}

func NewFailFast(ctx context.Context, config Config) *Client {
	client, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return client
}

func NewWithClient(rdb redis.UniversalClient, config Config) *Client {
	config.applyDefaults()
	return &Client{rdb: rdb, config: config}
}

func (c *Client) Redis() redis.UniversalClient {
	return c.rdb
}

func (c *Client) Namespace() string {
	return c.config.Namespace
}

func (c *Client) Close() error {
	if !c.owned {
		return nil
	}
	return c.rdb.Close()
}

// splitAddrs splits a comma-separated address list, dropping blanks.
func splitAddrs(input string) []string {
	var addrs []string
	for addr := range strings.SplitSeq(input, ",") {
		if addr = strings.TrimSpace(addr); addr != "" {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

func (c *Client) HealthCheck() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := c.rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redisstream: health: %w", err)
		}
		return nil
	}
}
