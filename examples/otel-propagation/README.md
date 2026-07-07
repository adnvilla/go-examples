# OpenTelemetry Propagation

**Category:** observability
**Difficulty:** Advanced

## Objective

Show the mechanism that makes tracing *distributed*: two HTTP services in one process, where the caller **Injects** the W3C `traceparent` header into the outgoing request and the callee **Extracts** it, so spans created in different services join a single trace. The [otel](../otel/) example covers spans within a process; this one covers the wire. The demo prints the actual header that crosses between the services and verifies — with the in-memory exporter — that both spans share one trace ID and chain parent to child across the process boundary.

## Concepts Covered

- `propagation.TraceContext{}` — the W3C standard codec: one `traceparent` header (`version-traceid-spanid-flags`)
- `otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))` — one line on the client side, and the entire "distributed" in distributed tracing
- `Extract(r.Context(), HeaderCarrier(r.Header))` on the server side — producing a context whose parent span is *remote* (`Parent.IsRemote() == true`)
- Root vs continued traces: the checkout span has no incoming traceparent (fresh trace); the inventory span joins it
- `tracetest.InMemoryExporter` + `WithSyncer` — inspecting finished spans programmatically, the same tool the tests use to pin the propagation invariants
- Manual propagation on purpose: middleware (`otelhttp`) does these two calls for you in production, and after this example you know exactly what it's doing

## Prerequisites

- Go 1.25+
- No external services or environment variables required — the otel SDK is the topic being taught

## Project Structure

```
otel-propagation/
├── go.mod
├── main.go        # checkout service (Inject) + inventory service (Extract) + span summary
├── main_test.go   # one-trace invariant, remote parent, wire-format header
├── Makefile
└── README.md
```

## How to Run

```bash
make run
make test
```

## Expected Output

Trace and span IDs are random per run; the structure is fixed:

```
--- two services, one trace ---
checkout: calling inventory with traceparent=00-<32 hex>-<16 hex>-01
inventory: extracted remote context (trace <8 hex>...) — joining the caller's trace
client got: reserved

--- exported spans ---
reserve-inventory  trace=<8 hex>... parent=span <8 hex>... (remote=true)
checkout           trace=<8 hex>... parent=none (root)
all spans share one trace id: true
```

## Code Walkthrough

- `checkoutHandler` (service A) starts a server span from `r.Context()` — since the demo's client sends no `traceparent`, this span is the trace **root**. The load-bearing line is the `Inject` before the outbound call: it serializes the current span context into the request headers. The demo prints that header, and it's worth reading once in full: `00-<trace id>-<span id>-01` — version, the trace every downstream span will join, the caller's span (the future parent), and the sampled flag.
- `handleInventory` (service B) mirrors it: `Extract` deserializes the header into a context carrying a **remote** span context, and the span started from that context becomes a child of the caller's span *in another service*. Comment out the Extract and the demo still "works" — but produces two unrelated traces, which is exactly the failure mode teams hit when one service in a chain lacks instrumentation.
- `printSpanSummary` closes the loop with evidence rather than prose: both exported spans carry the same trace ID, `reserve-inventory`'s parent is valid and `remote=true`, and `checkout` is the root. The tests pin the same three invariants plus the wire format itself (a 55-character W3C header observed by a middleware wrapper).
- `tracetest.InMemoryExporter` with `WithSyncer` (synchronous export) is what makes the assertions possible — the same setup belongs in any service's tests that claim "we propagate context correctly".
- Everything here is deliberately manual. In production you'd wrap handlers with `otelhttp.NewHandler` and use `otelhttp.NewTransport` on clients, which perform exactly these Extract/Inject calls — plus span naming and status conventions — for you.

## Common Pitfalls

- **Forgetting `SetTextMapPropagator`.** The default global propagator is a no-op: Inject writes nothing, Extract finds nothing, and every service starts its own trace. This is the most common "our traces are all disconnected" cause.
- **Extracting but not using the returned context.** `Extract` *returns* the context; starting the span from `r.Context()` instead of the extracted one silently orphans it.
- **Propagating only on the happy path.** Background retries, queue consumers, and goroutines that outlive the request each need explicit decisions: continue the trace (pass the context/headers) or start a new one with a link — dropping it by accident is how async work disappears from traces.
- **Hand-parsing `traceparent`.** The propagator is the codec; header formats also include `tracestate` and, in some fleets, B3. Configure a composite propagator rather than string-splitting.
- **Confusing propagation with exporting.** This example exports to memory; propagation is orthogonal to where spans go. A service can forward context flawlessly while exporting nothing (fine: the trace continues around it) — see [otel](../otel/) for the exporter side.

## References

- [W3C Trace Context specification](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry docs — Context propagation](https://opentelemetry.io/docs/concepts/context-propagation/)
- [go.opentelemetry.io/otel/propagation package docs](https://pkg.go.dev/go.opentelemetry.io/otel/propagation)
- [otelhttp — the production middleware for these calls](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)

## Next Steps

- [otel](../otel/) — spans, attributes, error recording, and real exporters (stdout/Jaeger)
- [grpc-advanced](../grpc-advanced/) — gRPC's metadata is the same idea; otelgrpc interceptors propagate over it
- [httptest](../httptest/) — the server-testing machinery the propagation tests build on
