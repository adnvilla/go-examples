# Circuit Breaker

**Category:** design-patterns
**Difficulty:** Advanced

## Objective

Implement the circuit breaker resilience pattern from scratch — no framework — and walk its full lifecycle deterministically: **closed** (calls flow, consecutive failures are counted) → **open** (fail fast with `ErrOpen`, the downstream is left alone to recover) → **half-open** (a limited probe tests recovery; failure re-opens, enough successes close). The demo proves the protective property with a counter: 9 attempts, only 7 reach the downstream.

## Concepts Covered

- The three-state machine and both trip directions: `failureThreshold` consecutive failures open it, `successThreshold` successful probes close it, a failed probe re-opens it *and restarts the cooldown*
- Failing fast as protection for **both** sides: callers stop burning latency/timeouts on a dead dependency, and the dependency stops receiving load while it recovers
- Single-probe half-open: concurrent callers during a probe get `ErrOpen` rather than stampeding a barely-recovered service
- Injected clock (`now func() time.Time`) — the seam that makes the state machine testable without `time.Sleep`
- `OnStateChange` callback for observability (state transitions are exactly what you want on a dashboard)
- Table-stakes concurrency: every decision under one mutex, race-detector clean

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
circuit-breaker/
├── go.mod
├── breaker.go        # the state machine (the reusable part)
├── main.go           # deterministic lifecycle walkthrough
├── breaker_test.go   # fake-clock tests for every transition
├── Makefile
└── README.md
```

## How to Run

```bash
make run    # lifecycle demo
make test   # fake-clock state-machine tests
```

## Expected Output

```
--- closed: three consecutive failures trip the breaker ---
call 1: downstream error (503 service unavailable)
call 2: downstream error (503 service unavailable)
  >> breaker: closed -> open
call 3: downstream error (503 service unavailable)

--- open: calls fail fast, the downstream is left alone ---
call 4: short-circuited (circuit breaker is open)
call 5: short-circuited (circuit breaker is open)
downstream calls during open state: 0

--- half-open probe while still broken: back to open ---
  >> breaker: open -> half-open
  >> breaker: half-open -> open
call 6 (probe): downstream error (503 service unavailable)

--- half-open probes after recovery: two successes close it ---
  >> breaker: open -> half-open
call 7 (probe): ok
  >> breaker: half-open -> closed
call 8 (probe): ok

--- closed again: traffic flows normally ---
call 9: ok
final state: closed, total downstream calls: 7 (of 9 attempts)
```

## Code Walkthrough

- `Breaker.Do` splits the pattern into its two halves: `allow` (may this call proceed?) and `record` (fold the outcome into the state machine). Keeping them separate makes each transition auditable — `allow` owns open→half-open (cooldown elapsed), `record` owns closed→open (failure streak), half-open→open (failed probe), and half-open→closed (probe quota met).
- The failure counter is *consecutive*, and `record` resets it on any success while closed — a 1% error rate never trips the breaker, a hard outage trips it in `failureThreshold` calls. (Production breakers often use a rolling error-rate window instead; consecutive-failures is the simplest policy that has the right shape.)
- Half-open admits **one probe at a time** (`probing` flag): the point of half-open is to gather a signal with minimal risk, and letting every queued caller through at once would re-create the stampede the breaker exists to prevent. A failed probe re-opens *and* resets `openedAt`, so the downstream gets a full fresh cooldown.
- `now` is a field, not a call to `time.Now` — that one line is why `breaker_test.go` can verify cooldown behavior by advancing a `fakeClock` two minutes in zero wall time. The tests cover every edge: trip threshold, streak reset, failed-probe re-open (including the restarted cooldown), close-after-probes, and the exact transition sequence via `OnStateChange`.
- The demo's `flakyService.calls` counter is the proof of the pattern's value: during the open window the downstream receives zero traffic, and across the whole run only 7 of 9 attempts touched it.

## Common Pitfalls

- **Tripping on error *rate* without a minimum volume.** 1 failure out of 1 request is a 100% error rate; naive rate-based breakers flap at low traffic. Consecutive-failure counting (used here) or rate-with-minimum-volume both avoid it.
- **Letting all traffic through in half-open.** Half-open is a *probe*, not a reopening. Admit one (or a small quota) and short-circuit the rest, or recovery kills the dependency again.
- **Not restarting the cooldown on a failed probe.** Otherwise the breaker probes on every call after the first cooldown, which is just the closed state with extra steps.
- **Wrapping the breaker around code that returns business errors.** A "user not found" must not count as downstream failure or normal traffic trips the breaker. Classify errors at the call site: only transport/5xx-class failures feed `record`.
- **One breaker for many dependencies (or one per request).** The breaker's state *is* per-downstream health; share one instance per dependency, not per call site or per request.
- **Forgetting the fallback.** `ErrOpen` is the *start* of the story — the caller still needs a policy: cached data, a default, a queue, or a clean error to the user.

## References

- [Martin Fowler — CircuitBreaker](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Microsoft Azure Architecture — Circuit Breaker pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker)
- [sony/gobreaker](https://github.com/sony/gobreaker) — a production-grade implementation of the same machine

## Next Steps

- [http-client](../http-client/) — retries with backoff; a breaker and a retrier compose (retry *inside*, breaker *outside*)
- [concurrency/rate-limiting](../concurrency/rate-limiting/) — the other classic protection, bounding rate instead of cutting off failure
- [mocking](../mocking/) — the injected-dependency testing style `breaker_test.go` uses
