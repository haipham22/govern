package outbox

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// MessageKey returns the message key a producer should set on its
// transport message (Kafka message key, NATS subject, ...).
//
// Using AggregateID here ensures records for the same aggregate land on the
// same partition and preserve delivery order.
func MessageKey(r Record) []byte {
	return []byte(r.AggregateID)
}

// MessageHeaders returns the standard event.* headers for r plus any
// user-supplied Headers, and injects the current context's OTel
// TraceContext + Baggage under the standard header names (`traceparent`,
// `baggage`).
//
// Use this from your Publisher implementation so header conventions stay
// consistent across producers regardless of which Kafka client library
// you use downstream.
func MessageHeaders(ctx context.Context, r Record) map[string]string {
	h := make(map[string]string, len(r.Headers)+3)
	for k, v := range r.Headers {
		h[k] = v
	}
	h["event.id"] = r.ID.String()
	h["event.type"] = r.EventType
	h["aggregate.type"] = r.AggregateType
	otel.GetTextMapPropagator().Inject(ctx, mapCarrier(h))
	return h
}

// mapCarrier adapts a header map to propagation.TextMapCarrier for OTel
// Inject/Extract.
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
