# OpenTelemetry Tracing

**Category:** observability
**Difficulty:** Intermediate

## Objective

Show distributed tracing with OpenTelemetry's Go SDK: assembling a `TracerProvider` (exporter + batcher + resource), creating nested spans whose structure flows through `context.Context`, and enriching them with attributes, events, and recorded errors. Completes the observability triad in this repo — [slog](../slog/) for logs, [metric](../metric/) for metrics, this example for traces.

By default the spans print to stdout, so the example runs with **zero infrastructure**; set one environment variable and the exact same code ships the traces to Jaeger for the real waterfall-view experience.

## Concepts Covered

- `TracerProvider` assembly: `WithBatcher(exporter)` (ship spans off the hot path) + `WithResource` (`service.name` — the identity every backend groups by)
- Swappable exporters behind one interface: `stdouttrace` for local dev, `otlptracehttp` for real backends, selected via the standard `OTEL_EXPORTER_OTLP_ENDPOINT` variable
- `tracer.Start(ctx, ...)` returning a *new context*: passing it down is what makes callee spans children — context flow **is** the trace structure
- Span enrichment: `WithAttributes` (indexed metadata), `AddEvent` (point-in-time annotations), `RecordError` + `SetStatus(codes.Error, ...)` (failures that show up red in the UI and feed error rates)
- `provider.Shutdown` flushing the batch — the step whose omission silently drops every span of a short-lived process

## Prerequisites

- Go 1.24+
- Nothing for the default run (stdout exporter)
- For the Jaeger run: Docker Compose (`docker compose up -d jaeger` or `make infra-up` from this directory)
- Dependencies justified: the `go.opentelemetry.io/otel` SDK and exporters are the topic being taught

## Project Structure

```
otel/
├── go.mod
├── go.sum
├── main.go
├── Makefile
└── README.md
```

## How to Run

```bash
# zero-infra: spans print to stdout
make run

# with Jaeger: start it, export, then explore the trace in the UI
make infra-up
make run-jaeger
open http://localhost:16686   # pick service "checkout-demo", Find Traces
```

## Expected Output

The demo lines are fixed; the span JSON that follows varies in IDs and timestamps (abridged here — 6 spans across 2 traces are printed):

```
exporting spans via: stdout (set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 for Jaeger)
checkout order-1001: completed
checkout order-1002: failed: charging card: card declined (failure recorded on the trace)
{
	"Name": "reserve-inventory",
	"SpanContext": { "TraceID": "…", "SpanID": "…", … },
	"Parent":      { "TraceID": "…", "SpanID": "…", … },
	"Attributes": [ { "Key": "items.count", "Value": { "Type": "INT64", "Value": 2 } } ],
	…
}
…
{
	"Name": "charge-card",
	"Events": [ { "Name": "exception", "Attributes": [ …"card declined"… ] } ],
	"Status": { "Code": "Error", "Description": "card declined" },
	…
}
```

In Jaeger (`make run-jaeger`), the same data appears as two traces for service `checkout-demo`: `checkout` → `reserve-inventory` + `charge-card`, with order-1002's trace flagged as an error.

## Code Walkthrough

- `newExporter` is the only environment-dependent code: with `OTEL_EXPORTER_OTLP_ENDPOINT` set it builds an OTLP/HTTP exporter (the env var is part of the OTel spec, so the exporter reads it on its own); otherwise `stdouttrace` with pretty-printing. Everything downstream — provider, tracer, spans — is identical either way, which is the SDK's core promise: instrumentation code never knows where telemetry goes.
- The `TracerProvider` gets a `WithBatcher` (spans are buffered and exported in the background, keeping `span.End()` cheap) and a `Resource` carrying `semconv.ServiceName("checkout-demo")` — resource attributes describe the *process* and are stamped on every span, which is how backends group traces by service.
- The deferred `provider.Shutdown` is load-bearing: the batcher holds spans in memory, and a short-lived program that exits without flushing exports nothing. `context.WithoutCancel` keeps the flush working even if the main context already expired; its error is joined into `run`'s return value rather than swallowed.
- `checkout` shows the propagation rule: `tracer.Start(ctx, "checkout")` returns a **new** `ctx` carrying the span, and `reserveInventory(ctx, …)`/`chargeCard(ctx, …)` start their spans from it — that's all it takes for the backend to render them as children. No IDs are threaded by hand.
- The declined-card path records failure twice, deliberately: `RecordError(err)` attaches the error as a structured `exception` event (type + message, visible in the JSON output above), and `SetStatus(codes.Error, …)` marks the span itself failed — the status is what error-rate dashboards and red trace listings key on. Returning the error still happens; tracing annotates, it doesn't replace error handling.

## Common Pitfalls

- **Exiting without `Shutdown` (or `ForceFlush`).** The batcher's whole point is not exporting synchronously — so buffered spans die with the process. In `main`-style programs, `defer provider.Shutdown(ctx)` immediately after construction.
- **Ignoring the context returned by `tracer.Start`.** `_, span := tracer.Start(ctx, …)` inside a helper silently orphans every downstream span into its own trace. Take both return values and pass the new context on (the leaf functions here use `_` only because nothing is called below them).
- **`RecordError` without `SetStatus`.** The event alone doesn't mark the span failed — backends still show it green. Pair them (and note the SDK never sets `Ok` automatically; unset and ok are distinct states).
- **High-cardinality span names.** `checkout order-1002` as a *name* explodes every backend's grouping; the order ID belongs in an attribute (as done here), names stay low-cardinality (`checkout`).
- **Confusing events with logs.** `AddEvent` annotations live and die with the span and its sampling decision. They're for trace-local context, not a replacement for `slog`.

## References

- [OpenTelemetry Go — Getting Started](https://opentelemetry.io/docs/languages/go/getting-started/)
- [go.opentelemetry.io/otel/sdk/trace package docs](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace)
- [OTLP Exporter configuration (env vars)](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/)
- [Jaeger — Getting Started](https://www.jaegertracing.io/docs/latest/getting-started/)

## Next Steps

- [slog](../slog/) — structured logging, the first leg of the triad
- [metric](../metric/) — StatsD metrics, the second leg
- [http-server](../http-server/) — a natural host for real instrumentation (`otelhttp` middleware wraps handlers the same way its middleware chain does)
