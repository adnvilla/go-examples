# Pool

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show a small, reusable worker-pool package with per-task error tracking: run a fixed number of `func() error` tasks at a configured concurrency, and check afterward which (if any) failed — without any task's error aborting the others.

## Concepts Covered

- A `Task` wrapping a `func() error`, recording its own result (`Err`) after running
- A `Pool` distributing `*Task` values to a fixed number of worker goroutines over a channel
- `sync.WaitGroup` to know when every task has finished, regardless of success/failure
- `HasErrors()` / per-task `Err` fields as a way to inspect partial failure after a batch completes, instead of stopping at the first error (contrast with [errgroup](../errgroup/), which stops early on first failure)

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
pool/
├── go.mod
├── pool.go
├── pool_test.go
└── README.md
```

## How to Run

This is a library package (`package pool`), not a `main` package — there's nothing to `go run`. Its tests are the runnable demonstration:

```bash
make run
# or
go test -v ./...
```

## Expected Output

```
=== RUN   TestEmptyPool
2026/07/05 16:42:57 Running 0 task(s) at concurrency 10.
--- PASS: TestEmptyPool (0.00s)
=== RUN   TestWithWork
2026/07/05 16:42:57 Running 3 task(s) at concurrency 10.
--- PASS: TestWithWork (0.00s)
=== RUN   TestWithError
2026/07/05 16:42:57 Running 3 task(s) at concurrency 10.
--- PASS: TestWithError (0.00s)
PASS
```

## Code Walkthrough

- `Task` wraps a `func() error` (`f`) and an `Err` field that's populated once `Run` executes it — `Err` is meaningless until the pool has actually run the task.
- `Pool.Run` starts `concurrency` worker goroutines (each running `p.work()`, which loops `for task := range p.tasksChan`), then feeds every task into `tasksChan`, closes it once all are sent, and blocks on `wg.Wait()`.
- Each worker's `task.Run(&p.wg)` executes the task's function, stores the result in `Err`, and calls `wg.Done()` — a failing task doesn't stop other tasks or workers; it just records its own error and moves on.
- `HasErrors()` scans every task's `Err` field after `Run()` returns, so a caller can check for *any* failure across the whole batch, and (since `Tasks` and each `Task.Err` remain accessible) inspect exactly which ones failed.
- `TestEmptyPool` confirms a pool with zero tasks runs cleanly; `TestWithWork` confirms all-success; `TestWithError` confirms one failing task among three doesn't prevent the others from completing, and is correctly reported by both `HasErrors()` and the per-task `Err`.

## Common Pitfalls

- **Expecting `Run()` to stop early on the first error**, like [errgroup](../errgroup/) does. This pool always runs every task to completion — if fail-fast behavior is needed instead, `errgroup.Group` is the better fit.
- **Reading a `Task.Err` before `Pool.Run()` has completed.** `Err` is only meaningful after `Run()` returns; reading it earlier just sees a stale zero value.
- **Setting `concurrency` higher than the number of tasks provides no benefit** — extra workers just block forever on the (now-closed) `tasksChan` once all tasks are consumed, which is harmless but wasteful.
- **Reusing a `Pool` for a second `Run()` call.** `tasksChan` is closed at the end of the first `Run()`; a fresh `Pool` (via `NewPool`) is needed for another batch.

## References

- [sync package docs — WaitGroup](https://pkg.go.dev/sync#WaitGroup)
- [Go by Example — Worker Pools](https://gobyexample.com/worker-pools)

## Next Steps

- [concurrency/worker-pool](../concurrency/worker-pool/) — the same fixed-pool shape as a runnable `main` example
- [errgroup](../errgroup/) — the fail-fast alternative, stopping on the first error instead of collecting all of them
