package redisstream

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	fieldKey     = "k"
	fieldPayload = "d"
	fieldHeaders = "h"
)

type Message struct {
	ID string

	// Key is free-form metadata — typically the aggregate ID, so a consumer
	// can hand it to events.RecordFromMessage. Nothing routes on it.
	Key string

	Payload []byte
	Headers map[string]string
}

func (m Message) values() (map[string]any, error) {
	values := map[string]any{
		fieldPayload: m.Payload,
	}
	if m.Key != "" {
		values[fieldKey] = m.Key
	}
	if len(m.Headers) > 0 {
		encoded, err := json.Marshal(m.Headers)
		if err != nil {
			return nil, err
		}
		values[fieldHeaders] = encoded
	}
	return values, nil
}

func messageFromEntry(entry redis.XMessage) (Message, error) {
	message := Message{ID: entry.ID}

	if raw, ok := entry.Values[fieldKey].(string); ok {
		message.Key = raw
	}
	if raw, ok := entry.Values[fieldPayload].(string); ok {
		message.Payload = []byte(raw)
	}
	if raw, ok := entry.Values[fieldHeaders].(string); ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &message.Headers); err != nil {
			return Message{}, err
		}
	}

	return message, nil
}

func newConsumerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	return hostname + "-" + uuid.NewString()
}
