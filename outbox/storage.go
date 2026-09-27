package outbox

import (
	"context"

	"github.com/google/uuid"
)

// Storage is the subset the Relay needs from an outbox implementation:
// pull pending records, mark them sent, and record retry errors.
//
// Insert (the producer-side write) is intentionally NOT on this interface —
// it is the concrete implementation's responsibility to expose a method that
// participates in the caller's business transaction (e.g. a Postgres impl
// might offer `InsertTx(ctx, tx pgx.Tx, records ...Record)`). Producers work
// against the concrete impl; govern only defines what the Relay consumes.
//
// See the outbox README for the reference schema + rationale so a per-project
// implementation stays wire-compatible with KafkaPublisher and the Relay.
type Storage interface {
	// Claim atomically returns up to `limit` pending records (sent_at IS
	// NULL), ordered by created_at, using an at-most-once selection primitive
	// (Postgres: SELECT ... FOR UPDATE SKIP LOCKED). Attempts on each
	// returned record MUST be incremented so failures are visible.
	Claim(ctx context.Context, limit int) ([]Record, error)

	// MarkSent stamps sent_at on the given ids so they never re-appear from
	// Claim. Must be idempotent for redelivery safety.
	MarkSent(ctx context.Context, ids ...uuid.UUID) error

	// FailRetry records the last publish error for a failed record. The
	// record stays pending — at-least-once retry on the next Claim.
	FailRetry(ctx context.Context, id uuid.UUID, cause error) error
}
