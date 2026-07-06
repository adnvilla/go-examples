# Pipeline

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show the pipeline pattern from the Go blog: independent stages connected by channels, where an upstream `close` propagates termination downstream through `range`, `merge` fans several streams into one, and context cancellation lets a consumer walk away early without leaking producer goroutines. Every stage goroutine is registered in a `sync.WaitGroup`, so the example *proves* the no-leak property instead of asserting it in a comment.

## Concepts Covered

- Stage shape: own your output channel, `defer close(out)`, `range` your input — close propagation is what terminates the whole chain
- `select { case out <- v: case <-ctx.Done(): }` on **every send** — the modern form of the Go blog's `done` channel, and the single detail that makes early cancellation leak-free
- Fan-in (`merge`): one forwarder per input plus a closer goroutine that waits for all forwarders — and why fan-in gives up ordering
- Work sharing: two `square` stages ranging over the *same* input channel split the values between them
- `sync.WaitGroup.Go` (Go 1.25) tracking every stage goroutine so `wg.Wait()` verifies clean shutdown

## Prerequisites

- Go 1.25+ (uses `sync.WaitGroup.Go`)
- No external services or environment variables required

## Project Structure

```
pipeline/
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

## Expected Output

```
--- linear pipeline: gen -> square ---
1 4 9 16 25

--- fan-in: two square stages share the work, merged ---
1 4 9 16 25 36 49 64 81 100 (sorted; arrival order varies)

--- early cancellation: take 3 of 1000, then cancel ---
took: 1 4 9
all stage goroutines exited — no leaks
```

## Code Walkthrough

- `gen` and `square` have the canonical stage shape: they *own* the channel they return, close it on the way out (`defer close(out)`), and guard every send with a `select` on `ctx.Done()`. Ownership is the discipline that makes `close` safe — only the sender closes, exactly once.
- In `linearPipeline`, nothing but `range` is needed on the consuming side: when `gen` runs out of numbers it closes its channel, `square`'s `range` ends, `square` closes its channel, the consumer's `range` ends. Termination is data-driven; no coordination code.
- `fanInPipeline` starts **two** `square` stages reading the same `gen` channel — each value is received by exactly one of them, so the work is split. `merge` then funnels both outputs into one channel: a forwarder goroutine per input, plus a closer that waits on the forwarders' own WaitGroup before `close(out)`. Without that closer the consumer's `range` would block forever; with it, but closing too early, values would be dropped. The output arrives in whatever order the two stages produce it — the example sorts before printing, a real consumer must either not care or reattach sequence numbers.
- `earlyCancellation` is why the `ctx` plumbing exists. The consumer takes 3 values from a 1000-value pipeline and `break`s. At that moment `square` is blocked sending value #4 and `gen` is blocked sending a later number — with a bare `out <- v` they would block forever, invisible until the process's goroutine count climbs. `cancel()` makes every blocked send's `select` take the `ctx.Done()` arm, and `wg.Wait()` returning is the proof that all four goroutines exited.
- The WaitGroup threading (`wg.Go` inside each stage constructor) isn't part of the classic pattern — it's here to make goroutine lifetimes observable. In production the same role is played by `errgroup.Group` (see [errgroup](../../errgroup/)), which adds error propagation.

## Common Pitfalls

- **A send without a `select` on cancellation.** `out <- n` in a stage is correct *only* if the consumer is guaranteed to drain the channel. Any consumer that can stop early (errors, limits, timeouts) turns that send into a permanent goroutine leak.
- **Closing a channel you don't own, or from the receiving side.** Close is a sender-side, owner-only operation; double-close or send-on-closed panics come from breaking this rule. Each stage closing only its own output is what keeps the property local and auditable.
- **Forgetting `merge`'s closer goroutine.** If nobody closes the merged channel, the consumer's `range` never terminates — the pipeline "works" but the program hangs at the end.
- **Assuming fan-in preserves order.** It doesn't, by design. If order matters, don't fan out (keep one stage), or tag values with an index and reorder at the sink.
- **Unbounded buffering as a substitute for cancellation.** Buffered channels can mask the leak (sends succeed into the buffer), but the goroutines and memory still linger; buffers size throughput, they don't manage lifetimes.

## References

- [Go Blog — Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
- [Go Blog — Share Memory By Communicating](https://go.dev/blog/codelab-share)
- [sync package docs — WaitGroup.Go](https://pkg.go.dev/sync#WaitGroup.Go)

## Next Steps

- [errgroup](../../errgroup/) — the production version of the WaitGroup threading here, with error propagation
- [concurrency/worker-pool](../worker-pool/) — a fixed pool consuming one job channel, the sibling pattern
- [concurrency/fan-out-timeout](../fan-out-timeout/) — per-worker deadlines instead of whole-pipeline cancellation
