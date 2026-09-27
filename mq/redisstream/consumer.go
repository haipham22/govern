package redisstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/haipham22/govern/log"
)

const readErrorBackoff = time.Second

type Consumer struct {
	client *Client
	config ConsumerConfig
	key    string

	messages chan Message

	// cancel stops the consume loops; done closes when run has returned, so
	// Shutdown waits for a real quiesce instead of returning optimistically.
	cancel   context.CancelFunc
	stopOnce sync.Once
	done     chan struct{}
	doneMu   sync.Mutex
}

func NewConsumer(client *Client, config ConsumerConfig) (*Consumer, error) {
	config.applyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client.config.ReadTimeout > 0 && config.BlockTimeout >= client.config.ReadTimeout {
		return nil, fmt.Errorf("redisstream: %w: block_timeout %s, read_timeout %s",
			ErrBlockTimeoutTooLong, config.BlockTimeout, client.config.ReadTimeout)
	}

	return &Consumer{
		client:   client,
		config:   config,
		key:      streamKey(client.config.Namespace, config.Stream),
		messages: make(chan Message, config.BufferSize),
	}, nil
}

func (c *Consumer) ID() string {
	return c.config.ConsumerID
}

func (c *Consumer) Messages() <-chan Message {
	return c.messages
}

// Start runs the consumer until ctx is canceled. It fits the graceful.Service
// shape: run it in its own goroutine (e.g. via graceful.Manager.Go) and pair
// it with Shutdown.
func (c *Consumer) Start(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel

	c.doneMu.Lock()
	c.done = make(chan struct{})
	c.doneMu.Unlock()

	defer func() {
		cancel()
		close(c.done)
	}()

	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	return c.run(ctx)
}

// Shutdown stops the consumer and waits for its loops to exit, bounded by ctx
// — or by the configured shutdown timeout when ctx carries no deadline. It is
// safe to call more than once, and returns nil when Start never ran.
func (c *Consumer) Shutdown(ctx context.Context) error {
	c.doneMu.Lock()
	done := c.done
	c.doneMu.Unlock()

	if c.cancel != nil {
		c.stopOnce.Do(func() { c.cancel() })
	}
	if done == nil {
		return nil
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.ShutdownTimeout)
		defer cancel()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) run(ctx context.Context) error {
	log.Default().Infow("redisstream consumer starting",
		"stream", c.config.Stream, "group", c.config.Group, "consumer", c.config.ConsumerID)

	var loops sync.WaitGroup
	for _, loop := range []func(context.Context){c.readLoop, c.claimLoop} {
		loops.Add(1)
		go func(fn func(context.Context)) {
			defer loops.Done()
			fn(ctx)
		}(loop)
	}

	<-ctx.Done()
	loops.Wait()
	close(c.messages)

	log.Default().Infow("redisstream consumer stopped", "consumer", c.config.ConsumerID)
	return nil
}

func (c *Consumer) Ack(ctx context.Context, message Message) error {
	if err := c.client.rdb.XAck(ctx, c.key, c.config.Group, message.ID).Err(); err != nil {
		return fmt.Errorf("redisstream: ack: %w", err)
	}
	ackMessageTotal.Inc()
	return nil
}

// ensureGroup creates the group, and the stream key with it, so a consumer can
// start before any producer has written.
func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.client.rdb.XGroupCreateMkStream(ctx, c.key, c.config.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("redisstream: create group: %w", err)
	}
	return nil
}

func (c *Consumer) readLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(f float64) {
			readMessageLatency.Observe(f)
		}))
		streams, err := c.client.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.config.Group,
			Consumer: c.config.ConsumerID,
			Streams:  []string{c.key, ">"},
			Count:    c.config.BatchSize,
			Block:    c.config.BlockTimeout,
		}).Result()
		timer.ObserveDuration()

		switch {
		case errors.Is(err, redis.Nil):
			continue
		case err != nil:
			if ctx.Err() != nil {
				return
			}
			if strings.Contains(err.Error(), "NOGROUP") {
				if createErr := c.ensureGroup(ctx); createErr != nil {
					log.Default().Errorw("redisstream cannot recreate group",
						"stream", c.config.Stream, "error", createErr)
				}
				continue
			}
			log.Default().Errorw("redisstream read failed",
				"stream", c.config.Stream, "error", err)
			sleep(ctx, readErrorBackoff)
			continue
		}

		for _, stream := range streams {
			if !c.deliver(ctx, stream.Messages) {
				return
			}
		}
	}
}

func (c *Consumer) claimLoop(ctx context.Context) {
	interval := c.config.MaxProcessingTime / 2
	if interval <= 0 {
		interval = DefaultMaxProcessingTime / 2
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.claim(ctx); err != nil && ctx.Err() == nil {
				log.Default().Errorw("redisstream claim failed",
					"stream", c.config.Stream, "error", err)
			}
		}
	}
}

func (c *Consumer) claim(ctx context.Context) error {
	entries, _, err := c.client.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.key,
		Group:    c.config.Group,
		Consumer: c.config.ConsumerID,
		MinIdle:  c.config.MaxProcessingTime,
		Start:    "0-0",
		Count:    c.config.BatchSize,
	}).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return nil
	case err != nil:
		return fmt.Errorf("redisstream: autoclaim: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	claimMessageTotal.Add(float64(len(entries)))
	c.deliver(ctx, entries)
	return nil
}

func (c *Consumer) deliver(ctx context.Context, entries []redis.XMessage) bool {
	for _, entry := range entries {
		message, err := messageFromEntry(entry)
		if err != nil {
			log.Default().Errorw("redisstream cannot decode entry, skipping",
				"stream", c.config.Stream, "id", entry.ID, "error", err)
			if ackErr := c.Ack(ctx, Message{ID: entry.ID}); ackErr != nil {
				log.Default().Errorw("redisstream cannot ack malformed entry",
					"stream", c.config.Stream, "id", entry.ID, "error", ackErr)
			}
			continue
		}

		select {
		case c.messages <- message:
			readMessageTotal.Inc()
		case <-ctx.Done():
			return false
		}
	}
	return true
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
