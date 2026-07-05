# HTTP Server

**Category:** http
**Difficulty:** Intermediate

## Objective

Show a production-shaped HTTP server built entirely on the standard library: a middleware chain (logging, panic recovery), JSON handlers, and graceful shutdown on `SIGINT`/`SIGTERM`.

## Concepts Covered

- `http.NewServeMux` with Go 1.22+ method-and-path patterns (`"GET /health"`, `"POST /echo"`)
- Composing middleware as `func(http.Handler) http.Handler`, applied in a defined order via `chain`
- Recovering from a handler panic in middleware instead of letting it crash the process
- `http.Server` timeouts (`ReadTimeout`/`WriteTimeout`/`IdleTimeout`) to bound slow clients
- Graceful shutdown: catching `os/signal`, then `srv.Shutdown(ctx)` with its own deadline so in-flight requests can finish

## Prerequisites

- Go 1.24+
- No external services; listens on `:8080`

## Project Structure

```
http-server/
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

The server blocks until you send `Ctrl+C` (SIGINT) or SIGTERM, then shuts down gracefully. In another terminal:

```bash
curl http://localhost:8080/health
curl -X POST http://localhost:8080/echo -d '{"hello":"world"}'
curl http://localhost:8080/panic
```

## Expected Output

Server log on startup and on each request (timestamps/durations will vary):

```
2026/07/05 13:09:36 INFO server started addr=:8080
2026/07/05 13:09:37 INFO request method=GET path=/health duration=221.375µs
2026/07/05 13:09:37 INFO request method=POST path=/echo duration=108.542µs
2026/07/05 13:09:37 ERROR panic recovered error="intentional panic — recovered by middleware"
2026/07/05 13:09:37 INFO request method=GET path=/panic duration=59.125µs
2026/07/05 13:09:37 INFO shutting down...
```

Client-side responses:

```
$ curl http://localhost:8080/health
{"data":{"status":"ok"}}

$ curl -X POST http://localhost:8080/echo -d '{"hello":"world"}'
{"data":{"hello":"world"}}

$ curl -o /dev/null -w "%{http_code}\n" http://localhost:8080/panic
500
```

## Code Walkthrough

- `chain(h, middlewares...)` wraps a handler with each middleware in order, applying them last-to-first so the *first* middleware listed runs *outermost* — here, `logging` sees the whole request/response cycle including anything `recoverer` catches.
- `logging` times each request and logs method/path/duration after the handler runs.
- `recoverer` wraps every request in a `defer`/`recover`, turning any handler panic into a logged error and a `500` response instead of taking down the whole server — demonstrated by `GET /panic`, which panics intentionally.
- `newRouter` registers three routes using Go 1.22's method-aware mux patterns (`"GET /health"` only matches GET requests to that path, no manual method check needed), then wraps the mux with both middlewares.
- Graceful shutdown: `main` starts the server in a goroutine, blocks on a `quit` channel fed by `signal.Notify`, and — once a signal arrives — calls `srv.Shutdown(ctx)` with a 10-second deadline. `Shutdown` stops accepting new connections and waits for in-flight ones to finish (or the deadline to expire), rather than dropping active requests.

## Common Pitfalls

- **Skipping `recoverer` in the middleware chain.** Without it, an unhandled panic in any handler crashes the entire process, taking down every other in-flight request too.
- **Calling `srv.Close()` instead of `srv.Shutdown(ctx)`.** `Close()` terminates all connections immediately; `Shutdown` lets in-flight requests finish within its deadline — the latter is what "graceful" means here.
- **Forgetting the shutdown deadline.** `Shutdown(ctx)` with no timeout on `ctx` could block indefinitely if a client holds a connection open; a bounded `context.WithTimeout` (10s here) ensures the process still exits.
- **Ordering middleware incorrectly.** Since `chain` applies middlewares in reverse, putting `recoverer` before `logging` in the argument list makes `logging` run *inside* the recovery, meaning a panic inside `logging` itself wouldn't be caught.
- **Not setting server timeouts.** Without `ReadTimeout`/`WriteTimeout`, a slow or malicious client can hold a connection open indefinitely, exhausting server resources.

## References

- [net/http package docs](https://pkg.go.dev/net/http)
- [Go Blog — Routing Enhancements for Go 1.22](https://go.dev/blog/routing-enhancements)
- [Go Blog — Graceful shutdown pattern](https://pkg.go.dev/net/http#Server.Shutdown)

## Next Steps

- [http-client](../http-client/) — the client side, including retries and backoff
- [recover](../recover/) — a closer look at `panic`/`recover` on its own
- [slog](../slog/) — more on structured logging with `log/slog`
