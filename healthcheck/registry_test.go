package healthcheck

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_Register(t *testing.T) {
	r := New()

	r.Register("check1", func(ctx context.Context) error {
		return nil
	})

	r.Register("check2", func(ctx context.Context) error {
		return errors.New("failed")
	}, WithTimeout(2*time.Second))

	// Should panic on duplicate name
	assert.Panics(t, func() {
		r.Register("check1", func(ctx context.Context) error {
			return nil
		})
	})
}

func TestRegistry_Unregister(t *testing.T) {
	r := New()

	r.Register("check1", func(ctx context.Context) error {
		return nil
	})

	r.Unregister("check1")

	// Should allow re-register after unregister
	assert.NotPanics(t, func() {
		r.Register("check1", func(ctx context.Context) error {
			return nil
		})
	})
}

func TestRegistry_Run_Passing(t *testing.T) {
	r := New()

	r.Register("db", func(ctx context.Context) error {
		return nil
	})

	r.Register("cache", func(ctx context.Context) error {
		return nil
	})

	resp := r.Run(context.Background())
	assert.Equal(t, StatusPassing, resp.Status)
	assert.Len(t, resp.Checks, 2)
	assert.Equal(t, StatusPassing, resp.Checks["db"].Status)
	assert.Equal(t, StatusPassing, resp.Checks["cache"].Status)
}

func TestRegistry_Run_Failing(t *testing.T) {
	r := New()

	r.Register("db", func(ctx context.Context) error {
		return nil
	})

	r.Register("cache", func(ctx context.Context) error {
		return errors.New("connection failed")
	})

	resp := r.Run(context.Background())
	assert.Equal(t, StatusFailing, resp.Status)
	assert.Len(t, resp.Checks, 2)
	assert.Equal(t, StatusPassing, resp.Checks["db"].Status)
	assert.Equal(t, StatusFailing, resp.Checks["cache"].Status)
	assert.Equal(t, "connection failed", resp.Checks["cache"].Message)
}

func TestRegistry_Run_Timeout(t *testing.T) {
	r := New()

	r.Register("slow", func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, WithTimeout(100*time.Millisecond))

	resp := r.Run(context.Background())
	assert.Equal(t, StatusFailing, resp.Status)
	assert.Equal(t, StatusFailing, resp.Checks["slow"].Status)
	assert.Equal(t, "timeout", resp.Checks["slow"].Message)
}

func TestRegistry_Run_ContextCanceled(t *testing.T) {
	r := New()

	r.Register("check", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	resp := r.Run(ctx)
	assert.Equal(t, StatusFailing, resp.Status)
}

func TestRegistry_Run_DisablePanic(t *testing.T) {
	r := New()

	r.Register("panic", func(ctx context.Context) error {
		panic("test panic")
	}, DisablePanic())

	resp := r.Run(context.Background())
	assert.Equal(t, StatusFailing, resp.Status)
	assert.Equal(t, StatusFailing, resp.Checks["panic"].Status)
	assert.Contains(t, resp.Checks["panic"].Message, "panic")
}

func TestRegistry_Run_WithPanic(t *testing.T) {
	r := New()

	r.Register("panic", func(ctx context.Context) error {
		panic("test panic")
	})

	// Panic in goroutine doesn't propagate - the check will fail instead
	resp := r.Run(context.Background())
	assert.Equal(t, StatusFailing, resp.Status)
	assert.Contains(t, resp.Checks["panic"].Message, "panic")
}

func TestRegistry_Empty(t *testing.T) {
	r := New()

	resp := r.Run(context.Background())
	assert.Equal(t, StatusPassing, resp.Status)
	assert.Empty(t, resp.Checks)
}

func TestRegistry_WithTimeout_Option(t *testing.T) {
	r := New()

	r.Register("fast", func(ctx context.Context) error {
		return nil
	}, WithTimeout(100*time.Millisecond))

	r.Register("slow", func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, WithTimeout(100*time.Millisecond))

	resp := r.Run(context.Background())
	assert.Equal(t, StatusPassing, resp.Checks["fast"].Status)
	assert.Equal(t, StatusFailing, resp.Checks["slow"].Status)
	assert.Equal(t, "timeout", resp.Checks["slow"].Message)
}

func TestCheckResult_Duration(t *testing.T) {
	r := New()

	r.Register("delayed", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	resp := r.Run(context.Background())
	assert.True(t, resp.Checks["delayed"].Duration >= 50*time.Millisecond)
}
