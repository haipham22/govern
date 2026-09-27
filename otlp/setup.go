// Package otlp wires OpenTelemetry TracerProvider, MeterProvider, and
// LoggerProvider with OTLP exporters (gRPC or HTTP) from a single Config,
// and returns a Shutdown func that flushes and closes them cleanly.
//
// The returned Shutdown is a drop-in deferred cleanup, so telemetry
// gets a chance to flush during shutdown.
package otlp

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
)

// Shutdown flushes and closes the providers registered by Setup.
// It is safe to call from a deferred cleanup hook.
type Shutdown func(context.Context) error

// Setup initializes OTel TracerProvider / MeterProvider / LoggerProvider
// per cfg and installs them as globals. It also installs a composite
// TextMapPropagator (TraceContext + Baggage).
//
// When cfg.Enabled is false, Setup is a no-op and returns a nil-safe shutdown.
//
// If any signal fails to initialize, providers created earlier in the call
// are shut down before returning the error.
func Setup(ctx context.Context, cfg Config) (Shutdown, error) {
	noop := func(context.Context) error { return nil }

	if !cfg.Enabled {
		return noop, nil
	}

	cfg.applyDefaults()

	if cfg.ServiceName == "" {
		return nil, errors.New("otlp: ServiceName is required")
	}
	if cfg.Endpoint == "" {
		return nil, errors.New("otlp: Endpoint is required")
	}
	if !cfg.Traces && !cfg.Metrics && !cfg.Logs {
		return noop, nil
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("otlp: build resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	var shutdowns []Shutdown
	rollback := func(cause error) (Shutdown, error) {
		_ = runShutdowns(context.Background(), shutdowns)
		return nil, cause
	}

	if cfg.Traces {
		tp, err := newTracerProvider(ctx, cfg, res)
		if err != nil {
			return rollback(fmt.Errorf("otlp: tracer provider: %w", err))
		}
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	if cfg.Metrics {
		mp, err := newMeterProvider(ctx, cfg, res)
		if err != nil {
			return rollback(fmt.Errorf("otlp: meter provider: %w", err))
		}
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	if cfg.Logs {
		lp, err := newLoggerProvider(ctx, cfg, res)
		if err != nil {
			return rollback(fmt.Errorf("otlp: logger provider: %w", err))
		}
		global.SetLoggerProvider(lp)
		shutdowns = append(shutdowns, lp.Shutdown)
	}

	return func(ctx context.Context) error {
		return runShutdowns(ctx, shutdowns)
	}, nil
}

// runShutdowns invokes each shutdown in LIFO order and joins any errors.
func runShutdowns(ctx context.Context, fns []Shutdown) error {
	var errs []error
	for i := len(fns) - 1; i >= 0; i-- {
		if err := fns[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
