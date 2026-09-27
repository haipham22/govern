package graceful

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// Run starts all services and handles graceful shutdown.
//
// A service's shutdown hook is registered only AFTER its Start succeeds, so
// on fail-fast rollback, Shutdown is never called for a service that never
// started. Trade-off: a service whose Start is still in flight when another
// fails will not get its Shutdown hook registered — Start must be
// cancel-safe via ctx for that window.
func Run(
	ctx context.Context,
	log *zap.SugaredLogger,
	shutdownTimeout time.Duration,
	services ...Service,
) error {
	if len(services) == 0 {
		log.Warn("No services provided, nothing to run")
		return nil
	}

	// Create graceful shutdown manager
	m := NewManager(ctx, WithLogger(log))

	// Start all services in goroutines; defer-on-success.
	for _, svc := range services {
		m.Go(func(ctx context.Context) error {
			if err := svc.Start(ctx); err != nil {
				return err
			}
			m.Defer(svc.Shutdown)
			return nil
		})
	}

	log.Info("All services started")

	// Wait for a shutdown signal or the first goroutine error. Either way,
	// Shutdown must run: on the fail-fast path it is the rollback that shuts
	// down every service whose Start already succeeded.
	waitErr := m.Wait()
	if waitErr != nil {
		log.Errorf("Error waiting for shutdown: %v", waitErr)
	}

	log.Info("Shutting down...")
	if err := m.Shutdown(shutdownTimeout); err != nil {
		log.Errorf("Shutdown error: %v", err)
		return errors.Join(waitErr, err)
	}

	if waitErr != nil {
		return waitErr
	}
	log.Info("Graceful shutdown complete")
	return nil
}
