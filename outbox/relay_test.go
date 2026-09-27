package outbox_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haipham22/govern/outbox"
)

type fakeStorage struct {
	pending []outbox.Record
	marked  []uuid.UUID
	failed  map[uuid.UUID]error
}

func (s *fakeStorage) Claim(_ context.Context, limit int) ([]outbox.Record, error) {
	if len(s.pending) > limit {
		return s.pending[:limit], nil
	}
	return s.pending, nil
}

func (s *fakeStorage) MarkSent(_ context.Context, ids ...uuid.UUID) error {
	s.marked = append(s.marked, ids...)
	return nil
}

func (s *fakeStorage) FailRetry(_ context.Context, id uuid.UUID, cause error) error {
	if s.failed == nil {
		s.failed = map[uuid.UUID]error{}
	}
	s.failed[id] = cause
	return nil
}

type fakePublisher struct {
	err       error
	published []outbox.Record
}

func (p *fakePublisher) Publish(_ context.Context, r outbox.Record) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, r)
	return nil
}

func TestRelayTickPublishesAndMarksSent(t *testing.T) {
	store := &fakeStorage{pending: []outbox.Record{{ID: uuid.New(), EventType: "order.created"}}}
	pub := &fakePublisher{}
	relay := outbox.NewRelay(store, pub, outbox.RelayConfig{})

	drained := relay.Tick(context.Background())

	assert.True(t, drained)
	assert.Equal(t, store.pending, pub.published)
	assert.Equal(t, []uuid.UUID{store.pending[0].ID}, store.marked)
}

func TestRelayTickRecordsFailureAndKeepsPending(t *testing.T) {
	boom := assert.AnError
	rec := outbox.Record{ID: uuid.New(), EventType: "order.created"}
	store := &fakeStorage{pending: []outbox.Record{rec}}
	pub := &fakePublisher{err: boom}
	relay := outbox.NewRelay(store, pub, outbox.RelayConfig{})

	drained := relay.Tick(context.Background())

	assert.True(t, drained)
	assert.Empty(t, pub.published)
	assert.Empty(t, store.marked)
	assert.Equal(t, boom, store.failed[rec.ID], "failure must be recorded, row stays pending")
}

func TestRelayTickEmptyQueue(t *testing.T) {
	relay := outbox.NewRelay(&fakeStorage{}, &fakePublisher{}, outbox.RelayConfig{})

	assert.False(t, relay.Tick(context.Background()))
}

func TestRelayStartReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	relay := outbox.NewRelay(&fakeStorage{}, &fakePublisher{}, outbox.RelayConfig{})

	require.NoError(t, relay.Start(ctx))
}
