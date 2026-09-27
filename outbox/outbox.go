// Package outbox specifies the transactional outbox pattern.
//
// It defines:
//   - Record: the persisted event shape (id, aggregate, event type, payload,
//     headers).
//   - Storage: interface consumed by Relay (Claim / MarkSent / FailRetry).
//     Producer-side Insert is a concrete-impl concern — the impl exposes a
//     tx-aware method matching its DB driver so the outbox row commits
//     atomically with the aggregate change.
//   - Publisher: transport-agnostic interface for shipping records
//     (Kafka, NATS, RabbitMQ, in-process bus).
//   - Relay: polling worker that reads pending records and hands them to
//     the Publisher. Start blocks on a caller-supplied context.
//   - Header/key helpers (MessageHeaders, MessageKey): standardize the wire
//     conventions producers must emit — event.id / event.type /
//     aggregate.type headers, W3C TraceContext injection, AggregateID as
//     message key — regardless of which client library the concrete
//     Publisher uses.
//
// The Storage implementation is agent-generated per project's stack; see
// the README's Schema section for the reference spec.
package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Record is a single event pending publication.
type Record struct {
	ID            uuid.UUID
	AggregateID   string
	AggregateType string
	EventType     string
	Payload       []byte
	Headers       map[string]string
	CreatedAt     time.Time
	Attempts      int
}

// Publisher sends a Record to its downstream destination (Kafka, NATS, ...).
// Publish must be safe for concurrent use.
type Publisher interface {
	Publish(ctx context.Context, r Record) error
}
