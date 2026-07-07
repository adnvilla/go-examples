# Transactional Outbox

**Category:** architecture
**Difficulty:** Advanced

## Objective

Implement the transactional outbox pattern and demonstrate the problem it solves *and* the guarantee it actually provides. The dual-write bug is reproduced first (commit the order, crash before publishing — the event silently never existed); then the fix: state and event committed in **one transaction**, drained later by a relay. The residual crash window (published but not marked) is shown honestly too — it makes delivery **at-least-once**, which the idempotent consumer absorbs by deduplicating on event ID. Runs entirely on in-memory SQLite: no broker, no Docker.

## Concepts Covered

- The dual-write problem: two systems (database + broker), no transaction spanning both, and a crash between the writes losing one side silently
- The outbox table as the fix: events ride the business transaction, so rollback removes both and commit persists both
- The relay/poller loop: `SELECT ... WHERE dispatched = 0` → publish → mark — and why the order of those last two steps decides the delivery guarantee
- At-least-once + idempotent consumer (dedupe by event ID) as the honest contract — exactly-once is not on the menu
- Crash-window testing: the "relay dies between publish and mark" scenario as a first-class test case, not a footnote

## Prerequisites

- Go 1.25+
- No external services, no environment variables — the single dependency (`modernc.org/sqlite`, pure Go) stands in for any transactional store

## Project Structure

```
outbox-pattern/
├── go.mod
├── outbox.go        # Store (PlaceOrder, PendingEvents, MarkDispatched), Broker, Relay
├── main.go          # dual-write bug, outbox write, relay, crash window — in sequence
├── outbox_test.go   # atomicity, happy path, crash-window redelivery
├── Makefile
└── README.md
```

## How to Run

```bash
make run
make test
```

## Expected Output

```
--- dual write, the broken way: commit, then crash before publish ---
app: order committed
app: CRASH before broker.Publish — the event never existed
result: 1 order in the database, 0 events delivered -> silently inconsistent

--- outbox write: state + event in ONE transaction ---
app: order 2 and its event committed atomically
outbox: 1 event pending, safe on disk next to the order

--- relay: poll pending -> publish -> mark dispatched ---
  broker: delivered event 1: order.placed {"order_id": 2, "item": "trackball"}
outbox: 0 events pending after relay run

--- crash window: publish succeeded, mark didn't -> redelivery ---
app: order 3 and its event committed atomically
relay run 1 (crashes mid-flight):
  broker: delivered event 2: order.placed {"order_id": 3, "item": "desk mat"}
  relay: CRASH after publish, before mark — event stays pending
relay run 2 (after restart):
  broker: duplicate event 2 ignored (idempotent consumer)
result: 2 unique events delivered, 1 duplicate absorbed, outbox drained
```

## Code Walkthrough

- `dualWriteProblem` is the motivation, made concrete: the order commits, the process "crashes" before `broker.Publish`, and the punchline is that **nothing records the loss** — no error, no retry queue, just an order whose downstream consumers never hear about it. Every system that writes to a database and then publishes to a broker has this window.
- `Store.PlaceOrder` is the pattern's whole trick: `INSERT INTO orders` and `INSERT INTO outbox` inside one `BeginTx`/`Commit`. The event is now exactly as durable as the state it describes; there is no window because there is only one write. Note what's absent: no broker client anywhere near the business transaction.
- `Relay` is deliberately boring — read pending, publish, mark — because the interesting part is the *order*: publish **then** mark. Mark-then-publish would turn a crash into a lost event (at-most-once); publish-then-mark turns it into a duplicate (at-least-once). The pattern chooses duplicates, because duplicates are recoverable.
- `Broker.Publish` carries the consumer half of the contract: it remembers event IDs and drops redeliveries. The demo's final line ("2 unique, 1 duplicate absorbed") is at-least-once working as designed, not a bug being tolerated.
- The tests pin each property separately: atomicity of `PlaceOrder`, the happy path draining exactly once (with an idempotent no-op rerun), and the crash window leaving the event pending for the next run to redeliver.

## Common Pitfalls

- **Publishing inside the business transaction.** The broker call can succeed while the transaction later rolls back — now you've announced state that doesn't exist. The outbox exists precisely so the broker is *never* inside the transaction.
- **Marking dispatched before publishing.** Silently converts the guarantee to at-most-once; the crash between mark and publish loses the event with no trace — the same bug the pattern was adopted to fix.
- **Consumers that assume exactly-once.** Redelivery is a *when*, not an *if* (relay restarts, broker retries). Dedupe by event ID, or make the handler naturally idempotent.
- **No ordering key in the outbox.** Draining in `id` order preserves per-aggregate ordering only if one relay runs at a time; parallel relays or sharded outboxes need an ordering key (e.g., aggregate ID) the same way [kafka](../kafka/) needs message keys.
- **Letting the outbox grow forever.** Dispatched rows need retention (delete or archive after N days); an unbounded outbox becomes the biggest table in the database.
- **Polling as the only trigger at scale.** Polling is fine (and this simple) for most systems; high-volume setups pair the same table with change-data-capture (e.g., Debezium) instead of a poll loop — the table contract stays identical.

## References

- [microservices.io — Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Debezium — Outbox event router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html)
- Designing Data-Intensive Applications (Kleppmann), ch. 11 — exactly-once and idempotence

## Next Steps

- [kafka](../kafka/) — the real broker this relay would publish to, with the same at-least-once discipline on the consumer side
- [postgres](../postgres/) — the transaction machinery the outbox write relies on
- [sqlite](../sqlite/) — the `database/sql` fundamentals used here
