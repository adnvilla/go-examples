# PostgreSQL (pgx)

**Category:** database
**Difficulty:** Advanced

## Objective

Show PostgreSQL through `pgx` beyond CRUD, centered on the thing most applications get wrong without noticing: **transaction isolation**. The demo reproduces a textbook write-skew anomaly live — two transactions that each check an invariant, write *disjoint* rows, and both commit under `READ COMMITTED`, breaking the invariant — then runs the identical schedule under `SERIALIZABLE`, where PostgreSQL rejects one with SQLSTATE `40001` and a retry preserves correctness. Plus `LISTEN/NOTIFY`: pub/sub straight from the database, no broker.

## Concepts Covered

- `pgxpool` — pgx's native pool (vs going through `database/sql`), `$1` positional placeholders (vs `?` in [sqlite](../sqlite/)/[mysql](../mysql/))
- `BeginTx` with `pgx.TxOptions{IsoLevel: ...}` and what the levels actually promise — read committed allows write skew because no row is written twice; nothing blocks, nothing conflicts, and the invariant still breaks
- SQLSTATE `40001` (`serialization_failure`) as a *normal* signal meaning "retry the whole transaction", detected with `errors.As` on `*pgconn.PgError` — and why the retry must re-read (the fresh read is what declines the second withdrawal)
- The `defer tx.Rollback(ctx)` idiom: harmless after commit, essential on every early return
- `LISTEN`/`NOTIFY` with a dedicated pooled connection and `WaitForNotification` — push, not polling
- Env-guarded integration tests (`POSTGRES_LOCAL=1`) pinning *both* isolation outcomes, so the demo's premise is verified, not narrated

## Prerequisites

- Go 1.25+
- PostgreSQL on `localhost:5432` (this repo's compose provides it):
  ```bash
  docker compose up -d postgres   # or, from this directory: make infra-up
  ```
  User `postgres`, password `secret`, database `examples`.
- Dependency justified: `pgx/v5` is the canonical PostgreSQL driver and the topic being taught

## Project Structure

```
postgres/
├── go.mod
├── main.go        # write-skew under two isolation levels + LISTEN/NOTIFY
├── main_test.go   # integration tests (POSTGRES_LOCAL=1)
├── Makefile
└── README.md
```

## How to Run

```bash
make infra-up   # start Postgres
make run        # the demos
make test       # integration tests (sets POSTGRES_LOCAL=1)
```

## Expected Output

```
--- schema: two doctors, both on call; invariant: at least one on call ---

--- write skew under read committed ---
alice's tx: sees 2 doctors on call -> ok to leave
bob's tx: sees 2 doctors on call -> ok to leave
alice's tx: committed
bob's tx: committed
doctors on call now: 0 -> invariant BROKEN

--- write skew under serializable ---
alice's tx: sees 2 doctors on call -> ok to leave
bob's tx: sees 2 doctors on call -> ok to leave
alice's tx: committed
bob's tx: rejected with SQLSTATE 40001 (serialization failure) — retrying
bob's retry: sees only 1 doctor on call -> stays on call
doctors on call now: 1 -> invariant PRESERVED

--- LISTEN/NOTIFY: pub/sub from the database ---
listener: subscribed to channel order_events
listener: received "order-1001 shipped" on channel "order_events"
```

## Code Walkthrough

- `writeSkew` runs the exact anomaly schedule from the literature (Kleppmann's on-call doctors): both transactions read the invariant (`COUNT(*) WHERE on_call` = 2), each updates **its own row**, both attempt to commit. Because the write sets are disjoint, no lock is ever contended and `READ COMMITTED` happily commits both — the invariant ("someone is on call") breaks *silently*. This is the crucial lesson: the default isolation level doesn't just reorder things, it permits outcomes no serial execution could produce, and no error tells you.
- Under `SERIALIZABLE`, PostgreSQL's SSI tracks the read/write dependency cycle and rejects the second commit with `40001`. The code treats that error at either the `UPDATE` or the `COMMIT` (version-dependent) as one outcome, because the contract is the same: *the whole transaction must be retried from the top*. `retryBob` re-reads — sees only 1 doctor on call — and declines. Retrying only the write, without re-reading, would re-break the invariant.
- `isSerializationFailure` uses `errors.As` on `*pgconn.PgError` and compares the SQLSTATE — the same wrapped-error discipline as everywhere else in this repo, applied to database error codes instead of sentinels.
- `listenNotify` acquires a dedicated connection from the pool for `LISTEN` (notifications are delivered per-session, so that connection can't be returned while subscribed), fires `pg_notify` from any other session, and receives the payload via `WaitForNotification` — a real push channel that's often enough to replace a broker for cache invalidation or job wake-ups.
- The integration tests pin both outcomes: read committed **must** break the invariant (if that test fails, the demo's premise changed) and serializable **must** preserve it. Testing the anomaly, not just the fix, is what keeps the example honest.

## Common Pitfalls

- **Assuming the default isolation level prevents anomalies your code cares about.** `READ COMMITTED` permits write skew, lost updates via read-modify-write, and phantoms. Either take explicit locks (`SELECT ... FOR UPDATE` — which fixes read-modify-write but *not* this disjoint-row skew), or use `SERIALIZABLE` and retry.
- **Treating `40001` as a failure.** It's the mechanism working. Serializable code without a retry loop is incomplete; production versions retry with a bounded attempt count and backoff.
- **Retrying just the statement instead of the whole transaction.** The point of the retry is the *fresh read*; replaying the write against stale premises recreates the anomaly.
- **`LISTEN` on a pooled connection you release.** The subscription dies with the session, and notifications route to whichever session listened — hold a dedicated connection (`pool.Acquire`) for the listener's lifetime, and expect to re-`LISTEN` after reconnects.
- **`NOTIFY` as a delivery guarantee.** Notifications are fire-and-forget: no persistence, no replay for disconnected listeners. Pair it with a table (the notification says "look", the table holds the truth) — the outbox shape.
- **Forgetting `defer tx.Rollback(ctx)`.** Every early-return path leaks an open transaction otherwise; after a successful commit the rollback is a harmless no-op.

## References

- [PostgreSQL docs — Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html)
- [pgx documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [PostgreSQL docs — NOTIFY](https://www.postgresql.org/docs/current/sql-notify.html)
- Designing Data-Intensive Applications (Kleppmann), ch. 7 — the on-call doctors example

## Next Steps

- [sqlite](../sqlite/) / [mysql](../mysql/) — the `database/sql` counterparts at Beginner level
- [distributed-lock](../distributed-lock/) — a different answer to "two workers, one invariant"
- [otel](../otel/) — where query spans would go in an instrumented service
