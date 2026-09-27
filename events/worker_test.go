package events_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/haipham22/govern/events"
)

type fakeInbox struct {
	pending   []events.Record
	marked    []uuid.UUID
	failed    map[uuid.UUID]error
	processed []uuid.UUID
}

func (s *fakeInbox) Insert(_ context.Context, _ events.Record) error { return nil }

func (s *fakeInbox) Claim(_ context.Context, limit int) ([]events.Record, error) {
	if len(s.pending) > limit {
		return s.pending[:limit], nil
	}
	return s.pending, nil
}

func (s *fakeInbox) MarkProcessed(_ context.Context, ids ...uuid.UUID) error {
	s.processed = append(s.processed, ids...)
	return nil
}

func (s *fakeInbox) FailRetry(_ context.Context, id uuid.UUID, cause error) error {
	if s.failed == nil {
		s.failed = map[uuid.UUID]error{}
	}
	s.failed[id] = cause
	return nil
}

func TestWorkerTickDispatchesAndMarksProcessed(t *testing.T) {
	rec := events.Record{ID: uuid.New(), EventType: "order.created"}
	store := &fakeInbox{pending: []events.Record{rec}}
	handler := func(_ context.Context, _ events.Record) error { return nil }
	w := events.NewWorker(store, handler, events.WorkerConfig{})

	assert.True(t, w.Tick(context.Background()))
	assert.Equal(t, []uuid.UUID{rec.ID}, store.processed)
}

func TestWorkerTickHandlerFailureKeepsPending(t *testing.T) {
	rec := events.Record{ID: uuid.New(), EventType: "order.created"}
	store := &fakeInbox{pending: []events.Record{rec}}
	handler := func(_ context.Context, _ events.Record) error { return assert.AnError }
	w := events.NewWorker(store, handler, events.WorkerConfig{})

	assert.True(t, w.Tick(context.Background()))
	assert.Empty(t, store.processed)
	assert.Equal(t, assert.AnError, store.failed[rec.ID], "failure recorded, row stays pending")
}
