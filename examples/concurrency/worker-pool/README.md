# Worker Pool

**Category:** concurrency
**Difficulty:** Beginner

## Objective

Show the worker pool pattern: a fixed number of goroutines pull jobs off a shared channel, so the amount of concurrent work is bounded regardless of how many jobs are queued.

## Concepts Covered

- Bounding concurrency with a fixed pool of goroutines
- Distributing work through a shared, buffered `chan int` (`jobs`)
- Collecting results through a second buffered channel (`results`)
- Closing the jobs channel to let every worker's `range` loop exit

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
worker-pool/
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

5 jobs, 3 workers, one second of simulated work per job — the run takes roughly 2 seconds. The exact interleaving and completion order vary between runs (see Common Pitfalls), but a typical run looks like:

```
worker 3 started  job 1
worker 1 started  job 2
worker 2 started  job 3
worker 2 finished job 3
worker 2 started  job 4
worker 1 finished job 2
worker 1 started  job 5
result: 6
result: 4
worker 3 finished job 1
result: 2
worker 1 finished job 5
worker 2 finished job 4
result: 10
result: 8
```

## Code Walkthrough

- `main` creates two buffered channels sized to the job count: `jobs` (work items) and `results` (their doubled values).
- Three `worker` goroutines are started before any job is sent. Each blocks on `for j := range jobs`, so it stays idle until work arrives.
- `main` sends all 5 jobs into `jobs`, then calls `close(jobs)` — closing is what lets every worker's `range` loop terminate once the channel is drained, instead of blocking forever.
- Each worker simulates one second of work (`time.Sleep`) per job, then writes `j * 2` to `results`.
- `main` reads exactly `numJobs` values off `results` to know when all work is done, without needing a `sync.WaitGroup`.

## Common Pitfalls

- **Expecting a specific interleaving.** Which worker picks up which job — and the order results arrive in — depends on goroutine scheduling. Don't assert on exact output order; assert on the *set* of results if you were to test this.
- **Under-sizing the buffered channels.** If `jobs` or `results` were unbuffered (or too small), sends could block in ways that change the concurrency behavior; here both are sized to `numJobs` so no send blocks.
- **Forgetting `close(jobs)`.** Without it, every worker's `range jobs` blocks forever after the last job, leaking all 3 goroutines.
- **Reading fewer than `numJobs` results.** `main` must drain exactly as many results as jobs sent, or it will exit before workers finish (and, in a longer-lived program, leak the remaining worker goroutines).

## References

- [Go by Example — Worker Pools](https://gobyexample.com/worker-pools)
- [Effective Go — Channels](https://go.dev/doc/effective_go#channels)

## Next Steps

- [concurrency/scatter-gather](../scatter-gather/) — one goroutine per task instead of a fixed pool
- [concurrency/fan-out-timeout](../fan-out-timeout/) — add a per-worker deadline with `time.After`
- [pool](../../pool/) — a generic worker pool with per-task error tracking
