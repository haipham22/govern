package redis_test

import (
	"testing"

	"github.com/haipham22/govern/database/redis"
)

func TestNew(t *testing.T) {
	client, cleanup, err := redis.New("localhost:6379")

	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("New() expected non-nil client")
	}

	if cleanup == nil {
		t.Fatal("New() expected non-nil cleanup function")
	}

	cleanup()
}

func TestCleanup(t *testing.T) {
	client, cleanup, err := redis.New("localhost:6379")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if cleanup == nil {
		t.Fatal("expected non-nil cleanup function")
	}

	cleanup()

	// Calling cleanup again should be safe
	cleanup()

	_ = client
}

func TestDefaults(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue interface{}
		field        string
	}{
		{"DefaultPoolSize", 100, "PoolSize"},
		{"DefaultMinIdle", 10, "MinIdleConns"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.field {
			case "PoolSize":
				if got := redis.DefaultPoolSize; got != tt.defaultValue.(int) {
					t.Errorf("DefaultPoolSize = %v, want %v", got, tt.defaultValue)
				}
			case "MinIdleConns":
				if got := redis.DefaultMinIdle; got != tt.defaultValue.(int) {
					t.Errorf("DefaultMinIdle = %v, want %v", got, tt.defaultValue)
				}
			}
		})
	}
}
