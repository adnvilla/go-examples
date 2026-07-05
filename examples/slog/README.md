# slog

**Category:** observability
**Difficulty:** Beginner

## Objective

Show structured logging with the standard library's `log/slog` (Go 1.21+): key-value fields instead of formatted strings, grouped fields, request-scoped fields propagated through `context.Context`, and swapping the output format (text vs. JSON) without changing any call site.

## Concepts Covered

- `slog.Info`/`Debug`/`Warn`/`Error` with structured key-value pairs instead of `fmt.Sprintf`-style messages
- `slog.SetDefault` + `slog.NewTextHandler`/`slog.NewJSONHandler` to change output format globally
- `slog.HandlerOptions{Level: ...}` to control the minimum logged level (here, `Debug` is visible with the text handler but the JSON handler is set to `Info`, hiding debug-level messages)
- `slog.Group` to namespace related fields (`db.host`, `db.port`, `db.name`) and avoid key collisions
- Threading a request-scoped field (`request_id`) through `context.Context` so a child logger (`logger(ctx)`) includes it automatically

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
slog/
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

Timestamps and `duration_ms` will differ between runs. The text handler writes to **stderr**, the JSON handler (set partway through `main`) writes to **stdout** — shown here separately:

stdout:
```
{"time":"2026-07-05T13:55:44.108692-05:00","level":"INFO","msg":"switched to JSON handler","env":"production"}
```

stderr:
```
time=2026-07-05T13:55:44.097-05:00 level=INFO msg="application started" version=1.0.0
time=2026-07-05T13:55:44.097-05:00 level=DEBUG msg="debug message — only visible at Debug level"
time=2026-07-05T13:55:44.097-05:00 level=WARN msg="high memory usage" used_mb=512 limit_mb=1024 pct=50
time=2026-07-05T13:55:44.097-05:00 level=INFO msg="database connected" db.host=localhost db.port=5432 db.name=orders
time=2026-07-05T13:55:44.097-05:00 level=INFO msg="processing order" request_id=req-abc-123 order_id=42
time=2026-07-05T13:55:44.108-05:00 level=INFO msg="order processed" request_id=req-abc-123 order_id=42 duration_ms=11
```

## Code Walkthrough

- `main` first installs a `TextHandler` writing to `os.Stderr` with `Level: slog.LevelDebug`, so every level including `Debug` is emitted, in a human-readable `key=value` format.
- Every `slog.Info`/`Warn`/etc. call takes alternating key/value arguments after the message — these become structured fields, not part of the message string, so a log aggregator can filter/query on `used_mb` or `pct` directly instead of parsing text.
- `slog.Group("db", "host", ..., "port", ..., "name", ...)` nests three fields under a `db.` prefix in the output, keeping them visually and semantically grouped.
- `withRequestID`/`logger` show the context-propagation pattern: a request ID is stored in `ctx` once, and `logger(ctx)` builds a child logger (`l.With("request_id", id)`) that includes it on every subsequent log line — this is how request-scoped fields (trace IDs, user IDs, etc.) get attached automatically without threading them through every function signature.
- Partway through, `slog.SetDefault` is called again with a `JSONHandler` writing to `os.Stdout` and `Level: slog.LevelInfo` — the same `slog.Info(...)` call site now produces a JSON line instead of text, and would no longer show `Debug`-level messages.

## Common Pitfalls

- **Interpolating values into the message string instead of passing them as fields.** `slog.Info("processing order 42")` loses structure; `slog.Info("processing order", "order_id", 42)` keeps `order_id` queryable as its own field.
- **Mismatched key/value pairs.** `slog` arguments after the message must alternate key, value, key, value — an odd number of arguments (or a non-string key) produces a `!BADKEY` field in the output instead of failing to compile, since these are `...any` arguments.
- **Forgetting a handler's level filters independently of the logger.** Switching handlers (as this example does, text → JSON) can also silently change which levels are emitted if the new handler's `Level` option differs.
- **Building a new logger with `.With(...)` on every call instead of once per request/component.** `logger(ctx)` here constructs a child logger once and reuses it for both log lines in `processOrder`, which is both clearer and avoids redundant work.

## References

- [log/slog package docs](https://pkg.go.dev/log/slog)
- [Go Blog — Structured Logging with slog](https://go.dev/blog/slog)

## Next Steps

- [context](../context/) — more on propagating request-scoped values through `context.Context`
- [http-server](../http-server/) — logging middleware in a real server, using the same structured-field approach
