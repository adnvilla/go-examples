# Graceful Shutdown with OS Signals

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show the standard shape of signal-driven graceful shutdown: `signal.NotifyContext` converts SIGINT/SIGTERM into context cancellation, workers treat cancellation as "finish what you're doing, then stop," and `main` enforces a shutdown deadline so one stuck worker can't hang the process forever. This is the pattern behind every well-behaved service under Kubernetes, systemd, or a terminal Ctrl+C.

## Concepts Covered

- `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` — signals as context cancellation, composable with everything else that takes a `context.Context`
- Why `defer stop()` matters: it restores default signal handling, so a *second* Ctrl+C force-kills a shutdown that is itself stuck
- `sync.WaitGroup.Go` (Go 1.25) — the one-call replacement for the `Add(1)` / `go` / `defer Done()` pattern
- Draining: cancellation means "stop taking new work," not "drop the job in flight"
- Bounding the wait: racing `wg.Wait()` (via a completion channel) against `time.After` because `WaitGroup` has no timeout of its own

## Prerequisites

- Go 1.25+ (uses `sync.WaitGroup.Go`)
- No external services or environment variables required
- The self-signaling demo (`SIGINT` to own pid) works on Linux/macOS; on Windows, run it and stop it with a real Ctrl+C instead

## Project Structure

```
graceful-shutdown-signals/
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

The program stops on its own: after 300ms it sends SIGINT to itself, standing in for the operator's Ctrl+C. Press Ctrl+C earlier to trigger the same path manually.

## Expected Output

Worker lines can interleave in any order (they're independent goroutines); the `main:` lines always appear in this sequence:

```
main: 3 workers running; press Ctrl+C to stop
worker 2: processing jobs
worker 3: processing jobs
worker 1: processing jobs
main: simulating Ctrl+C (sending SIGINT to self)
main: shutdown signal received; waiting up to 2s for workers
worker 3: shutdown requested, draining current job
worker 2: shutdown requested, draining current job
worker 1: shutdown requested, draining current job
worker 1: stopped
worker 3: stopped
worker 2: stopped
main: all workers stopped cleanly
```

## Code Walkthrough

- `signal.NotifyContext` returns a context that's canceled on the first SIGINT or SIGTERM. Because shutdown arrives as *context cancellation*, everything downstream that already accepts a context — workers here, but equally HTTP servers, database calls, message consumers — participates in shutdown with no extra plumbing.
- The deferred `stop()` is not just cleanup: while the returned context is armed, the signals are *captured* and no longer kill the process. `stop()` re-enables default handling, so if graceful shutdown itself wedges, the operator's second Ctrl+C still works. Forgetting this is how services end up needing `kill -9`.
- Each worker `select`s between `ctx.Done()` and its work source (a ticker standing in for a job queue). On cancellation it finishes the in-flight job (simulated by the 50ms sleep) before returning — cancellation is a request to wind down, not to abort mid-write.
- `wg.Go(func() { worker(ctx, id) })` (Go 1.25) replaces the classic three-line `Add`/`go`/`defer Done()` dance and makes the off-by-one `Add` mistakes impossible.
- `wg.Wait()` blocks forever if a worker never returns, so `run` waits for it in a goroutine that closes `workersDone`, then races that channel against `time.After(shutdownTimeout)`. On timeout the process exits nonzero — deliberately, since a supervisor (Kubernetes, systemd) will escalate to SIGKILL anyway, and exiting with an error is more honest than hanging.
- `signalSelf` exists only so the demo terminates deterministically: it delivers a real SIGINT through `os.FindProcess(os.Getpid()).Signal`, exercising the actual signal path rather than faking cancellation.

## Common Pitfalls

- **Calling `stop()` (or `signal.Reset`) too early.** If you release the signal registration before shutdown finishes, an impatient second Ctrl+C kills the process mid-drain. Deferring `stop()` in `main`/`run` gets the order right: capture → drain → restore.
- **Treating cancellation as "abort now."** Dropping an in-flight job on `ctx.Done()` can lose data (half-written files, unacked messages). Check for cancellation *between* units of work, not in the middle of one — or pass the context into the work itself if it's long-running.
- **Waiting on `wg.Wait()` with no deadline.** One worker blocked on an unresponsive dependency turns "graceful shutdown" into "hangs until SIGKILL." Always bound the wait and exit nonzero on timeout.
- **Ignoring SIGTERM.** Terminals send SIGINT, but orchestrators (Kubernetes, Docker, systemd) send SIGTERM first and SIGKILL after a grace period. Registering only `os.Interrupt` means your service never sees Kubernetes asking nicely.
- **Doing real work in the signal-handling goroutine.** With `NotifyContext` there's no handler goroutine to misuse — another reason to prefer it over a raw `signal.Notify` channel unless you need per-signal behavior (e.g. SIGHUP for config reload).

## References

- [os/signal package docs — NotifyContext](https://pkg.go.dev/os/signal#NotifyContext)
- [sync package docs — WaitGroup.Go](https://pkg.go.dev/sync#WaitGroup.Go)
- [Go Blog — Context and structs](https://go.dev/blog/context-and-structs)

## Next Steps

- [http-server](../http-server/) — the same idea applied to `http.Server.Shutdown` with in-flight requests
- [context](../context/) — the cancellation machinery this pattern is built on
- [concurrency/worker-pool](../concurrency/worker-pool/) — the worker/job-channel structure these workers sketch
