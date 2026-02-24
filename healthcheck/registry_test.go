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

func TestRegistry_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Registry)
		wantStatus Status
		validate func(*testing.T, *Response)
	}{
		{
			name: "all checks passing",
			setup: func(r *Registry) {
				r.Register("db", func(ctx context.Context) error {
					return nil
				})
				r.Register("cache", func(ctx context.Context) error {
					return nil
				})
			},
			wantStatus: StatusPassing,
			validate: func(t *testing.T, resp *Response) {
				assert.Len(t, resp.Checks, 2)
				assert.Equal(t, StatusPassing, resp.Checks["db"].Status)
				assert.Equal(t, StatusPassing, resp.Checks["cache"].Status)
			},
		},
		{
			name: "one check failing",
			setup: func(r *Registry) {
				r.Register("db", func(ctx context.Context) error {
					return nil
				})
				r.Register("cache", func(ctx context.Context) error {
					return errors.New("connection failed")
				})
			},
			wantStatus: StatusFailing,
			validate: func(t *testing.T, resp *Response) {
				assert.Len(t, resp.Checks, 2)
				assert.Equal(t, StatusPassing, resp.Checks["db"].Status)
				assert.Equal(t, StatusFailing, resp.Checks["cache"].Status)
				assert.Equal(t, "connection failed", resp.Checks["cache"].Message)
			},
		},
		{
			name: "check timeout",
			setup: func(r *Registry) {
				r.Register("slow", func(ctx context.Context) error {
					select {
					case <-time.After(10 * time.Second):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}, WithTimeout(100*time.Millisecond))
			},
			wantStatus: StatusFailing,
			validate: func(t *testing.T, resp *Response) {
				assert.Equal(t, StatusFailing, resp.Checks["slow"].Status)
				assert.Equal(t, "timeout", resp.Checks["slow"].Message)
			},
		},
		{
			name: "context canceled during run",
			setup: func(r *Registry) {
				r.Register("check", func(ctx context.Context) error {
					select {
					case <-time.After(5 * time.Second):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})
			},
			wantStatus: StatusFailing,
			validate: func(t *testing.T, resp *Response) {
				// Check fails when context is canceled
				assert.Equal(t, StatusFailing, resp.Status)
			},
		},
		{
			name: "panic with DisablePanic option",
			setup: func(r *Registry) {
				r.Register("panic", func(ctx context.Context) error {
					panic("test panic")
				}, DisablePanic())
			},
			wantStatus: StatusFailing,
			validate: func(t *testing.T, resp *Response) {
				assert.Equal(t, StatusFailing, resp.Checks["panic"].Status)
				assert.Contains(t, resp.Checks["panic"].Message, "panic")
			},
		},
		{
			name: "panic without DisablePanic option",
			setup: func(r *Registry) {
				r.Register("panic", func(ctx context.Context) error {
					panic("test panic")
				})
			},
			wantStatus: StatusFailing,
			validate: func(t *testing.T, resp *Response) {
				// Panic in goroutine doesn't propagate - the check will fail instead
				assert.Contains(t, resp.Checks["panic"].Message, "panic")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			tt.setup(r)

			var resp *Response
			if tt.name == "context canceled during run" {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}()
				resp = r.Run(ctx)
			} else {
				resp = r.Run(context.Background())
			}

			assert.Equal(t, tt.wantStatus, resp.Status)
			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestRegistry_Run_TimeoutOption(t *testing.T) {
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

func TestRegistry_Run_Empty(t *testing.T) {
	r := New()

	resp := r.Run(context.Background())
	assert.Equal(t, StatusPassing, resp.Status)
	assert.Empty(t, resp.Checks)
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
