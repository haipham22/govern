# outbox

Transactional outbox pattern — the foundation for reliable domain-event publishing (DDD choreography) and saga steps.

Producers write events into an `outbox` table **inside the same transaction** as the aggregate change; a background `Relay` polls pending records and hands each to a `Publisher`. Delivery is **at-least-once** — consumers must be idempotent (see the [events](../events/) inbox pattern).

## Why

Double-writes are unreliable: writing to the DB and then publishing to Kafka can succeed halfway. The outbox makes the event write part of the DB transaction; the Relay owns delivery.

## What govern ships (spec-first)

- **Types + interfaces** — `Record`, `Storage`, `Publisher`, `RelayConfig`.
- **`Relay`** — polling worker; `Start(ctx)` blocks until you cancel ctx.
- **Wire helpers** — `MessageKey(r)` and `MessageHeaders(ctx, r)` standardize the message key + `event.* / traceparent` headers your Publisher emits, independent of which Kafka client library you use.

## What your project owns

- The **`Storage` implementation** (Postgres / MySQL / …) — agent-generated per your DB stack. Schema spec is below.
- The **`Publisher` implementation** — a thin wrapper around your Kafka client (segmentio / franz-go / sarama / …) that calls the wire helpers. Example below.
- The **migration** for the outbox table (from the schema spec).

## Installation

```bash
go get github.com/haipham22/govern/outbox
```

Then in your project:

1. Generate the `Storage` impl + migration from the spec below (agent job).
2. Write a ~30-line `Publisher` impl for your Kafka client (example below).
3. Wire `Relay` + your Publisher in `main` and run it in a goroutine.

## Storage — implement in your project

The `outbox.Storage` interface covers only what the Relay consumes:

```go
type Storage interface {
    Claim(ctx context.Context, limit int) ([]Record, error)
    MarkSent(ctx context.Context, ids ...uuid.UUID) error
    FailRetry(ctx context.Context, id uuid.UUID, cause error) error
}
```

**Insert is NOT on the interface** — the producer-side write must participate
in the caller's business transaction, and the shape of "transaction" is
DB-specific (a `pgx.Tx`, a `*sql.Tx`, a Mongo session, ...). Your concrete
impl exposes its own tx-aware Insert method, and producers work against the
concrete type. Example shape:

```go
// Project-owned code (agent-generated).
type PgOutbox struct{ pool *pgxpool.Pool }

func (o *PgOutbox) InsertTx(ctx context.Context, tx pgx.Tx, records ...outbox.Record) error {
    // INSERT INTO outbox (...) VALUES (...) — see Schema section
}

// Also satisfies outbox.Storage for the Relay:
func (o *PgOutbox) Claim(ctx context.Context, limit int) ([]outbox.Record, error)     { ... }
func (o *PgOutbox) MarkSent(ctx context.Context, ids ...uuid.UUID) error              { ... }
func (o *PgOutbox) FailRetry(ctx context.Context, id uuid.UUID, cause error) error    { ... }
```

## Producer side (inside your transaction)

Any pool works — begin a transaction with your driver and hand the tx handle
to your `InsertTx` implementation, wrapping the aggregate write and the
outbox insert in one transaction.

```go
func (r *OrderRepo) Place(ctx context.Context, o *Order) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    // 1. Aggregate change
    if err := r.insertOrder(ctx, tx, o); err != nil { return err }

    // 2. Domain event in the SAME tx (via your concrete impl's tx-aware method)
    payload, _ := json.Marshal(OrderPlaced{OrderID: o.ID, TotalCents: o.Total})
    err = r.outbox.InsertTx(ctx, tx, outbox.Record{
        ID:            uuid.New(),
        AggregateID:   o.ID.String(),
        AggregateType: "order",
        EventType:     "OrderPlaced",
        Payload:       payload,
    })
    if err != nil { return err }

    return tx.Commit(ctx)
}
```

## Publisher — implement in your project

The `Publisher` interface is one method:

```go
type Publisher interface {
    Publish(ctx context.Context, r Record) error
}
```

Use `outbox.MessageKey(r)` for the message key (AggregateID → per-partition
order) and `outbox.MessageHeaders(ctx, r)` for the standard headers
(`event.id`, `event.type`, `aggregate.type` + injected W3C `traceparent`).
Header conventions stay identical to what consumers expect, regardless of
which Kafka client library you use.

Example — `segmentio/kafka-go`:

```go
type KafkaPublisher struct {
    Writer  *kafka.Writer
    TopicFn func(outbox.Record) string
}

func (p *KafkaPublisher) Publish(ctx context.Context, r outbox.Record) error {
    hs := outbox.MessageHeaders(ctx, r)
    kh := make([]kafka.Header, 0, len(hs))
    for k, v := range hs {
        kh = append(kh, kafka.Header{Key: k, Value: []byte(v)})
    }
    return p.Writer.WriteMessages(ctx, kafka.Message{
        Topic:   p.TopicFn(r),
        Key:     outbox.MessageKey(r),
        Value:   r.Payload,
        Headers: kh,
    })
}

func (p *KafkaPublisher) Shutdown(context.Context) error {
    return p.Writer.Close()
}
```

For `franz-go`, `sarama`, or NATS: swap the client, keep the helper calls.

## Relay wiring (usually in main)

```go
import (
    "github.com/haipham22/govern/outbox"
)

func main() {
    ctx := context.Background()

    store := myproject.NewOutbox(pool) // your Storage impl
    pub := &myproject.KafkaPublisher{  // your Publisher impl
        Writer:  writer,
        TopicFn: func(r outbox.Record) string { return "events." + r.AggregateType },
    }

    relay := outbox.NewRelay(store, pub, outbox.RelayConfig{
        PollInterval: 500 * time.Millisecond,
        BatchSize:    100,
    })

    relayCtx, stopRelay := context.WithCancel(ctx)
    defer stopRelay()
    defer pub.Shutdown(ctx) // flush + close writer

    go func() { _ = relay.Start(relayCtx) }()

    // Blocks until SIGINT/SIGTERM; the deferred stopRelay then stops the relay.
    _ = httpSrv.Start()
}
```

`Relay` is caller-driven — unlike a blocking server
and `redisstream.Consumer`, it does not install a signal handler of its own.
Cancel the context you gave `Start` to stop it.

## Adding a new topic

No code change in this package is needed — routing is a function of the
`TopicFn` you set on `KafkaPublisher`. The steps below take you from
"we want to emit a new event type" to "consumer reads it in prod".

### 1. Pick / confirm the routing convention

The producer side has already committed to one of these — reuse it, don't mix.

| Convention          | `TopicFn`                                               | Topic examples                                  | Trade-off                                                               |
| ------------------- | --------------------------------------------------------- | ----------------------------------------------- | ----------------------------------------------------------------------- |
| Per aggregate type  | `func(r) string { return "events." + r.AggregateType }` | `events.order`, `events.invoice`            | One topic per bounded-context aggregate. Fewer topics, coarser fan-out. |
| Per event type      | `func(r) string { return "events." + r.EventType }`     | `events.OrderPlaced`, `events.OrderShipped` | Finer subscriptions per consumer, more topics to manage.                |
| Static single topic | `nil` — set `Writer.Topic = "events.all"`            | `events.all`                                  | Simplest ops, consumers filter by header. Fine at low volume.           |

If the new event fits your existing convention, you have nothing to change in
the publisher wiring — go to step 2.

### 2. Provision the topic in Kafka

Ops action, not code. Standard config for domain-event topics:

- **Partition count** — pick from expected fan-out and per-key throughput. For per-aggregate ordering, all records with the same `AggregateID` land on the same partition, so `partitions = target parallelism` on the consumer side. Common starting point: 6–12.
- **Replication factor** — match your cluster policy (usually 3 in prod).
- **Retention** — long enough for the slowest consumer to catch up after a redeploy (often 7 days). Use `cleanup.policy=delete`. Don't use `compact` for event streams — it drops history.
- **Min in-sync replicas** — `min.insync.replicas=2` with `KafkaWriter.RequiredAcks = kafka.RequireAll` gives durable writes.

Example:

```bash
kafka-topics.sh --create --bootstrap-server kafka:9092 \
  --topic events.order --partitions 6 --replication-factor 3 \
  --config retention.ms=604800000 --config min.insync.replicas=2
```

If your cluster has `auto.create.topics.enable=true` you can skip this, but explicit is safer — auto-created topics inherit cluster defaults which are usually wrong for event streams.

### 3. Producer — insert Records with the right fields

Your business code writes an event Record inside the aggregate's transaction. The routing decision (step 1) reads the `TopicFn` inputs, so make sure they're set:

```go
outbox.Record{
    ID:            uuid.New(),
    AggregateID:   order.ID.String(),   // Kafka key → partition
    AggregateType: "order",             // used by TopicFn (per-aggregate convention)
    EventType:     "OrderShipped",      // used by TopicFn (per-event convention)
    Payload:       payload,             // JSON / protobuf bytes
    Headers:       map[string]string{"tenant": tenantID},
}
```

No new code in govern — the Relay picks it up next tick.

### 4. Consumer — subscribe to the new topic

Wire an `events.Consumer` (see [../events](../events/)) at the new topic name. If you're adding a *new* consumer service, that's a new deployment; if an existing service should also handle this topic, run a second `events.Consumer` alongside its existing one (different `GroupID`s if the subscriptions are independent).

### 5. Rolling out safely

- **New topic, greenfield** — provision → deploy producer → deploy consumer. No coordination.
- **Splitting one topic into many** — dual-publish during transition:
  1. Add the new topic + provision it.
  2. Change `TopicFn` to publish to BOTH old and new topics for affected records (write a `MultiPublisher`, or extend `TopicFn` to a slice). Deploy.
  3. Consumers migrate to the new topic when ready.
  4. When all consumers moved, drop the old-topic branch.
- **Renaming a topic** — same as split; two-phase dual-publish → cut over → drop old.

### 6. Verify in staging

- Insert one test Record; watch the outbox row transition `sent_at IS NULL` → `sent_at` populated (within one `PollInterval`).
- `kafka-console-consumer.sh --topic events.<new> --from-beginning` sees the message with the right headers (`event.id`, `event.type`, `aggregate.type`, plus `traceparent`).
- Consumer group lag drops to zero once your consumer is up.

### Common mistakes

- Publishing to a topic that doesn't exist — with auto-create off, `WriteMessages` errors and the record stays pending forever. Provision first.
- Setting `TopicFn` per-event-type but keeping only one partition — kills per-aggregate ordering across events of the same aggregate.
- Compacting an event-stream topic — silently deletes history; use `retention.ms` instead.

## Delivery semantics

- **At-least-once.** If publishing succeeds but the `MarkSent` UPDATE fails, the record is picked up again next tick.
- **Per-aggregate ordering** — as long as your Publisher uses `outbox.MessageKey(r)` for the message key, records for the same aggregate land on the same partition in `created_at` order. Multiple Relay workers preserve this thanks to SKIP LOCKED + partition key; a single Relay per pod is simpler.
- **No dead-letter yet.** Failed records stay pending with `last_error` set. Query `SELECT * FROM outbox WHERE attempts > 10 ORDER BY created_at` for stuck items.

## Consumer idempotency

Downstream consumers MUST dedup on the `event.id` header (which `outbox.MessageHeaders(ctx, r)` sets). The [events](../events/) package solves this via the inbox pattern — event ID is the primary key so duplicates are no-ops.

## Config

`RelayConfig`:

| Field          | Type             | Default          | Notes                                             |
| -------------- | ---------------- | ---------------- | ------------------------------------------------- |
| `PollInterval` | `time.Duration`  | `500ms`          | Sleep between polls when no records were claimed. |
| `BatchSize`    | `int`            | `100`            | Max records claimed per tick.                     |
| `Logger`       | `*slog.Logger`   | `slog.Default()` | For lifecycle + error logs.                       |

Wire helpers:

| Helper                          | Purpose                                                                              |
| ------------------------------- | ------------------------------------------------------------------------------------ |
| `MessageKey(r) []byte`          | Message key = `AggregateID`. Preserves per-aggregate partition order.                |
| `MessageHeaders(ctx, r) map`    | `event.id` / `event.type` / `aggregate.type` + injected W3C `traceparent` / `baggage`. |

## Schema (spec for your Storage impl)

Postgres 12+ (JSONB + partial index). Default table name is `outbox`.

### Columns

| Column           | Type          | Constraints                     | Purpose                                                                              |
| ---------------- | ------------- | ------------------------------- | ------------------------------------------------------------------------------------ |
| `id`             | `UUID`        | `PRIMARY KEY`                   | Event identity. Assigned by the producer (auto-fill with `uuid.New()` if empty).     |
| `aggregate_id`   | `TEXT`        | `NOT NULL`                      | Owning aggregate root ID. Used as message key (`MessageKey(r)`) — per-partition order. |
| `aggregate_type` | `TEXT`        | `NOT NULL`                      | Aggregate type, e.g. `"order"`. Emitted as header via `MessageHeaders`.               |
| `event_type`     | `TEXT`        | `NOT NULL`                      | Domain event name, e.g. `"OrderPlaced"`. Emitted as header via `MessageHeaders`.      |
| `payload`        | `BYTEA`       | `NOT NULL`                      | Encoded event body — JSON, Protobuf, Avro; the outbox does not interpret it.         |
| `headers`        | `JSONB`       | nullable                        | Optional string-keyed metadata (`map[string]string`).                                |
| `created_at`     | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()`        | Insertion time. `Claim` orders by this ascending, so per-aggregate order is stable.  |
| `sent_at`        | `TIMESTAMPTZ` | nullable                        | Set by `MarkSent` on successful publish. `NULL` = pending.                           |
| `attempts`       | `INT`         | `NOT NULL DEFAULT 0`            | Incremented on every `Claim`. Not a hard retry cap — used for observability.         |
| `last_error`     | `TEXT`        | nullable                        | Written by `FailRetry`. Latest publish error message.                                |

### Indexes

- **`outbox_pending_idx`** — partial index on `(created_at)` `WHERE sent_at IS NULL`.
  - Why: `Claim` scans only pending rows ordered by `created_at`. A partial index keeps it tiny even when history grows large, and dropping `sent_at IS NOT NULL` rows out of the tree removes the biggest source of bloat.

### Rationale

- **UUID PK, not serial** — records are assigned IDs before insert (so producers can log/return them), and Kafka header propagation carries UUIDs cleanly.
- **`payload BYTEA`, not JSONB** — the SDK is encoding-agnostic. Storing JSON in BYTEA avoids paying JSONB parsing/validation on the write path.
- **`headers JSONB`** — small, only user-supplied strings; JSONB is fine here and keeps queries ergonomic.
- **No `poisoned` / dead-letter column yet** — `last_error` + `attempts` is enough for querying stuck items (`WHERE attempts > 10`). Add DLQ when a real need emerges.
- **No `updated_at` / `claimed_by`** — `SELECT ... FOR UPDATE SKIP LOCKED` handles concurrent workers without needing lease bookkeeping.

### Migration checklist (for the agent that generates the SQL)

1. `CREATE TABLE outbox` with the columns above.
2. Partial index on `(created_at) WHERE sent_at IS NULL`.
3. Consider a retention policy — e.g. `DELETE FROM outbox WHERE sent_at < NOW() - INTERVAL '7 days'` as a scheduled job. The SDK does not clean up sent rows.
4. Multi-tenant / per-schema layouts: pass a custom table name to your Storage constructor.

## Notes

- Payloads are `[]byte` — JSON, Protobuf, Avro — the outbox doesn't care.
- Your `Storage` impl's tx-aware Insert (see [Storage — implement in your project](#storage--implement-in-your-project)) is what enforces atomicity between the aggregate change and the outbox row. The govern `Storage` interface intentionally omits Insert — the tx type is DB-specific.
- To scale beyond one Relay per pod, run multiple; SKIP LOCKED prevents duplicate claims. Per-aggregate ordering is still preserved when your Publisher uses `outbox.MessageKey(r)`.
