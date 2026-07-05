# errgroup

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show `golang.org/x/sync/errgroup` replacing the manual `sync.WaitGroup` + shared error-variable/channel pattern for fanning out goroutines that can fail — including automatic context cancellation on first error and an optional concurrency cap.

## Concepts Covered

- `errgroup.WithContext(ctx)` — returns a `*Group` and a derived `context.Context` that's cancelled as soon as any goroutine in the group returns a non-nil error
- `g.Go(func() error {...})` — launch a goroutine whose returned error is captured by the group
- `g.Wait()` — blocks until all goroutines finish, returning the *first* non-nil error (if any)
- `g.SetLimit(n)` — cap how many `g.Go` functions run concurrently, useful against rate-limited downstream services

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
errgroup/
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

Each `fetch` call's delay is proportional to its `id` (`id*10ms`), and results are written into a pre-sized slice by index (not appended), so output order is always index order regardless of completion order:

```
--- unbounded concurrency ---
  1: data-1
  2: data-2
  3: data-3
  4: data-4
  5: data-5
--- limited to 2 concurrent ---
  1: data-1
  2: data-2
  3: data-3
  4: data-4
  5: data-5
```

## Code Walkthrough

- `fetchAll` allocates a `results` slice up front, sized to `len(ids)`, then starts one `g.Go` goroutine per ID — each writes to `results[i]` (its own index), which is why output order is deterministic even though goroutines complete in varying order.
- `errgroup.WithContext(ctx)` returns a context that's cancelled the moment any `g.Go` function returns an error — every `fetch` call is written to respect that via `select` on `ctx.Done()`, so a single failure stops the *other* in-flight fetches from continuing needlessly.
- `fetchWithLimit` is identical except for `g.SetLimit(concurrency)`, called right after creating the group — subsequent `g.Go` calls block until a "slot" frees up once the limit is reached, capping concurrent in-flight fetches to `concurrency` at a time.
- `g.Wait()` returns the *first* error encountered (not all of them) — if multiple goroutines fail, only one error propagates to the caller.

## Common Pitfalls

- **Forgetting to capture loop variables before Go 1.22.** `i, id := i, id` inside the loop is the classic guard against every goroutine closing over the same shared loop variable — as of Go 1.22 the language changed loop variable semantics to make each iteration's variable distinct, but this repo also runs on Go 1.24/1.25 where it's no longer strictly necessary; it's kept here for clarity and because the pattern is still idiomatic when targeting older Go versions.
- **Ignoring the context `errgroup.WithContext` returns.** Using the *original* `ctx` instead of the one returned by `WithContext` means goroutines never see the early-cancellation-on-error behavior — the whole point of using `errgroup` over a plain `WaitGroup` for this use case.
- **Assuming `g.Wait()` reports every failure.** Only the first error is returned; if you need all errors, collect them explicitly (e.g. via `errors.Join` — see [errors](../errors/)) inside each `g.Go` closure instead of relying on the group's return value.
- **Setting `SetLimit` too low for the workload.** A limit lower than the number of goroutines that must run concurrently to make progress (e.g. if later tasks depend on earlier ones completing within the same batch) can deadlock — `SetLimit` is for bounding *independent* work, not for scheduling dependent tasks.

## References

- [golang.org/x/sync/errgroup package docs](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [Go Blog — Error Handling and Go](https://go.dev/blog/error-handling-and-go)

## Next Steps

- [concurrency/scatter-gather](../concurrency/scatter-gather/) — the same fan-out shape using a raw `sync.WaitGroup`, without structured error handling
- [context](../context/) — a deeper look at `context.Context` cancellation and deadlines on their own
