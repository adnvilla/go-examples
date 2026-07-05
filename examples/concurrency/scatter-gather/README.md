# Scatter-Gather

**Category:** concurrency
**Difficulty:** Beginner

## Objective

Contrast with the [worker-pool](../worker-pool/) pattern: instead of a fixed number of goroutines pulling from a shared queue, scatter-gather starts one goroutine per unit of work and gathers all the results once every goroutine finishes.

## Concepts Covered

- Fanning out N goroutines up front (`go fetch(...)` in a loop), one per task
- `sync.WaitGroup` to know when *all* of them have finished
- A buffered result channel sized to the number of goroutines, so no send blocks
- Closing the result channel after `wg.Wait()` so the final `range` can drain it and terminate

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
scatter-gather/
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

10 goroutines, each sleeping a random delay up to 500ms, then writing one line to `results`. Worker IDs and exact delays differ on every run, but since each goroutine writes to the buffered channel as soon as it finishes, and the final loop drains the channel in write order, the lines tend to come out sorted by delay (fastest first):

```
result from worker 6 (took 18ms)
result from worker 7 (took 21ms)
result from worker 4 (took 44ms)
result from worker 9 (took 110ms)
result from worker 1 (took 186ms)
result from worker 5 (took 191ms)
result from worker 3 (took 202ms)
result from worker 10 (took 315ms)
result from worker 2 (took 320ms)
result from worker 8 (took 408ms)
```

## Code Walkthrough

- `main` allocates a `results` channel buffered to exactly `n`, so every `fetch` goroutine can write its result and return without waiting for a reader.
- `wg.Add(n)` registers all 10 units of work before any goroutine starts, then the loop launches one `fetch` goroutine per task.
- Each `fetch` calls `wg.Done()` via `defer`, sleeps a random delay to simulate variable-latency work, and writes a formatted string to `results`.
- `wg.Wait()` blocks `main` until every goroutine has called `Done` — i.e., until all 10 results have been written.
- Only after `wg.Wait()` returns does `main` `close(results)` and range over it — closing is safe here specifically because all writers are guaranteed done.

## Common Pitfalls

- **Closing `results` before `wg.Wait()` returns.** If any `fetch` goroutine were still running, closing the channel out from under it would panic on its next send.
- **Sizing `results` smaller than `n`.** A `fetch` goroutine's send would block waiting for a reader that isn't there yet (nothing reads until after `wg.Wait()`), deadlocking the whole program.
- **One goroutine per task doesn't bound concurrency.** Unlike `worker-pool`, this pattern is only appropriate when the number of tasks is small/known — with thousands of tasks, spawning a goroutine each would exhaust resources. Use a worker pool instead when the fan-out count needs a ceiling.
- **`math/rand` without a seed** (as used here for the simulated delay) is fine for demonstration but must never be relied on for anything requiring reproducibility or security.

## References

- [Effective Go — Channels](https://go.dev/doc/effective_go#channels)
- [Go Blog — Concurrency is not Parallelism](https://go.dev/blog/waza-talk)

## Next Steps

- [concurrency/worker-pool](../worker-pool/) — bound concurrency with a fixed pool instead of one goroutine per task
- [concurrency/fan-out-timeout](../fan-out-timeout/) — add a per-goroutine deadline with `time.After`
- [errgroup](../../errgroup/) — the same fan-out/gather shape with structured error propagation
