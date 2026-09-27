package events

import (
	"context"

	"github.com/google/uuid"
)

// InboxStorage is the interface Consumer and Worker consume from an inbox
// implementation. Govern defines the shape; each project provides the
// concrete impl (Postgres, MySQL, ...). See the events README for the
// reference schema + rationale so an impl stays wire-compatible with
// Consumer + Worker.
//
// Note: exactly-once effect on the database requires the Handler to include
// the MarkProcessed call inside its own business transaction. That is a
// concrete-impl concern (e.g. `MarkProcessedTx(ctx, tx pgx.Tx, ids...)`);
// the Worker's post-return MarkProcessed on this interface is idempotent so
// the two do not conflict.
type InboxStorage interface {
	// Insert idempotently stores r. Duplicate ID (same event.id) MUST be a
	// no-op (Postgres: ON CONFLICT (id) DO NOTHING). Called by the Consumer
	// before it commits the Kafka offset.
	Insert(ctx context.Context, r Record) error

	// Claim atomically returns up to `limit` pending records
	// (processed_at IS NULL), ordered by received_at, using an at-most-once
	// selection primitive (Postgres: SELECT ... FOR UPDATE SKIP LOCKED).
	// Attempts on each returned record MUST be incremented.
	Claim(ctx context.Context, limit int) ([]Record, error)

	// MarkProcessed stamps processed_at on the given ids. Must be idempotent
	// (Postgres: `WHERE processed_at IS NULL`) so a Handler that already
	// marked the row in its own transaction does not conflict with the
	// Worker's post-return call.
	MarkProcessed(ctx context.Context, ids ...uuid.UUID) error

	// FailRetry records the last Handler error. The row stays pending —
	// Claim will return it again on a later tick (at-least-once retry).
	FailRetry(ctx context.Context, id uuid.UUID, cause error) error
}
