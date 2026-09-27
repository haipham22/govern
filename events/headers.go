package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// RecordFromMessage builds a Record from a transport message's raw parts.
// key is the message key (matches AggregateID from the producer's
// outbox.MessageKey); headers is a string-string map of the transport
// message's headers as populated by outbox.MessageHeaders.
//
// Returns an error if event.id is missing or unparseable — that indicates a
// producer bug (the outbox helper always sets a valid UUID).
//
// Use this in your Consumer implementation so the inbox row stays
// wire-compatible with what outbox producers wrote, regardless of which
// Kafka client library you use.
func RecordFromMessage(payload []byte, key []byte, headers map[string]string) (Record, error) {
	idStr, ok := headers["event.id"]
	if !ok {
		return Record{}, errors.New("events: missing event.id header")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return Record{}, fmt.Errorf("events: event.id is not a UUID (%q): %w", idStr, err)
	}
	return Record{
		ID:            id,
		EventType:     headers["event.type"],
		AggregateID:   string(key),
		AggregateType: headers["aggregate.type"],
		Payload:       payload,
		Headers:       headers,
	}, nil
}

// ContextFromHeaders returns a context that continues the upstream trace
// stored in the headers map (via `traceparent` / `baggage`). Use this in
// the Consumer to reconstruct the trace before Insert-ing to the inbox,
// and again in the Worker before calling Handler.
func ContextFromHeaders(ctx context.Context, headers map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, mapCarrier(headers))
}

// mapCarrier adapts a header map to propagation.TextMapCarrier.
type mapCarrier map[string]string

var _ propagation.TextMapCarrier = mapCarrier(nil)

func (m mapCarrier) Get(key string) string { return m[key] }
func (m mapCarrier) Set(key, val string)   { m[key] = val }
func (m mapCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
