package redisstream

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type Stream struct {
	client *Client
	config StreamConfig
}

func NewStream(client *Client, config StreamConfig) (*Stream, error) {
	config.applyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Stream{client: client, config: config}, nil
}

func (s *Stream) Name() string {
	return s.config.Name
}

func (s *Stream) Send(ctx context.Context, message Message) (string, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(f float64) {
		sendMessageLatency.Observe(f)
	}))
	defer timer.ObserveDuration()

	args, err := s.addArgs(message)
	if err != nil {
		return "", err
	}

	id, err := s.client.rdb.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("redisstream: send: %w", err)
	}
	sendMessageTotal.Inc()

	return id, nil
}

func (s *Stream) SendBatch(ctx context.Context, messages ...Message) ([]string, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(f float64) {
		sendMessageLatency.Observe(f)
	}))
	defer timer.ObserveDuration()

	pipe := s.client.rdb.Pipeline()
	commands := make([]*redis.StringCmd, len(messages))
	for i, message := range messages {
		args, err := s.addArgs(message)
		if err != nil {
			return nil, err
		}
		commands[i] = pipe.XAdd(ctx, args)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("redisstream: send batch: %w", err)
	}

	ids := make([]string, len(commands))
	for i, command := range commands {
		id, err := command.Result()
		if err != nil {
			return ids[:i], fmt.Errorf("redisstream: send batch: %w", err)
		}
		ids[i] = id
	}
	sendMessageTotal.Add(float64(len(ids)))

	return ids, nil
}

func (s *Stream) addArgs(message Message) (*redis.XAddArgs, error) {
	values, err := message.values()
	if err != nil {
		return nil, fmt.Errorf("redisstream: encode message: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: streamKey(s.client.config.Namespace, s.config.Name),
		Values: values,
	}
	if s.config.MaxLen > 0 {
		args.MaxLen = s.config.MaxLen
		args.Approx = true
	}

	return args, nil
}
