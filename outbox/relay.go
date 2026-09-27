package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// RelayConfig tunes the polling loop.
type RelayConfig struct {
	// PollInterval between claim attempts when no records are found.
	// When Claim returns records, the next iteration runs immediately (no
	// sleep) so a backlog drains without idle waiting. Default 500ms.
	PollInterval time.Duration

	// BatchSize per claim. Default 100.
	BatchSize int

	// Logger used for lifecycle + error events. Default slog.Default().
	Logger *slog.Logger
}

func (c *RelayConfig) applyDefaults() {
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

// Relay is the background worker that reads pending records from Storage and
// hands each to Publisher. It is caller-driven: Start blocks on the context
// you pass it, so cancel that context to stop the relay.
//
// Semantics:
//   - At-least-once: publish success + MarkSent failure -> republish next tick.
//     Consumers must be idempotent.
//   - Ordering: per-aggregate order is preserved by (a) ORDER BY created_at in
//     Claim, (b) single-worker publishing per tick, and (c) Kafka partitioning
//     by AggregateID (set as message key by KafkaPublisher).
type Relay struct {
	Storage   Storage
	Publisher Publisher

	cfg RelayConfig
}

// NewRelay wires a Relay. Storage and Publisher are required.
func NewRelay(storage Storage, publisher Publisher, cfg RelayConfig) *Relay {
	cfg.applyDefaults()
	return &Relay{Storage: storage, Publisher: publisher, cfg: cfg}
}

// Start blocks until ctx is canceled.
func (r *Relay) Start(ctx context.Context) error {
	if r.Storage == nil || r.Publisher == nil {
		return errors.New("outbox: Relay requires Storage and Publisher")
	}
	r.cfg.Logger.Info("outbox relay starting",
		"poll", r.cfg.PollInterval, "batch", r.cfg.BatchSize)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		drained := r.Tick(ctx)

		if drained {
			// Backlog: loop immediately without sleeping.
			continue
		}
		// Idle: wait for the next tick or shutdown.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(r.cfg.PollInterval):
		}
	}
}

// Shutdown is a no-op: the polling loop exits when Start's ctx is canceled.
// It exists so Relay pairs with a Start/Shutdown wiring.
func (r *Relay) Shutdown(_ context.Context) error {
	return nil
}

// Tick claims a batch and publishes each record. Returns true if any records
// were processed (so the caller can drain a backlog without sleeping).
func (r *Relay) Tick(ctx context.Context) bool {
	records, err := r.Storage.Claim(ctx, r.cfg.BatchSize)
	if err != nil {
		r.cfg.Logger.ErrorContext(ctx, "outbox claim failed", "error", err)
		return false
	}
	if len(records) == 0 {
		return false
	}

	for _, rec := range records {
		if err := r.Publisher.Publish(ctx, rec); err != nil {
			r.cfg.Logger.ErrorContext(ctx, "outbox publish failed",
				"id", rec.ID, "event_type", rec.EventType,
				"attempts", rec.Attempts, "error", err)
			// Record the error; row stays pending for the next claim.
			if ferr := r.Storage.FailRetry(ctx, rec.ID, err); ferr != nil {
				r.cfg.Logger.ErrorContext(ctx, "outbox fail_retry failed",
					"id", rec.ID, "error", ferr)
			}
			continue
		}
		if err := r.Storage.MarkSent(ctx, rec.ID); err != nil {
			// Published but not marked sent — will republish. At-least-once.
			r.cfg.Logger.ErrorContext(ctx, "outbox mark_sent failed",
				"id", rec.ID, "error", err)
		}
	}
	return true
}
