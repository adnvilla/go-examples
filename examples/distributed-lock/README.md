# Distributed Lock

**Category:** architecture
**Difficulty:** Advanced

## Objective

Build a distributed lock over a single Redis and demonstrate not just the happy path but the failure mode that makes naive implementations dangerous: **mutual exclusion** via atomic `SET NX PX`, **owner-only release/extend** via Lua compare-and-act scripts, **lease renewal** for work that outlives the TTL, and **fencing tokens** — the monotonic counter that protects the data when a paused holder's lock silently expires and it comes back believing it still owns the world.

## Concepts Covered

- `SET key value NX PX ttl` — atomic create-if-absent with expiry; the TTL is the liveness guarantee (a crashed holder can't wedge the system)
- Random owner tokens + Lua scripts for release/extend: `GET`-compare and `DEL`/`PEXPIRE` must be atomic, or a client races expiry and deletes the *next* holder's lock
- Lease renewal (heartbeat): extending well before expiry so the lock lives exactly as long as the work
- Fencing tokens (`INCR` per acquisition) and a downstream check that rejects stale fences — safety even after mutual exclusion has already been violated by an expiry
- The stale-holder scenario reproduced live: 100ms TTL, 250ms pause, a second holder, and both of the first holder's operations (release, write) correctly refused
- Env-guarded integration tests (`REDIS_LOCAL=1`), same convention as [dynamodb](../dynamodb/)

## Prerequisites

- Go 1.25+
- Redis on `localhost:6379` (this repo's compose provides it):
  ```bash
  docker compose up -d redis   # or, from this directory: make infra-up
  ```
- Dependency justified: `go-redis/v9` is the canonical Redis client and Redis is the lock store being taught

## Project Structure

```
distributed-lock/
├── go.mod
├── lock.go        # Acquire / Release / Extend + fencing (the reusable part)
├── main.go        # three scenarios, deterministic output
├── lock_test.go   # integration tests (REDIS_LOCAL=1)
├── Makefile
└── README.md
```

## How to Run

```bash
make infra-up   # start Redis
make run        # the three scenarios
make test       # integration tests (sets REDIS_LOCAL=1)
```

## Expected Output

```
--- mutual exclusion: one holder at a time ---
worker A: acquired (fence 1)
worker B: denied — lock is held
worker A: released
worker B: acquired after release (fence 2)

--- lease renewal: work longer than the TTL ---
worker: acquired with 200ms TTL (fence 3)
worker: slice 1 done, lease extended
worker: slice 2 done, lease extended
worker: slice 3 done, lease extended
  storage: accepted "renewal result" with fence 3
worker: released after ~450ms of work on a 200ms lease

--- stale holder: expiry + fencing tokens ---
worker C: acquired with 100ms TTL (fence 4)
worker C: pauses for 250ms (GC pause / network partition)...
worker D: acquired the expired lock (fence 5)
  storage: accepted "D's update" with fence 5
worker C: wakes; release refused (owner token no longer matches)
worker C: write rejected: fence 4 is stale (highest seen: 5)
worker D: released cleanly
```

## Code Walkthrough

- `Acquire` does two things atomically-enough: `SetNX` with a random 16-byte owner token and the TTL, then `INCR` on a companion fence counter. The owner token answers "is this still *my* lock?" to Redis; the fence answers "is this holder still the *newest*?" to everything downstream.
- `Release` and `Extend` are Lua scripts because the check and the action must be one atomic step. The classic bug they prevent: holder's lock expires → someone else acquires → original holder's plain `DEL` deletes the *new* holder's lock. With the script, the stale holder gets `ErrNotHeld` and nothing changes.
- `leaseRenewal` shows the heartbeat discipline: 150ms work slices under a 200ms TTL, extending after each slice. The TTL stays short (fast recovery if the holder dies) while ownership stays continuous for as long as the work actually runs. Production versions run the heartbeat in a goroutine; the demo keeps it inline for readability.
- `staleHolder` is the payoff scene. C's 100ms lock expires during a 250ms pause — mutual exclusion has *already failed* at this point, silently, from C's perspective. D acquires legitimately (fence 5) and writes. When C wakes: Redis refuses its release (token mismatch), and — the part the lock itself cannot do — `storage.write` refuses fence 4 because it has seen 5. That last check is the fencing insight: **the resource, not the lock, enforces the final safety**, because only the resource observes writes in order.
- The integration tests cover each property in isolation (exclusion, owner-only release after expiry, renewal keeping the lock past its nominal TTL, fence monotonicity across acquisitions) with per-test keys so they run in parallel.

## Common Pitfalls

- **Releasing with a plain `DEL`.** Deletes whoever's lock is there *now*, not necessarily yours. Owner token + Lua compare-and-delete is the minimum viable release.
- **No TTL ("we always release in a defer").** A SIGKILL, OOM, or network partition and the lock is held forever. The TTL is not optional; it's the liveness half of the design.
- **TTL without renewal for long work.** Pick short-TTL-plus-heartbeat over "TTL long enough for the slowest case" — the latter turns every crash into a slowest-case outage.
- **Trusting the lock alone for correctness.** Expiry under a paused holder means two processes *will* occasionally believe they hold the lock. If the protected resource can't check fencing tokens (or do an equivalent conditional write), the lock is an optimization, not a guarantee — design accordingly.
- **One Redis is one failure domain.** This is the single-instance pattern: correct against client failures, not against Redis failover losing the key (a replica promoted before replicating the lock). Multi-node schemes (Redlock) exist but are debated (see References); for correctness-critical exclusion, prefer a consensus store (etcd/ZooKeeper) or fencing at the resource.
- **Reusing one lock key for unrelated resources.** Contention and fence sequences get entangled; one key per protected resource.

## References

- [Redis docs — Distributed Locks with Redis](https://redis.io/docs/latest/develop/use/patterns/distributed-locks/)
- [Martin Kleppmann — How to do distributed locking](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html) (the fencing-token argument)
- [Antirez — Is Redlock safe?](http://antirez.com/news/101) (the counterpoint)
- [go-redis docs — Scripting](https://redis.uptrace.dev/guide/lua-scripting.html)

## Next Steps

- [redis](../redis/) — the task-queue example sharing this backing service
- [circuit-breaker](../circuit-breaker/) — a different failure-containment pattern; they often guard the same dependency
- [sync-primitives](../sync-primitives/) — in-process mutual exclusion, for contrast with the distributed kind
