package otlp

import (
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// NewSlogLogger returns an *slog.Logger whose records flow through the
// globally-configured OTel LoggerProvider (set by Setup with cfg.Logs = true).
//
// name identifies the emitting scope (typically the service or module name)
// and appears as the `otel.scope.name` attribute on each log record.
//
// When Setup has not installed a LoggerProvider, records fall through to a
// no-op logger and are silently dropped — safe to call unconditionally.
func NewSlogLogger(name string) *slog.Logger {
	return otelslog.NewLogger(name)
}
