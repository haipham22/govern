package redis_test

import (
	"testing"

	"github.com/haipham22/govern/database/redis"
)

func TestClusterOptions(t *testing.T) {
	tests := []struct {
		name   string
		option redis.Option
		verify func(*redis.Config) bool
	}{
		{
			name:   "WithAddrs sets multiple addresses",
			option: redis.WithAddrs("node1:6379", "node2:6379", "node3:6379"),
			verify: func(cfg *redis.Config) bool {
				return len(cfg.Addrs) == 3 &&
					cfg.Addrs[0] == "node1:6379" &&
					cfg.Addrs[1] == "node2:6379" &&
					cfg.Addrs[2] == "node3:6379"
			},
		},
		{
			name:   "WithAddrs sets single address",
			option: redis.WithAddrs("localhost:6379"),
			verify: func(cfg *redis.Config) bool {
				return len(cfg.Addrs) == 1 && cfg.Addrs[0] == "localhost:6379"
			},
		},
		{
			name:   "WithRouteByLatency true",
			option: redis.WithRouteByLatency(true),
			verify: func(cfg *redis.Config) bool {
				return cfg.RouteByLatency == true
			},
		},
		{
			name:   "WithRouteByLatency false",
			option: redis.WithRouteByLatency(false),
			verify: func(cfg *redis.Config) bool {
				return cfg.RouteByLatency == false
			},
		},
		{
			name:   "WithRouteRandomly true",
			option: redis.WithRouteRandomly(true),
			verify: func(cfg *redis.Config) bool {
				return cfg.RouteRandomly == true
			},
		},
		{
			name:   "WithRouteRandomly false",
			option: redis.WithRouteRandomly(false),
			verify: func(cfg *redis.Config) bool {
				return cfg.RouteRandomly == false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &redis.Config{}
			tt.option(cfg)

			if !tt.verify(cfg) {
				t.Errorf("Option did not set expected value")
			}
		})
	}
}
