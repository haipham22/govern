//go:build e2e

// End-to-end tests against a real Redis. They are excluded from the normal
// build; run them with a server available:
//
//	docker run --rm -d -p 6379:6379 --name govern-redis redis:7
//	go test ./redisstream/ -tags e2e -v
//	docker rm -f govern-redis
//
// Override the address with REDIS_ADDR.
package redisstream

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func e2eClient(t *testing.T) *Client {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	// A per-test namespace keeps runs isolated without flushing the database.
	client, err := New(context.Background(), Config{
		Addr:      addr,
		Namespace: "e2e-" + uuid.NewString()[:8],
	})
	require.NoError(t, err, "redis must be running at %s", addr)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func fastConsumerConfig(stream, group, id string) ConsumerConfig {
	return ConsumerConfig{
		Stream:            stream,
		Group:             group,
		ConsumerID:        id,
		BatchSize:         10,
		BlockTimeout:      200 * time.Millisecond,
		MaxProcessingTime: 2 * time.Second,
	}
}

// startConsumer runs consumer in the background and stops it when the test ends.
func startConsumer(t *testing.T, consumer *Consumer) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- consumer.Start() }()
	t.Cleanup(func() {
		_ = consumer.Shutdown(context.Background())
		<-done
	})
}

func TestE2E_RoundTrip(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()

	stream, err := NewStream(client, StreamConfig{Name: "orders"})
	require.NoError(t, err)

	consumer, err := NewConsumer(client, fastConsumerConfig("orders", "g1", "c1"))
	require.NoError(t, err)
	startConsumer(t, consumer)

	const total = 50
	for i := 0; i < total; i++ {
		_, err := stream.Send(ctx, Message{
			Key:     fmt.Sprintf("order-%d", i),
			Payload: []byte(fmt.Sprintf("%d", i)),
			Headers: map[string]string{"event.type": "OrderPlaced"},
		})
		require.NoError(t, err)
	}

	received := map[string]bool{}
	deadline := time.After(10 * time.Second)
	for len(received) < total {
		select {
		case message := <-consumer.Messages():
			assert.Equal(t, "OrderPlaced", message.Headers["event.type"])
			assert.Equal(t, "order-"+string(message.Payload), message.Key)
			received[string(message.Payload)] = true
			require.NoError(t, consumer.Ack(ctx, message))
		case <-deadline:
			t.Fatalf("only received %d of %d messages", len(received), total)
		}
	}
	assert.Len(t, received, total)
}

// Redis balances a group's entries across its members: every entry goes to
// exactly one consumer, and a second consumer takes real work off the first.
func TestE2E_TwoConsumersSplitTheLoad(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()

	stream, err := NewStream(client, StreamConfig{Name: "orders"})
	require.NoError(t, err)

	// A batch size of 1 keeps either consumer from swallowing the whole batch
	// in one read before the other has blocked on the group.
	configA := fastConsumerConfig("orders", "g1", "consumer-a")
	configA.BatchSize = 1
	configB := fastConsumerConfig("orders", "g1", "consumer-b")
	configB.BatchSize = 1

	first, err := NewConsumer(client, configA)
	require.NoError(t, err)
	startConsumer(t, first)

	second, err := NewConsumer(client, configB)
	require.NoError(t, err)
	startConsumer(t, second)

	const total = 100
	for i := 0; i < total; i++ {
		_, err := stream.Send(ctx, Message{Payload: []byte(fmt.Sprintf("%d", i))})
		require.NoError(t, err)
	}

	seen := map[string]int{}
	counts := map[string]int{}
	deadline := time.After(15 * time.Second)
	for len(seen) < total {
		select {
		case message := <-first.Messages():
			seen[message.ID]++
			counts["a"]++
			require.NoError(t, first.Ack(ctx, message))
		case message := <-second.Messages():
			seen[message.ID]++
			counts["b"]++
			require.NoError(t, second.Ack(ctx, message))
		case <-deadline:
			t.Fatalf("only received %d of %d messages", len(seen), total)
		}
	}

	assert.Len(t, seen, total)
	for id, times := range seen {
		assert.Equalf(t, 1, times, "entry %s was delivered more than once", id)
	}
	assert.Positive(t, counts["a"], "consumer-a received nothing")
	assert.Positive(t, counts["b"], "consumer-b received nothing")
}

// A consumer that takes delivery and dies leaves its entry pending. The claim
// loop is what brings it back.
func TestE2E_UnackedMessagesAreReclaimed(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()

	stream, err := NewStream(client, StreamConfig{Name: "orders"})
	require.NoError(t, err)

	config := fastConsumerConfig("orders", "g1", "consumer-a")
	config.MaxProcessingTime = time.Second
	consumer, err := NewConsumer(client, config)
	require.NoError(t, err)
	startConsumer(t, consumer)

	_, err = stream.Send(ctx, Message{Payload: []byte("payload")})
	require.NoError(t, err)

	// Take delivery but never ack: the entry stays pending and must come back.
	var first Message
	select {
	case first = <-consumer.Messages():
		assert.Equal(t, []byte("payload"), first.Payload)
	case <-time.After(10 * time.Second):
		t.Fatal("message was never delivered")
	}

	select {
	case second := <-consumer.Messages():
		assert.Equal(t, first.ID, second.ID, "expected the same entry to be reclaimed")
		assert.NoError(t, consumer.Ack(ctx, second))
	case <-time.After(10 * time.Second):
		t.Fatal("unacked message was never reclaimed")
	}
}

func TestE2E_ShutdownClosesMessages(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()

	consumer, err := NewConsumer(client, fastConsumerConfig("orders", "g1", "c1"))
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- consumer.Start() }()

	assert.NoError(t, consumer.Shutdown(ctx))
	assert.NoError(t, <-done)

	_, open := <-consumer.Messages()
	assert.False(t, open, "Messages() must be closed once Start returns")
}
