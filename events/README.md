# events

Consumer-side inbox pattern — the mirror of [outbox](../outbox/) for the read side of the event pipeline.

A **Consumer** (your project owns this) reads messages from Kafka and persists them into an `inbox` table (idempotent Insert on `event.id`), then commits the Kafka offset. A `Worker` (shipped by govern) polls the inbox and dispatches each row to a `Handler`. Handlers may participate in their own transaction for **exactly-once effect** on the database.

## Why inbox over inline consume-and-process

- **Decoupling** — Kafka offset commit doesn't wait for slow business logic. Consumer is a thin recorder; Worker paces the effects.
- **Idempotency built-in** — the primary key on `event.id` makes duplicate Kafka deliveries no-ops. No separate dedup table or middleware.
- **Exactly-once effect (opt-in)** — Handler can call your `Storage` impl's tx-aware `MarkProcessedTx` inside its business transaction, atomically committing the mark with its side effects. Redelivery after that becomes a no-op.
- **Backpressure / observability** — inbox size + `attempts` column shows lag and stuck rows directly in SQL.
- **Reprocessing** — replay events from the inbox without re-consuming from Kafka.

Trade-offs: one extra DB write per message; inbox table needs retention. Usually worth it in a DDD + saga setup.

## What govern ships (spec-first)

- **Types + interfaces** — `Record`, `Handler`, `InboxStorage`, `WorkerConfig`.
- **`Worker`** — polling worker that consumes any `InboxStorage`; `Start(ctx)` blocks until you cancel ctx.
- **Wire helpers** — `RecordFromMessage(payload, key, headers)` parses transport messages into `Record`s in a way that stays wire-compatible with `outbox.MessageHeaders`. `ContextFromHeaders(ctx, headers)` continues the upstream trace.

## What your project owns

- The **`InboxStorage` implementation** (Postgres / MySQL / …) — agent-generated per your DB stack. Schema spec is below.
- The **Consumer loop** — a ~30-line fetch loop around your Kafka client that calls `RecordFromMessage` + `Storage.Insert`. Example below.
- The **migration** for the inbox table.

## Installation

```bash
go get github.com/haipham22/govern/events
```

Then in your project:

1. Generate the `InboxStorage` impl + migration from the spec below (agent job).
2. Write a Consumer loop for your Kafka client library (example below).
3. Wire Consumer + Worker in `main` and run them in goroutines.

## InboxStorage — implement in your project

On Postgres, use your own pool and pass its transaction handle (a `*sql.Tx`,
a `pgx.Tx`, a GORM tx) to the storage implementation — that handle is
how a handler's effect and the inbox `MarkProcessed` land in the same transaction.

Interface:

```go
type InboxStorage interface {
    Insert(ctx context.Context, r Record) error
    Claim(ctx context.Context, limit int) ([]Record, error)
    MarkProcessed(ctx context.Context, ids ...uuid.UUID) error
    FailRetry(ctx context.Context, id uuid.UUID, cause error) error
}
```

For **exactly-once effect**, your impl should ALSO expose a tx-aware
`MarkProcessedTx(ctx, tx, ids...)` method on the concrete type (not on the
interface). The Handler holds a reference to your concrete impl and calls
that inside its business tx.

## Consumer loop — implement in your project

Example — `segmentio/kafka-go`:

```go
type Consumer struct {
    Reader  *kafka.Reader
    Storage events.InboxStorage
    Logger  *slog.Logger
}

func (c *Consumer) Start(ctx context.Context) error {
    for {
        raw, err := c.Reader.FetchMessage(ctx)
        if err != nil {
            if errors.Is(err, context.Canceled) { return nil }
            c.Logger.ErrorContext(ctx, "fetch", "error", err)
            select { case <-ctx.Done(): return nil; case <-time.After(500 * time.Millisecond): }
            continue
        }

        // headers → map[string]string
        h := make(map[string]string, len(raw.Headers))
        for _, kh := range raw.Headers { h[kh.Key] = string(kh.Value) }

        rec, err := events.RecordFromMessage(raw.Value, raw.Key, h)
        if err != nil {
            c.Logger.ErrorContext(ctx, "decode", "error", err)
            _ = c.Reader.CommitMessages(ctx, raw) // don't loop forever on a poison message
            continue
        }

        if err := c.Storage.Insert(ctx, rec); err != nil {
            c.Logger.ErrorContext(ctx, "inbox insert", "error", err)
            // don't commit — Kafka redelivers; Insert is idempotent on redelivery.
            select { case <-ctx.Done(): return nil; case <-time.After(500 * time.Millisecond): }
            continue
        }

        if err := c.Reader.CommitMessages(ctx, raw); err != nil {
            c.Logger.ErrorContext(ctx, "commit", "error", err)
        }
    }
}

func (c *Consumer) Shutdown(context.Context) error { return c.Reader.Close() }
```

For `franz-go`, `sarama`: swap the client, keep the helper calls.

## Wiring — main

```go
import (
    "github.com/haipham22/govern/events"
)

func main() {
    ctx := context.Background()

    storage := myproject.NewInbox(pool) // your InboxStorage impl

    consumer := &myproject.Consumer{
        Reader:  reader,
        Storage: storage,
    }

    worker := events.NewWorker(storage, handleOrderEvent(pool, storage), events.WorkerConfig{
        PollInterval: 500 * time.Millisecond,
        BatchSize:    100,
    })

    workerCtx, stopWorker := context.WithCancel(ctx)
    defer stopWorker()

    go func() { _ = consumer.Start(workerCtx) }()
    go func() { _ = worker.Start(workerCtx) }()

    // Blocks until SIGINT/SIGTERM; the deferred stopWorker then stops both.
    _ = httpSrv.Start()
}
```

## Handler — exactly-once effect

Your concrete `InboxStorage` impl exposes a tx-aware `MarkProcessedTx`. Call it inside your business tx so the mark and the side effect commit together. Any later Worker call to `Storage.MarkProcessed` is a no-op.

```go
func handleOrderEvent(store *myproject.Inbox) events.Handler {
    return func(ctx context.Context, r events.Record) error {
        tx, err := pool.Begin(ctx)
        if err != nil { return err }
        defer tx.Rollback(ctx)

        switch r.EventType {
        case "OrderPlaced":
            if err := reserveInventory(ctx, tx, r.Payload); err != nil { return err }
        case "OrderCancelled":
            if err := releaseInventory(ctx, tx, r.Payload); err != nil { return err }
        }

        // Mark IN the same tx — exactly-once effect.
        if err := store.MarkProcessedTx(ctx, tx, r.ID); err != nil { return err }
        return tx.Commit(ctx)
    }
}
```

### Handler — simple at-least-once effect

If the side effect is idempotent by construction (upsert, math), skip the tx dance — the Worker's post-return `MarkProcessed` is sufficient:

```go
func handleUserPresence(ctx context.Context, r events.Record) error {
    return updatePresence(ctx, r.Payload) // idempotent upsert
}
```

## Delivery semantics

- **Kafka → inbox**: at-least-once. Your `InboxStorage.Insert` MUST be idempotent (`ON CONFLICT DO NOTHING` on `id`), so a re-delivered Kafka message is a no-op in the table.
- **inbox → Handler**: at-least-once by default. For exactly-once effect on the database, have Handler call `MarkProcessedTx` inside its business transaction (pattern above).
- **Failure modes**:
  - Consumer can't reach DB → don't commit Kafka offset → Kafka re-delivers → retry.
  - Consumer commits Kafka offset but crashes before Worker runs → row is safely in inbox → Worker processes on next tick.
  - Worker Handler returns error → `FailRetry` writes `last_error`; row is re-claimed next tick. No backoff yet.

## Reprocessing / recovery

- **Replay one event**: `UPDATE inbox SET processed_at = NULL WHERE id = '...'` — Worker will re-run Handler.
- **Replay a range**: same, `WHERE received_at BETWEEN ... AND ...`.
- **Skip a poison message**: `UPDATE inbox SET processed_at = NOW() WHERE id = '...'`.
- **Inspect lag**: `SELECT COUNT(*) FROM inbox WHERE processed_at IS NULL`.
- **Find stuck rows**: `SELECT id, event_type, attempts, last_error FROM inbox WHERE attempts > 10 ORDER BY received_at`.

## Adding a new topic (consumer side)

Producer-side steps are in the [outbox README](../outbox/README.md#adding-a-new-topic). On the consumer side per new topic:

1. Provision the topic (done as part of producer rollout).
2. Add a new Kafka reader for the topic + a `GroupID` reflecting this consumer's identity.
3. Wire a Consumer with the SAME `InboxStorage` (one inbox table per service is standard — event IDs are globally unique).
4. Wire a `Worker` with a Handler that switches on `Record.EventType`.
5. Start both in goroutines alongside your other components.

Running many Consumers in one process is fine (one per topic subscription). Workers can also scale — SKIP LOCKED prevents overlap.

## Schema (spec for your InboxStorage impl)

Postgres 12+. Default table name is `inbox`.

### Columns

| Column           | Type          | Constraints                     | Purpose                                                                                       |
| ---------------- | ------------- | ------------------------------- | --------------------------------------------------------------------------------------------- |
| `id`             | `UUID`        | `PRIMARY KEY`                   | The `event.id` from producer headers — matches `outbox.id`. Enables ON CONFLICT dedup.       |
| `event_type`     | `TEXT`        | `NOT NULL`                      | Domain event name (`OrderPlaced`, ...). Handler dispatches on this.                           |
| `aggregate_id`   | `TEXT`        | `NOT NULL`                      | Aggregate root ID from the message key.                                                       |
| `aggregate_type` | `TEXT`        | `NOT NULL`                      | Aggregate type header value.                                                                  |
| `payload`        | `BYTEA`       | `NOT NULL`                      | Encoded event body — the SDK does not interpret it.                                           |
| `headers`        | `JSONB`       | nullable                        | All headers (including `traceparent` for OTel continuation by the Worker).                    |
| `received_at`    | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()`        | When Consumer wrote the row. `Claim` orders by this ascending.                                |
| `processed_at`   | `TIMESTAMPTZ` | nullable                        | Set by `MarkProcessed`. `NULL` = pending.                                                     |
| `attempts`       | `INT`         | `NOT NULL DEFAULT 0`            | Incremented on every `Claim`. Query stuck rows via `attempts > N`.                            |
| `last_error`     | `TEXT`        | nullable                        | Written by `FailRetry`. Latest Handler error message.                                         |

### Indexes

- **`inbox_pending_idx`** — partial index on `(received_at)` `WHERE processed_at IS NULL`.
  - Why: `Claim` scans only unprocessed rows ordered by `received_at`. Partial index keeps it tight even when history grows large.

### Rationale

- **`id` = event UUID, not a separate row ID** — dedup and Kafka correlation both use the same identifier.
- **No `topic` / `partition` / `offset` columns** — the inbox is a semantic layer above Kafka. If you need Kafka position for debugging, store it in `headers` from your Consumer.
- **`payload BYTEA` not JSONB** — encoding-agnostic; teams use protobuf or Avro as well as JSON.
- **`headers JSONB`** — small, ergonomic to query. `SELECT * FROM inbox WHERE headers->>'tenant' = 'x'`.
- **No `retry_at` column yet** — failed rows retry every poll. If a handler starts failing loudly, `attempts` grows and monitoring picks it up.

### Migration checklist (for the agent that generates the SQL)

1. `CREATE TABLE inbox` with the columns above.
2. Partial index on `(received_at) WHERE processed_at IS NULL`.
3. Retention policy — e.g. `DELETE FROM inbox WHERE processed_at < NOW() - INTERVAL '30 days'` as a scheduled job. Retention must exceed the max replay window.
4. One inbox table per service (default). Multi-tenant / per-schema layouts pass a custom table name to your Storage constructor.

## Config

`WorkerConfig`:

| Field          | Type           | Default          | Notes                                                    |
| -------------- | -------------- | ---------------- | -------------------------------------------------------- |
| `PollInterval` | `time.Duration`| `500ms`          | Sleep between polls when no rows are pending.            |
| `BatchSize`    | `int`          | `100`            | Max rows claimed per tick.                               |
| `Logger`       | `*slog.Logger` | `slog.Default()` | Lifecycle + error events.                                |

Wire helpers:

| Helper                                                    | Purpose                                                                        |
| --------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `RecordFromMessage(payload, key, headers) (Record, error)`| Parse a transport message into a `Record`. Errors if `event.id` is missing.    |
| `ContextFromHeaders(ctx, headers) context.Context`        | Extract W3C TraceContext + Baggage — continues the upstream trace.             |

## Notes

- Your Consumer relies on the `event.id` header. `outbox.MessageHeaders(ctx, r)` sets it — keep the two sides aligned.
- Consumer commits the Kafka offset AFTER a successful inbox Insert. From there on, the DB is source of truth for the message.
- For fanout (same topic consumed by multiple services), each service gets its own consumer group AND its own inbox table.
- To scale Workers, run multiple; SKIP LOCKED prevents duplicate claims. Per-aggregate ordering across events depends on Consumer running one partition per instance (Kafka partition assignment gives you this within a group).
