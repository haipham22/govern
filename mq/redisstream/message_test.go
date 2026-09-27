package redisstream

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestMessageValues_OmitsHeadersWhenEmpty(t *testing.T) {
	values, err := Message{Payload: []byte("hi")}.values()

	assert.NoError(t, err)
	assert.Equal(t, []byte("hi"), values[fieldPayload])
	assert.NotContains(t, values, fieldKey)
	assert.NotContains(t, values, fieldHeaders)
}

func TestMessageFromEntry_RoundTripsThroughRedisFields(t *testing.T) {
	original := Message{
		Key:     "order-1",
		Payload: []byte(`{"total":10}`),
		Headers: map[string]string{"event.type": "OrderPlaced"},
	}

	values, err := original.values()
	assert.NoError(t, err)

	// Redis hands every field back as a string.
	entry := redis.XMessage{ID: "1-0", Values: map[string]any{}}
	for key, value := range values {
		switch typed := value.(type) {
		case []byte:
			entry.Values[key] = string(typed)
		default:
			entry.Values[key] = typed
		}
	}

	decoded, err := messageFromEntry(entry)

	assert.NoError(t, err)
	assert.Equal(t, "1-0", decoded.ID)
	assert.Equal(t, original.Key, decoded.Key)
	assert.Equal(t, original.Payload, decoded.Payload)
	assert.Equal(t, original.Headers, decoded.Headers)
}

func TestMessageFromEntry_ReportsMalformedHeaders(t *testing.T) {
	entry := redis.XMessage{ID: "1-0", Values: map[string]any{
		fieldPayload: "d",
		fieldHeaders: "{not json",
	}}

	_, err := messageFromEntry(entry)

	assert.Error(t, err)
}

func TestNewConsumerID_IsUniquePerCall(t *testing.T) {
	assert.NotEqual(t, newConsumerID(), newConsumerID())
}
