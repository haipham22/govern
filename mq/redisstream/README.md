# redisstream

A small wrapper over Redis Streams consumer groups: at-least-once delivery,
automatic recovery of a crashed consumer.s in-flight entries, and a
graceful-shutdown shape that fits a supervisor.

A **`Stream`** (producer) appends entries with `XADD`. A **`Consumer`** joins a
consumer group, reads with `XREADGROUP`, and hands messages to your code on a
channel. It fits the `graceful.Service` shape:
`Start(ctx)` blocks until ctx is canceled, `Shutdown(ctx)` stops it
programmatically.

Redis balances a group's entries across its members, so **running several
consumers in one group is how you scale** — and it is also why there is no
ordering guarantee between them (see [Delivery semantics](#delivery-semantics)).

## Installation

```bash
go get github.com/haipham22/govern/mq/redisstream
```

## Usage

### Connect

```go
client, err := redisstream.New(ctx, redisstream.Config{
    Addr:      "localhost:6379",
    Namespace: "orders-svc",
})
if err != nil {
    return err
}
defer client.Close()
```

### Produce

```go
stream, err := redisstream.NewStream(client, redisstream.StreamConfig{
    Name:   "orders",
    MaxLen: 1000,
})
if err != nil {
    return err
}

id, err := stream.Send(ctx, redisstream.Message{
    Key:     record.AggregateID,                 // = outbox.MessageKey
    Payload: payload,
    Headers: outbox.MessageHeaders(ctx, record), // event.* + traceparent
})
```

`NewStream` touches no Redis — the stream key is created by the first `XADD`, or
by a consumer's `XGROUP CREATE MKSTREAM`. Use `SendBatch` to pipeline several
messages; its returned IDs are positional.

### Consume

```go
consumer, err := redisstream.NewConsumer(client, redisstream.ConsumerConfig{
    Stream: "orders",
    Group:  "fulfilment",
})
if err != nil {
    return err
}

go func() {
    for message := range consumer.Messages() {
        if err := handle(ctx, message); err != nil {
            continue // no ack — redelivered after MaxProcessingTime
        }
        if err := consumer.Ack(ctx, message); err != nil {
            log.Default().Errorw("ack failed", "id", message.ID, "error", err)
        }
    }
}()

// Start blocks until ctx is canceled — run it in its own goroutine, or via
// graceful.Manager.
return consumer.Start(ctx)
```

To stop it from your own code instead of by signal:

```go
go func() { _ = consumer.Start(ctx) }()
...
if err := consumer.Shutdown(ctx); err != nil {   // waits for the reader to quiesce
    log.Default().Errorw("consumer shutdown timed out", "error", err)
}
```

`Messages()` is closed when `Start` returns, so the range loop ends on shutdown.
`Shutdown` blocks until the reader has quiesced — bounded by `ctx`, or by
`ShutdownTimeout` when `ctx` has no deadline — and leaves the Redis client open,
since it may be shared with the rest of the service. Close that through
`Client.Close`.

### Consumer idempotency

Duplicates are a question of when, not if. Three things produce them:

1. **A failed `Ack`** — the entry stays in the group's pending list and is
   claimed later.
2. **A crashed consumer** — same thing, picked up by a peer.
3. **A slow handler** — `XAUTOCLAIM` takes the entry after `MaxProcessingTime`
   *while your handler is still running*, so two consumers genuinely process it
   at the same time.

That third case is the one that decides the design: any dedup mechanism has to
be an atomic claim, never a read-then-write. `SELECT`-then-`INSERT` has both
racers see "not present" and both proceed; a unique index has Postgres pick a
winner.

So the answer this module ships is the [`events`](../../events/) inbox, where dedup
is a constraint rather than a code path. Your `InboxStorage.Insert` is:

```sql
INSERT INTO inbox (id, event_type, aggregate_id, aggregate_type, payload, headers, received_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (id) DO NOTHING
```

with `id` holding the producer's `event.id`. A duplicate affects zero rows and
returns nil — a no-op is success, not an error.

`Message` is shaped to feed that directly: `Key` carries what
`outbox.MessageKey` would have (the aggregate ID) and `Headers` is what
`outbox.MessageHeaders` wrote. The consumer loop stays a thin recorder with no
"have I seen this?" branch anywhere in it:

```go
for message := range consumer.Messages() {
    ctx := events.ContextFromHeaders(ctx, message.Headers) // continues the trace

    record, err := events.RecordFromMessage(message.Payload, []byte(message.Key), message.Headers)
    if err != nil {
        // missing or malformed event.id — a producer bug, not a retryable one
        log.Default().Errorw("undecodable event, dropping", "id", message.ID, "error", err)
        _ = consumer.Ack(ctx, message)
        continue
    }

    if err := storage.Insert(ctx, record); err != nil {
        continue // no ack — redelivered, and the retry no-ops if the row landed
    }
    if err := consumer.Ack(ctx, message); err != nil {
        log.Default().Errorw("ack failed", "id", message.ID, "error", err)
    }
}
```

If `Insert` succeeds and `Ack` then fails, redelivery re-inserts, hits the
conflict, no-ops, and acks — the loop self-heals.

An `events.Worker` then polls the inbox and paces the actual side effects, so a
slow handler never holds up the read loop.

**Dedup on `event.id`, not on `Message.ID`.** A reclaimed entry keeps its Redis
ID, but a producer that retries `Send` writes a *new* entry carrying the *same*
`event.id`. Only the header-based key catches that; keying on `Message.ID` would
let it through as two distinct events.

**The inbox dedups receipt, not effect.** For an exactly-once *effect*, the
handler calls your concrete impl's `MarkProcessedTx(ctx, tx, r.ID)` inside its
business transaction, so the mark and the side effect commit atomically. Without
it you get idempotent receipt with at-least-once effects — often fine, but it is
the difference between "the event is recorded once" and "inventory is reserved
once".

Two operational notes:

- **Inbox retention must exceed your replay window.** Once a processed row is
  purged its `event.id` no longer conflicts, so a very late redelivery reads as
  new. The 30 days `events` suggests dwarfs anything this package can produce.
- **`Insert` failing is not a dedup question**, it is a database outage. Do not
  ack; let `MaxProcessingTime` lapse and have the entry reclaimed.

### Sharing an existing Redis connection

```go
consumerClient := redisstream.NewWithClient(existingRedisClient, redisstream.Config{
    Namespace: "orders-svc",
})
```

`Close` is a no-op for such a client: whoever created the connection still owns it.

### Readiness

```go
health.Register("redis", client.HealthCheck())            // wire into your health registry
prometheus.MustRegister(redisstream.RedisStreamMetricsCollectors...)
```

## Key layout

| Key             | Type   | Purpose                            |
| --------------- | ------ | ---------------------------------- |
| `{ns}:{stream}` | stream | The `XADD` / `XREADGROUP` target.  |

That is the whole layout. Group membership, per-consumer pending entries and
their idle times live inside the stream key — Redis manages them, this package
does not. `Namespace` prefixes the key so several logical systems can share one
Redis safely.

## Config

### `Config` — connection

| Field          | Type            | Default          | Notes                                                     |
| -------------- | --------------- | ---------------- | --------------------------------------------------------- |
| `Addr`         | `string`        | `localhost:6379` | Comma-separated; several addresses use a cluster client.   |
| `Username`     | `string`        | —                | `json:"-"`                                                 |
| `Password`     | `string`        | —                | `json:"-"`                                                 |
| `DB`           | `int`           | `0`              |                                                            |
| `Namespace`    | `string`        | `redisstream`    | Prefix for every key this package touches.                 |
| `DialTimeout`  | `time.Duration` | `5s`             |                                                            |
| `ReadTimeout`  | `time.Duration` | `10s`            | **Must exceed `BlockTimeout`** — `NewConsumer` enforces it. |
| `WriteTimeout` | `time.Duration` | `3s`             |                                                            |
| `PoolSize`     | `int`           | go-redis default |                                                            |

### `StreamConfig`

| Field    | Type     | Default | Notes                                                 |
| -------- | -------- | ------- | ----------------------------------------------------- |
| `Name`   | `string` | —       | Required.                                              |
| `MaxLen` | `int64`  | `1000`  | Approximate trim (`MAXLEN ~`). `0` disables trimming.  |

### `ConsumerConfig`

| Field               | Type            | Default             | Notes                                                            |
| ------------------- | --------------- | ------------------- | ---------------------------------------------------------------- |
| `Stream`            | `string`        | —                   | Required.                                                         |
| `Group`             | `string`        | —                   | Required.                                                         |
| `ConsumerID`        | `string`        | `<hostname>-<uuid>` | Must be unique across processes and stable for the process life.  |
| `BatchSize`         | `int64`         | `10`                | `XREADGROUP COUNT`.                                               |
| `BlockTimeout`      | `time.Duration` | `5s`                | `XREADGROUP BLOCK`. Must be shorter than `Config.ReadTimeout`.    |
| `MaxProcessingTime` | `time.Duration` | `30s`               | Unacked longer than this ⇒ reclaimable (`XAUTOCLAIM MINIDLE`).    |
| `BufferSize`        | `int`           | `100`               | Capacity of the `Messages()` channel.                             |
| `ShutdownTimeout`   | `time.Duration` | `10s`               | Bound applied by `Shutdown` when its context carries no deadline. |

## Delivery semantics

- **At-least-once.** A message is redelivered until `Ack` succeeds. Handlers must
  be idempotent — see [Consumer idempotency](#consumer-idempotency).
- **No ordering guarantee.** Redis hands each entry to whichever consumer in the
  group asks first, so two events for the same entity can be processed
  concurrently by two consumers. If you need per-entity serialisation, take a
  lock on the entity in your handler, or run a single consumer.
- **Crash recovery is by claim, not by ownership.** Every consumer runs
  `XAUTOCLAIM` on a `MaxProcessingTime / 2` tick, so entries a dead consumer took
  but never acked are picked up by whoever is still running. A slow handler that
  exceeds `MaxProcessingTime` has its entry claimed too — size that budget
  generously.
- **A malformed entry is acked away**, not redelivered — an entry this package
  cannot decode would otherwise be reclaimed forever. It is logged first.

## Notes

- Logging goes through [`log`](../../log/)'s `*zap.SugaredLogger`; metrics
  are declared but never auto-registered — register
  `RedisStreamMetricsCollectors` yourself.
- A consumer creates its group with `XGROUP CREATE MKSTREAM` at startup, so it
  can start before any producer has written, and recreates it if a `NOGROUP`
  error shows up later (the stream key having been trimmed or flushed away).
- Unit tests here touch no Redis. The end-to-end tests are behind a build tag:

  ```bash
  docker run --rm -d -p 6379:6379 --name govern-redis redis:7
  go test ./mq/redisstream/ -tags e2e -v  # REDIS_ADDR overrides the address
  docker rm -f govern-redis
  ```

  They cover a produce/consume round trip, two consumers sharing one group's
  load without duplicates, reclaim of unacked entries, and shutdown closing
  `Messages()`.
