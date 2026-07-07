# Fan-Out with Per-Worker Timeout

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show how to bound how long you'll wait for any single goroutine in a fan-out, using `select` racing the real result against `time.After`, so one slow worker can't stall the whole batch.

## Concepts Covered

- `select` racing two channels: a real result vs. a deadline
- `time.After` as a one-shot timeout channel
- Isolating each worker's timeout so slow workers degrade gracefully instead of blocking everything
- Fanning out N goroutines and gathering results through a buffered channel (same shape as [scatter-gather](../scatter-gather/))

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
fan-out-timeout/
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

Each of the 10 workers simulates a random delay up to 500ms against a 300ms timeout, so roughly 40% time out on average — the exact count and worker IDs vary between runs:

```
worker 1 done in 58ms
worker 2 done in 135ms
worker 4 done in 155ms
worker 7 done in 182ms
worker 6 done in 201ms
worker 9 timed out
worker 10 timed out
worker 8 timed out
worker 5 timed out
worker 3 timed out

5/10 workers timed out
```

## Code Walkthrough

- Each `worker` starts an inner goroutine that does the actual (simulated) work and writes to a private, buffered `done` channel.
- The outer `worker` function then `select`s between `<-done` (the real result) and `<-time.After(workerTimeout)`. Whichever fires first wins.
- If the timeout wins, `worker` sends a sentinel `result{timedOut: true, ...}` to `out` instead of the real value — the inner goroutine is left to finish on its own time and write to `done`, which nobody will read again (safe because `done` is buffered with capacity 1, so that final send doesn't block and the goroutine can exit).
- `main` fans out all 10 workers, waits for all of them via `wg.Wait()` (every `worker` call returns once it has produced *a* result — real or timeout — not once the inner goroutine finishes), then tallies how many timed out.

## Common Pitfalls

- **Unbuffered `done` channel.** If `done` weren't buffered, the inner goroutine's send would block forever after a timeout (nobody is still receiving), leaking the goroutine permanently. Buffering it with capacity 1 lets that final send always succeed.
- **Assuming the timed-out inner goroutine is cancelled.** `time.After` racing in a `select` does not stop the other branch's work — it only stops *waiting* for it. If the underlying work needs to actually be cancelled (e.g., an in-flight HTTP request), use `context.WithTimeout` instead, which propagates cancellation.
- **Reusing `time.After` in a loop.** Each call allocates a new timer that isn't garbage-collected until it fires; fine for a one-shot `select` like this, but wasteful in a hot loop — use `time.NewTimer` + `Stop()`/`Reset()` there instead.
- **Treating the timeout ratio as deterministic.** With `math/rand` unseeded delays, the exact count of timeouts (and which worker IDs) changes every run — don't write a test that asserts an exact count.

## References

- [Go Blog — Go Concurrency Patterns: Timing out, moving on](https://go.dev/blog/context)
- [Effective Go — Select](https://go.dev/doc/effective_go#select)

## Next Steps

- [context](../../context/) — replace `time.After` with `context.WithTimeout` for real cancellation propagation
- [concurrency/scatter-gather](../scatter-gather/) — the same fan-out shape without a timeout
- [errgroup](../../errgroup/) — structured error handling across a fan-out
