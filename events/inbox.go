// Package events implements the inbox pattern — the consumer-side mirror of
// the outbox pattern in the outbox package.
//
// A Consumer reads messages from Kafka and inserts them into an `inbox` table
// with an idempotent Insert (ON CONFLICT DO NOTHING on the event ID), then
// commits the Kafka offset. A Worker polls the inbox with SELECT ... FOR
// UPDATE SKIP LOCKED and dispatches each record to a Handler.
//
// Two immediate benefits over an inline Kafka-consumer:
//
//   - Decoupling: Kafka offset commit does not wait for slow business logic.
//     The Worker paces the side effects; Kafka just fills the inbox.
//   - Exactly-once effect (opt-in): Handler can call
//     InboxStorage.MarkProcessed inside its own business transaction so the
//     mark and the side effect commit atomically together.
//
// Delivery is at-least-once end-to-end, but the effect on the database can
// be made exactly-once via the transactional MarkProcessed call.
package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Record is one inbox row — the persisted form of a received event.
//
// ID equals the producer's event UUID (Kafka header `event.id`) so dedup
// works via the primary key: a duplicate Kafka delivery attempts to INSERT
// the same ID and is a no-op under ON CONFLICT DO NOTHING.
type Record struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   string
	AggregateType string
	Payload       []byte
	Headers       map[string]string
	ReceivedAt    time.Time
	ProcessedAt   *time.Time
	Attempts      int
}

// Handler processes one Record from the inbox. Returning nil marks the row
// processed (via Worker.tick); returning an error leaves it pending —
// Worker records the last error and the row is re-claimed on a later tick.
//
// For exactly-once effect on the database, the Handler MAY mark the row
// processed inside its own business transaction. That requires a
// concrete-impl method (e.g. `MarkProcessedTx(ctx, tx, ids...)`) — the
// govern InboxStorage interface only guarantees the post-return
// MarkProcessed variant, which the Worker calls after Handler returns.
type Handler func(ctx context.Context, r Record) error
