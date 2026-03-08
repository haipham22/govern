package asynq

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid args", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		mux.HandleFunc("test", func(ctx context.Context, task *asynq.Task) error {
			return nil
		})

		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		require.NotNil(t, server)
		require.NotNil(t, cleanup)
		defer cleanup()

		assert.NotNil(t, server.server)
		assert.NotNil(t, server.mux)
		assert.NotNil(t, server.config)
	})

	t.Run("panics on nil mux", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		_, _, err := NewServer(redisClient, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mux cannot be nil")
	})

	t.Run("applies options", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()

		server, cleanup, err := NewServer(redisClient, mux,
			WithConcurrency(20),
			WithQueues(map[string]int{"critical": 5}),
		)
		require.NoError(t, err)
		defer cleanup()

		assert.Equal(t, 20, server.config.Concurrency)
		assert.Equal(t, 5, server.config.Queues["critical"])
	})
}

func TestServer_Start(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	t.Run("starts server successfully", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		mux.HandleFunc("test", func(ctx context.Context, task *asynq.Task) error {
			return nil
		})

		server, cleanup, err := NewServer(redisClient, mux,
			WithConcurrency(1),
		)
		require.NoError(t, err)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())

		// Start server in background
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		assert.True(t, server.IsStarted())

		// Shutdown server
		cancel()
		err = server.Shutdown(context.Background())
		assert.NoError(t, err)

		select {
		case err := <-errChan:
			// Server should exit after context cancellation
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("server did not shutdown in time")
		}
	})

	t.Run("cannot start twice", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// First start
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		time.Sleep(50 * time.Millisecond)

		// Second start should fail
		err = server.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		// Cleanup
		cancel()
		<-errChan
	})
}

func TestServer_Shutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	t.Run("shuts down server gracefully", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		mux.HandleFunc("test", func(ctx context.Context, task *asynq.Task) error {
			return nil
		})

		server, cleanup, err := NewServer(redisClient, mux,
			WithConcurrency(1),
			WithShutdownTimeout(1*time.Second),
		)
		require.NoError(t, err)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server
		go func() {
			_ = server.Start(ctx)
		}()

		time.Sleep(50 * time.Millisecond)
		assert.True(t, server.IsStarted())

		// Shutdown
		err = server.Shutdown(context.Background())
		assert.NoError(t, err)
		assert.True(t, server.IsShutdown())
	})

	t.Run("cannot shutdown twice", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		// First shutdown
		err = server.Shutdown(context.Background())
		assert.NoError(t, err)

		// Second shutdown should fail
		err = server.Shutdown(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already shutdown")
	})
}

func TestServer_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	t.Run("close is alias for shutdown", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		// Close should work without starting
		err = server.Close()
		assert.NoError(t, err)
		assert.True(t, server.IsShutdown())
	})
}

func TestServer_SetLogger(t *testing.T) {
	t.Run("sets logger", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		logger := zap.NewNop().Sugar()
		server.SetLogger(logger)

		// Logger should be set (no direct way to verify, but no panic is success)
	})
}

func TestServer_IsStarted(t *testing.T) {
	t.Run("returns false before start", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		assert.False(t, server.IsStarted())
	})
}

func TestServer_IsShutdown(t *testing.T) {
	t.Run("returns false before shutdown", func(t *testing.T) {
		redisClient := mockRedisClient(t)
		if redisClient == nil {
			return
		}
		defer redisClient.Close()

		mux := NewTaskMux()
		server, cleanup, err := NewServer(redisClient, mux)
		require.NoError(t, err)
		defer cleanup()

		assert.False(t, server.IsShutdown())
	})
}
