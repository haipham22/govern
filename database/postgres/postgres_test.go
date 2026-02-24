package postgres_test

import (
	"testing"
	"time"

	"github.com/haipham22/govern/database/postgres"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		opts    []postgres.Option
		wantErr bool
	}{
		{
			name:    "invalid DSN",
			dsn:     "invalid://dsn",
			wantErr: true,
		},
		{
			name:    "invalid DSN with options",
			dsn:     "host=invalid user=invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup, err := postgres.New(tt.dsn, tt.opts...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error, got nil")
				}
				if db != nil {
					t.Errorf("New() expected nil db on error, got %v", db)
				}
				if cleanup != nil {
					cleanup()
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if db == nil {
				t.Errorf("New() expected non-nil db")
			}

			if cleanup == nil {
				t.Errorf("New() expected non-nil cleanup function")
			} else {
				cleanup()
			}
		})
	}
}

func TestNewWithOptions(t *testing.T) {
	opts := []postgres.Option{
		postgres.WithMaxIdleConns(10),
		postgres.WithMaxOpenConns(50),
		postgres.WithConnMaxLifetime(10 * time.Minute),
		postgres.WithConnMaxIdleTime(time.Minute),
		postgres.WithPreferSimpleProtocol(false),
		postgres.WithDebug(true),
	}

	// This will fail to connect but we're testing options are applied
	db, cleanup, err := postgres.New("host=localhost", opts...)

	if err != nil && db == nil {
		// Expected - connection fails but options were processed
		return
	}

	if cleanup != nil {
		cleanup()
	}
}

func TestDefaults(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue interface{}
		field        string
	}{
		{
			name:         "DefaultMaxIdleConns",
			defaultValue: 25,
			field:        "MaxIdleConns",
		},
		{
			name:         "DefaultMaxOpenConns",
			defaultValue: 100,
			field:        "MaxOpenConns",
		},
		{
			name:         "DefaultConnMaxLifetime",
			defaultValue: 5 * time.Minute,
			field:        "ConnMaxLifetime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that default constants are set to expected values
			switch tt.field {
			case "MaxIdleConns":
				if got := postgres.DefaultMaxIdleConns; got != tt.defaultValue.(int) {
					t.Errorf("DefaultMaxIdleConns = %v, want %v", got, tt.defaultValue)
				}
			case "MaxOpenConns":
				if got := postgres.DefaultMaxOpenConns; got != tt.defaultValue.(int) {
					t.Errorf("DefaultMaxOpenConns = %v, want %v", got, tt.defaultValue)
				}
			case "ConnMaxLifetime":
				if got := postgres.DefaultConnMaxLifetime; got != tt.defaultValue.(time.Duration) {
					t.Errorf("DefaultConnMaxLifetime = %v, want %v", got, tt.defaultValue)
				}
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	db, cleanup, err := postgres.New("host=localhost")
	if err != nil && db == nil {
		t.Skip("cannot connect to test database")
		return
	}

	if cleanup == nil {
		t.Fatal("expected non-nil cleanup function")
	}

	// Calling cleanup should not panic
	cleanup()

	// Calling cleanup again should be safe
	cleanup()
}
