# Context

**Category:** context
**Difficulty:** Intermediate

## Objective

Show the three main things `context.Context` is used for in idiomatic Go: cancelling work early, bounding work with a deadline, and threading request-scoped values through a call chain without changing every function signature.

## Concepts Covered

- `context.WithCancel` — a parent explicitly cancels in-flight work
- `context.WithTimeout` — work races against an automatic deadline
- `context.WithValue` — passing request-scoped data (e.g. a request ID) through nested calls
- `select` on `ctx.Done()` to make a function cancellation-aware
- `errors.Is(err, context.DeadlineExceeded)` to distinguish a timeout from other errors
- Child contexts inherit the parent's deadline/cancellation

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
context/
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

```
cancelExample: cancelled: context canceled
timeoutExample: timeout: context deadline exceeded
valueExample: [req-abc-123] done
```

## Code Walkthrough

- **`cancelExample`** starts `slowOp` (a 500ms simulated operation) in a goroutine, waits 100ms, then calls `cancel()`. `slowOp`'s `select` sees `ctx.Done()` fire before its `time.After(500ms)`, so it returns `ctx.Err()` (`context.Canceled`) instead of completing.
- **`timeoutExample`** is the same shape, but the deadline is set automatically via `context.WithTimeout(ctx, 100*time.Millisecond)` instead of an explicit `cancel()` call — the 500ms `slowOp` still loses the race, this time to `context.DeadlineExceeded`.
- **`slowOp`** is the reusable pattern: any function that does cancellable work should `select` on `ctx.Done()` alongside its actual work, and return `ctx.Err()` when it fires.
- **`valueExample`** attaches a request ID to a `context.Context` with `WithValue`, using an unexported `ctxKey` type as the key — this avoids collisions with keys other packages might store in the same context. `processRequest` retrieves it with a type assertion.
- Inside `processRequest`, a *child* context is derived with its own 2-second timeout — it inherits `req-abc-123` from the parent automatically, showing that values and cancellation both propagate down the context tree.
- `fetchData` finishes in 50ms, well under the 2-second deadline, so `valueExample` prints `done` rather than a deadline-exceeded error.

## Common Pitfalls

- **Storing context values with a plain `string` key.** Two unrelated packages could collide on the same string key; declaring an unexported key type (`type ctxKey string`, as done here) prevents that.
- **Using `context.Value` for anything other than request-scoped metadata.** It should never carry required function parameters — if a value is needed for correctness (not just tracing/logging), pass it explicitly as an argument.
- **Not checking `ctx.Done()` in long-running or blocking work.** A function that ignores its context can't be cancelled or timed out no matter what the caller does — `select`-ing on `ctx.Done()` (as `slowOp` and `fetchData` do) is what makes cancellation actually work.
- **Forgetting `defer cancel()`.** Every `context.With{Cancel,Timeout,Deadline}` returns a `cancel` function that must be called (typically via `defer`) once the context is no longer needed, or its resources (an internal timer, for `WithTimeout`/`WithDeadline`) leak until the parent's own deadline.
- **Comparing errors with `==` instead of `errors.Is`.** `ctx.Err()` returns sentinel errors (`context.Canceled`, `context.DeadlineExceeded`) that could be wrapped by callers — `errors.Is` (used in `processRequest`) unwraps correctly; `==` doesn't.

## References

- [context package docs](https://pkg.go.dev/context)
- [Go Blog — Context](https://go.dev/blog/context)
- [Go Blog — Go Concurrency Patterns: Timing out, moving on](https://go.dev/blog/context)

## Next Steps

- [concurrency/fan-out-timeout](../concurrency/fan-out-timeout/) — the same deadline-racing idea using raw `time.After` instead of `context`
- [errgroup](../errgroup/) — `errgroup.WithContext` cancels sibling goroutines when one fails
- [errors](../errors/) — more on `errors.Is`/`errors.As` and wrapping
