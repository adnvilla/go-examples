# Rate Limiting

**Category:** concurrency
**Difficulty:** Intermediate

## Objective

Show the two standard ways to rate-limit in Go and when each fits: a `time.Ticker` as a rigid fixed-cadence limiter, and `golang.org/x/time/rate`'s token bucket — `Allow` for load-shedding decisions (answer 429 now) and `Wait` for pacing work under a quota (delay, don't drop). The demo verifies its own pacing guarantee instead of printing machine-dependent timings.

## Concepts Covered

- `time.Ticker` as a one-request-per-tick limiter, and why it's the weakest option (no bursts, fixed slot size)
- The token-bucket model: refill rate (`rate.Limit`) vs bucket size (`burst`), and how burst absorbs spikes while the rate bounds the steady state
- `Limiter.Allow` — non-blocking take-or-reject, the server-side load-shedding shape
- `Limiter.Wait(ctx)` — blocking until a token is available, the client-side quota-respecting shape, with context cancellation cutting the wait short
- Verifying timing behavior structurally (elapsed ≥ guaranteed minimum) rather than printing raw durations

## Prerequisites

- Go 1.25+
- No external services or environment variables required — the single dependency (`golang.org/x/time`) is the canonical rate-limiting package and the thing being taught

## Project Structure

```
rate-limiting/
├── go.mod
├── go.sum
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
--- time.Ticker: fixed cadence, one request per tick ---
request 1 processed on its tick
request 2 processed on its tick
request 3 processed on its tick
request 4 processed on its tick

--- rate.Limiter Allow: refill 10/s, burst 3 — shed what overflows ---
request 1: allowed (token consumed)
request 2: allowed (token consumed)
request 3: allowed (token consumed)
request 4: rejected — bucket empty, a real server would answer 429
request 5: rejected — bucket empty, a real server would answer 429

--- rate.Limiter Wait: refill 50/s, burst 1 — pace instead of reject ---
request 1 sent
request 2 sent
request 3 sent
request 4 sent
request 5 sent
5 requests paced over at least 80ms — nothing dropped, nothing early
```

## Code Walkthrough

- `tickerCadence` blocks on `<-ticker.C` before each request: one slot every 20ms, full stop. It's fine for polling loops, but as a limiter it can't absorb bursts (a request arriving just after a tick waits the full interval) and it keeps firing whether or not anyone is consuming.
- `tokenBucketAllow` builds `rate.NewLimiter(rate.Limit(10), 3)`: tokens drip in at 10/s and the bucket holds at most 3. Five back-to-back `Allow` calls consume the 3 accumulated tokens and reject the other two — deterministically, because at 10/s no meaningful refill happens between consecutive calls. This is the server-side shape: decide *now*, shed what overflows, let the client retry.
- `tokenBucketWait` flips the policy from rejecting to pacing: `Wait(ctx)` sleeps until a token is available. With refill 50/s and burst 1, the first request goes immediately and each of the next four waits ~20ms, so five requests cannot legally complete in under 80ms. The code asserts that lower bound and prints the *guarantee*, not the raw elapsed time — keeping the output deterministic while still verifying the limiter did its job.
- `Wait`'s context parameter is the important signature detail: a canceled context or expired deadline aborts the sleep with an error, so a shutting-down service doesn't strand goroutines queueing for tokens.

## Common Pitfalls

- **Confusing rate with burst.** `rate.Limit(10)` bounds the long-run average; `burst` is instantaneous capacity. Burst 1 makes traffic perfectly smooth but brutal on spikes; a larger burst forgives clumps of requests while keeping the same average. Most misconfigured limiters got only one of the two right.
- **Using `Wait` where you needed `Allow` (or vice versa).** `Wait` inside a request handler queues work invisibly under overload — latency balloons and nothing says why. Servers usually want `Allow` + 429; clients respecting someone else's quota want `Wait`.
- **One global limiter when the quota is per-key.** A per-user or per-API-key quota needs a limiter per key (a map guarded by a mutex, with eviction); one shared bucket lets a single noisy client starve everyone.
- **Forgetting `ticker.Stop()`.** A leaked ticker keeps its goroutine and channel alive for the life of the process. (`defer ticker.Stop()` — same discipline as closing a body.)
- **Testing limiters by asserting exact durations.** Wall-clock assertions flake under load. Assert structural guarantees (at least N tokens' worth of time passed; no more than burst succeeded instantly), or inject a fake clock.

## References

- [golang.org/x/time/rate package docs](https://pkg.go.dev/golang.org/x/time/rate)
- [Go Wiki — Rate Limiting](https://go.dev/wiki/RateLimiting)
- [time package docs — Ticker](https://pkg.go.dev/time#Ticker)

## Next Steps

- [concurrency/worker-pool](../worker-pool/) — bounding *concurrency* instead of *rate*; the two compose
- [http-client](../http-client/) — retries with backoff, the client-side companion to respecting quotas
- [context](../context/) — the cancellation semantics `Wait` builds on
