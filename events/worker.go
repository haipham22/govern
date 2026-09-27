package events

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// WorkerConfig tunes the inbox-dispatching Worker.
type WorkerConfig struct {
	// PollInterval between claim attempts when no records are found. When
	// Claim returns records, the next tick runs immediately (drains a
	// backlog without idle waiting). Default 500ms.
	PollInterval time.Duration

	// BatchSize per claim. Default 100.
	BatchSize int

	// Logger for lifecycle + error events. Default: slog.Default().
	Logger *slog.Logger
}

func (c *WorkerConfig) applyDefaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = 500 * time.Millisecond
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

// Worker polls the inbox and dispatches each Record to Handler.
// It is caller-driven: Start blocks on the context you pass it.
//
// On Handler success, Worker calls Storage.MarkProcessed. That call MUST be
// idempotent per the InboxStorage contract — so if Handler already marked
// the row inside its own business transaction (using a concrete-impl method
// like MarkProcessedTx) for exactly-once effect, the Worker's post-return
// call is a harmless no-op.
type Worker struct {
	Storage InboxStorage
	Handler Handler

	cfg WorkerConfig
}

// NewWorker wires a Worker. Storage and Handler are required.
func NewWorker(storage InboxStorage, handler Handler, cfg WorkerConfig) *Worker {
	cfg.applyDefaults()
	return &Worker{Storage: storage, Handler: handler, cfg: cfg}
}

// Start blocks until ctx is canceled.
func (w *Worker) Start(ctx context.Context) error {
	if w.Storage == nil || w.Handler == nil {
		return errors.New("events: Worker requires Storage and Handler")
	}
	w.cfg.Logger.Info("events worker starting",
		"poll", w.cfg.PollInterval, "batch", w.cfg.BatchSize)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		drained := w.Tick(ctx)
		if drained {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.cfg.PollInterval):
		}
	}
}

// Shutdown is a no-op — the polling loop exits when Start's ctx is canceled.
// It exists so Worker pairs with a Start/Shutdown wiring.
func (w *Worker) Shutdown(_ context.Context) error {
	return nil
}

// Tick claims a batch and dispatches each record. Returns true if one or more
// records were claimed (so the caller drains a backlog without sleeping).
func (w *Worker) Tick(ctx context.Context) bool {
	records, err := w.Storage.Claim(ctx, w.cfg.BatchSize)
	if err != nil {
		w.cfg.Logger.ErrorContext(ctx, "events claim failed", "error", err)
		return false
	}
	if len(records) == 0 {
		return false
	}

	for _, rec := range records {
		recCtx := ContextFromHeaders(ctx, rec.Headers)

		if err := w.Handler(recCtx, rec); err != nil {
			w.cfg.Logger.ErrorContext(recCtx, "events handler failed",
				"event_id", rec.ID, "event_type", rec.EventType,
				"attempts", rec.Attempts, "error", err)
			if ferr := w.Storage.FailRetry(ctx, rec.ID, err); ferr != nil {
				w.cfg.Logger.ErrorContext(recCtx, "events fail_retry failed",
					"event_id", rec.ID, "error", ferr)
			}
			continue
		}

		// Idempotent: no-op if Handler already marked the row in its own tx
		// (via a concrete-impl method the Handler holds a reference to).
		if err := w.Storage.MarkProcessed(ctx, rec.ID); err != nil {
			w.cfg.Logger.ErrorContext(recCtx, "events mark_processed failed",
				"event_id", rec.ID, "error", err)
		}
	}
	return true
}
