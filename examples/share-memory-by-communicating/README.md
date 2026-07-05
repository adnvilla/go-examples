# Share Memory By Communicating

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

The classic example from the [Go Blog post of the same name](https://go.dev/blog/codelab-share): a URL poller where state (each URL's last-known status) is owned by a single goroutine and only ever mutated by messages sent to it over a channel — no mutex, no shared map accessed from multiple goroutines.

## Concepts Covered

- "Don't communicate by sharing memory; share memory by communicating" (the Go proverb this example is named after)
- A single owner goroutine (`StateMonitor`) that exclusively holds a `map[string]string`, updated only via messages on a channel
- A `select` inside that goroutine multiplexing two channels: incoming state updates and a periodic `time.Ticker`
- A pipeline of worker goroutines (`Poller`) reading from one channel (`pending`) and writing to two others (`complete`, `status`)
- Self-adjusting backoff: each `Resource` sleeps longer after consecutive errors (`errTimeout * errCount`)

## Prerequisites

- Go 1.24+
- **Internet access** — polls `google.com`, `golang.org`, and `blog.golang.org` over real HTTP
- Runs forever until interrupted; there is no built-in exit condition

## Project Structure

```
share-memory-by-communicating/
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

Stop it with `Ctrl+C` — it never exits on its own.

## Expected Output

A status line every 10 seconds; the poll interval itself is 60 seconds, so most of the time the status is unchanged from before. Line order within each block varies (`StateMonitor`'s map is unordered):

```
2026/07/05 13:48:48 Current state:
2026/07/05 13:48:48  http://www.google.com/ 200 OK
2026/07/05 13:48:48  http://blog.golang.org/ 200 OK
2026/07/05 13:48:48  http://golang.org/ 200 OK
2026/07/05 13:48:58 Current state:
2026/07/05 13:48:58  http://blog.golang.org/ 200 OK
2026/07/05 13:48:58  http://golang.org/ 200 OK
2026/07/05 13:48:58  http://www.google.com/ 200 OK
```

## Code Walkthrough

- `StateMonitor` is the only goroutine that ever touches `urlStatus` (a plain `map[string]string`, not a `sync.Map`) — it owns that memory exclusively, and every other goroutine can only affect it by sending a `State` value on the `updates` channel it returns.
- Inside `StateMonitor`'s goroutine, a `select` between `ticker.C` (print the current state) and `updates` (record a new state) means the map is read and written from exactly one place, with no lock needed — the alternative to a mutex-protected shared map.
- `Poller` is a worker: it reads a `*Resource` from `in`, polls it (`r.Poll()`), reports the result to `status`, then forwards the same `*Resource` to `out` — reusing the object rather than allocating a new one per poll.
- `Resource.Sleep` waits `pollInterval` plus an error-dependent backoff (`errTimeout * errCount`), then sends itself back into `pending` — this is what turns a one-shot poll into a continuous polling loop, and what makes a resource poll less frequently the more consecutive errors it's had.
- `main` wires it together: `numPollers` `Poller` goroutines share the `pending`/`complete` channels; a separate goroutine seeds `pending` with the initial URLs; and the main goroutine's `for r := range complete` loop is what re-queues each resource (via `go r.Sleep(pending)`) after every poll completes.

## Common Pitfalls

- **Reaching for a mutex-guarded shared map instead of a single owner goroutine.** Both work, but this pattern (channels as the only way to touch shared state) is what "share memory by communicating" specifically refers to, and it tends to make ownership boundaries more explicit.
- **Expecting deterministic output ordering.** Both the map iteration order in `logState` and the arrival order of concurrent `Poller` results are non-deterministic.
- **Running this expecting it to terminate.** It's an infinite polling loop by design (like a real monitoring daemon) — there's no exit condition; interrupt it manually.
- **The polled URLs being unreachable.** Since this hits real public endpoints, a network-restricted environment will show `Poll` returning connection errors (which correctly triggers the backoff logic) rather than `200 OK`.

## References

- [Go Blog — Share Memory By Communicating](https://go.dev/blog/codelab-share)
- [Effective Go — Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Go Proverbs](https://go-proverbs.github.io/)

## Next Steps

- [concurrency/worker-pool](../concurrency/worker-pool/) — the same worker/channel shape without the polling/backoff layer
- [sync-primitives](../sync-primitives/) — the mutex-based alternative this pattern avoids
