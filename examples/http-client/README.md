# HTTP Client

**Category:** http
**Difficulty:** Intermediate

## Objective

Show a production-shaped HTTP client built entirely on the standard library: per-request timeout, exponential-backoff retries limited to retryable failures, `context` cancellation honored mid-retry, and JSON decoding straight from the response body.

## Concepts Covered

- `http.Client{Timeout: ...}` for an overall per-request timeout
- `http.NewRequestWithContext` so the request is cancellable via `context`
- Distinguishing retryable errors (network failures, 5xx responses) from non-retryable ones (4xx responses) with a custom `*retryableError` type and `errors.As`
- Exponential backoff (`baseDelay * 2^(attempt-1)`) between retries, itself cancellable via `select` on `ctx.Done()`
- Streaming JSON decode of the response body with `json.NewDecoder(resp.Body).Decode(...)`
- `log/slog` for structured retry/error logging

## Prerequisites

- Go 1.25+
- **Internet access** — this example makes a real GET request to `https://jsonplaceholder.typicode.com/posts/1`, a public test API. No other setup required.

## Project Structure

```
http-client/
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

The response body from the public test API is stable, so this should be reproducible as long as the service is reachable and unchanged:

```
Post #1: sunt aut facere repellat provident occaecati excepturi optio reprehenderit
```

## Code Walkthrough

- `NewClient` wraps a standard `*http.Client` (with an overall timeout) alongside retry configuration (`maxRetries`, `baseDelay`).
- `GetJSON` is the retry loop: it calls `doGet` up to `maxRetries + 1` times, sleeping an exponentially growing delay between attempts (the *n*th retry waits `baseDelay * 2^(n-1)`), and logs each retry via `slog.Info`.
- The backoff sleep itself is a `select` between `time.After(delay)` and `ctx.Done()` — if the caller's context is cancelled while waiting to retry, the function returns immediately instead of sleeping out the full delay.
- `doGet` classifies failures: a transport-level error or a 5xx response becomes a `*retryableError` (which `GetJSON` catches via `errors.As` and loops again); a 4xx response is returned as a plain error and propagates immediately without retrying, since retrying a client error won't help.
- On success, the response body is decoded directly into the caller's destination (`dst any`) with `json.NewDecoder(...).Decode(dst)`, avoiding an intermediate `[]byte` buffer.

## Common Pitfalls

- **Retrying 4xx errors.** A malformed request or bad auth won't succeed on retry — only network failures and 5xx (server-side, possibly transient) errors are retried here.
- **Not closing the response body.** `defer resp.Body.Close()` is required on every successful `c.http.Do(req)` call, or the connection can't be reused (or, worse, its resources leak).
- **Retrying without a cap or backoff.** Retrying immediately in a tight loop can amplify load on a struggling server; exponential backoff spaces retries out as failures continue.
- **Ignoring the request context during backoff.** Sleeping with a plain `time.Sleep(delay)` between retries would ignore `ctx` cancellation entirely — the `select` here is what makes the whole retry loop respect the caller's deadline.
- **Building the request without `NewRequestWithContext`.** A request built with plain `http.NewRequest` can't be cancelled via context at all.

## References

- [net/http package docs](https://pkg.go.dev/net/http)
- [Go Blog — Go Concurrency Patterns: Timing out, moving on](https://go.dev/blog/context)
- [Google SRE Book — Handling Overload (retry/backoff rationale)](https://sre.google/sre-book/handling-overload/)

## Next Steps

- [http-server](../http-server/) — the server side, including graceful shutdown
- [context](../context/) — a deeper look at cancellation and deadlines on their own
- [errors](../errors/) — more on the `errors.As`/custom error type pattern used for `*retryableError`
