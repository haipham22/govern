package redis_test

import (
	"testing"

	"github.com/haipham22/govern/database/redis"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		wantAddr string
		wantOpts int // expected number of options
		wantErr  bool
	}{
		{
			name:     "simple address",
			dsn:      "redis://localhost:6379",
			wantAddr: "localhost:6379",
			wantOpts: 0,
		},
		{
			name:     "with password",
			dsn:      "redis://:secret@localhost:6379",
			wantAddr: "localhost:6379",
			wantOpts: 1,
		},
		{
			name:     "with db in path",
			dsn:      "redis://localhost:6379/1",
			wantAddr: "localhost:6379",
			wantOpts: 1,
		},
		{
			name:     "with password and db",
			dsn:      "redis://:secret@localhost:6379/2",
			wantAddr: "localhost:6379",
			wantOpts: 2,
		},
		{
			name:     "with query options",
			dsn:      "redis://localhost:6379?pool_size=50&db=1",
			wantAddr: "localhost:6379",
			wantOpts: 2,
		},
		{
			name:     "with all options",
			dsn:      "redis://:secret@localhost:6379/1?pool_size=50&min_idle=5",
			wantAddr: "localhost:6379",
			wantOpts: 4,
		},
		{
			name:     "without port",
			dsn:      "redis://localhost",
			wantAddr: "localhost",
			wantOpts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, opts, err := redis.ParseDSN(tt.dsn)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDSN() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDSN() unexpected error: %v", err)
				return
			}

			if addr != tt.wantAddr {
				t.Errorf("ParseDSN() addr = %v, want %v", addr, tt.wantAddr)
			}

			if len(opts) != tt.wantOpts {
				t.Errorf("ParseDSN() opts count = %v, want %v", len(opts), tt.wantOpts)
			}
		})
	}
}

func TestNewFromDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "simple DSN",
			dsn:     "redis://localhost:6379",
			wantErr: false,
		},
		{
			name:    "DSN with password and db",
			dsn:     "redis://:secret@localhost:6379/1",
			wantErr: false,
		},
		{
			name:    "DSN with options",
			dsn:     "redis://localhost:6379?pool_size=50&min_idle=5",
			wantErr: false,
		},
		{
			name:    "invalid DSN",
			dsn:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanup, err := redis.NewFromDSN(tt.dsn)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewFromDSN() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewFromDSN() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewFromDSN() expected non-nil client")
			}

			if cleanup == nil {
				t.Error("NewFromDSN() expected non-nil cleanup function")
			} else {
				cleanup()
			}
		})
	}
}
