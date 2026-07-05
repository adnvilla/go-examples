# Sync Primitives

**Category:** synchronization
**Difficulty:** Intermediate

## Objective

Show three `sync`/`sync/atomic` tools that solve specific coordination problems more cheaply than a general-purpose mutex: `sync.Once` for exactly-once initialization, `sync.Map` for a concurrent map with specific access patterns, and `atomic` for lock-free counters and compare-and-swap.

## Concepts Covered

- `sync.Once.Do` — runs its function exactly once no matter how many goroutines call it concurrently
- `sync.Map` — a concurrent map optimized for write-once/read-many or disjoint-key-per-goroutine access patterns
- `atomic.Int64`/`atomic.Int32` — lock-free counters (`Add`, `Load`) and `CompareAndSwap` for conditional updates

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
sync-primitives/
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

The `sync.Map` section's line order is not deterministic (`Range` makes no ordering guarantee) — everything else is fixed:

```
=== sync.Once ===
service initialised
once: all goroutines got singleton

=== sync.Map ===
syncMap: key-3 = 9
syncMap: key-2 = 4
syncMap: key-4 = 16
syncMap: key-1 = 1
syncMap: key-0 = 0

=== atomic ===
atomic counter: 1000
CAS (0→1): true
CAS (0→2, should fail): false
```

## Code Walkthrough

- **`sync.Once`**: five goroutines all call `getService()` concurrently; `once.Do` guarantees the closure inside it (which prints `"service initialised"` and sets `serviceInstance`) runs exactly once — the message prints once no matter how many goroutines race to call it first.
- **`sync.Map`**: five goroutines each `Store` a distinct key (`key-0` through `key-4`) — a disjoint-key write pattern, one of the two cases `sync.Map` is optimized for (the other being write-once/read-many, e.g. a cache). `Range` iterates all stored key-value pairs afterward, in unspecified order.
- **`atomic`**: 1000 goroutines each call `counter.Add(1)` on an `atomic.Int64` with no mutex at all — the final `Load()` is always exactly 1000, since `Add` is a single atomic hardware operation with no possibility of a lost update (unlike a plain `int++`, which is not safe for concurrent access). `CompareAndSwap(expected, new)` then shows conditional update: the first call succeeds (`0 → 1`, since the current value is `0`), the second fails (it still expects `0`, but the value is now `1`).

## Common Pitfalls

- **Using `sync.Map` as a default choice over a plain `map` + `sync.RWMutex`.** The Go documentation explicitly recommends the mutex-guarded plain map for most cases — `sync.Map` only wins for its two specific access patterns (write-once/read-many, or disjoint-key-per-goroutine); a mutex-guarded map is usually simpler and just as fast otherwise.
- **Assuming `sync.Once` re-runs if the first call panics.** If the function passed to `Do` panics, `Once` still considers itself "done" — a subsequent `Do` call will not retry it.
- **Reaching for `atomic` for anything beyond a single value.** Atomics operate on one variable at a time; coordinating multiple related fields consistently generally needs a mutex instead, since there's no way to atomically update several fields together.
- **Plain `int++`/`counter++` from multiple goroutines.** That's a data race — increments can be lost when two goroutines read the same value before either writes back. `atomic.Int64.Add` (or a mutex) is required for a correct concurrent counter.

## References

- [sync package docs](https://pkg.go.dev/sync)
- [sync/atomic package docs](https://pkg.go.dev/sync/atomic)
- [sync.Map docs — "the Map type is optimized for two common use cases"](https://pkg.go.dev/sync#Map)

## Next Steps

- [share-memory-by-communicating](../share-memory-by-communicating/) — the channel-based alternative to a mutex-guarded shared map
- [errgroup](../errgroup/) — coordinating goroutines with structured error propagation instead of raw `sync.WaitGroup`
