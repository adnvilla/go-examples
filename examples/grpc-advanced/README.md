# gRPC Advanced

**Category:** grpc
**Difficulty:** Advanced

## Objective

Show the gRPC machinery production services actually depend on, beyond the request/reply basics of [grpc](../grpc/): **interceptors** on both sides (metadata-based auth as server middleware, token injection as client middleware), **bidirectional streaming** over a single RPC, and **deadline propagation** — the client's timeout cancels the handler's context on the server, so abandoned work stops instead of completing into the void. Server and client run in one process; `go run .` is deterministic and self-terminating.

## Concepts Covered

- `grpc.UnaryInterceptor` / `grpc.StreamInterceptor` (server) and `grpc.WithUnaryInterceptor` / `grpc.WithStreamInterceptor` (client) — gRPC's middleware seam, and why unary and stream interceptors are separate registrations
- Metadata as headers: `metadata.AppendToOutgoingContext` on the client, `metadata.FromIncomingContext` + `codes.Unauthenticated` on the server — the handler never runs for rejected calls
- Bidirectional streaming: `grpc.BidiStreamingServer`, independent `Send`/`Recv` on both sides, `CloseSend`, and `io.EOF` as the clean end-of-conversation signal
- Deadline propagation: `context.WithTimeout` on the client crossing the wire, the handler observing `ctx.Done()`, and `status.FromContextError` translating the cancellation into the right status code
- Structural verification: the demo asserts the 5s handler was canceled in ~100ms rather than trusting log output
- `bufconn` tests covering both auth outcomes, the stream round trip, and the deadline behavior

## Prerequisites

- Go 1.25+
- Nothing extra to run it — generated code is committed; regenerating `echo.proto` needs `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (see [grpc](../grpc/) for install steps), then `make generate`

## Project Structure

```
grpc-advanced/
├── go.mod
├── echo.proto            # service contract: unary, slow-unary, bidi stream
├── echopb/               # generated code (do not edit)
├── main.go               # interceptors + server + client demos
├── main_test.go          # bufconn tests
├── Makefile
└── README.md
```

## How to Run

```bash
make run    # all three demos in one process
make test   # bufconn tests, no network
```

## Expected Output

```
--- interceptors: auth via metadata ---
  [server interceptor] authorized unary /echo.v1.Echo/UnaryEcho
with token:    "echo: hello"
without token: code=Unauthenticated desc="missing or invalid bearer token" (handler never ran)

--- deadline propagation: client timeout cancels the handler ---
  [server interceptor] authorized unary /echo.v1.Echo/SlowEcho
code=DeadlineExceeded after ~100ms (the 5s handler was canceled, not awaited)

--- bidirectional streaming ---
  [server interceptor] authorized stream /echo.v1.Echo/BidiEcho
sent "one" -> received "echo: one"
sent "two" -> received "echo: two"
sent "three" -> received "echo: three"
CloseSend acknowledged: server closed its side, Recv returned io.EOF

server: stopped cleanly
```

## Code Walkthrough

- The **auth pair** is the canonical interceptor use case. Client side, `tokenUnaryInterceptor`/`tokenStreamInterceptor` append the bearer token to the outgoing metadata of every call — application code never mentions auth. Server side, `authUnaryInterceptor`/`authStreamInterceptor` check it *before* invoking the handler, so an unauthenticated call is rejected without any handler code running (the demo shows this: the anonymous connection gets `Unauthenticated` and the service's own logic is untouched).
- Unary and stream interceptors must **both** be registered: they are disjoint code paths, and a team that only wires the unary one has unauthenticated streaming endpoints. The demo's second connection (`anonymous`) exists precisely to prove the rejection path.
- `SlowEcho` is the deadline demo's server half: it `select`s between its 5s "work" and `ctx.Done()`. When the client's 100ms deadline expires, gRPC cancels the handler's context *across the wire*; `status.FromContextError` maps the cancellation to `DeadlineExceeded`. The client asserts the elapsed time (~100ms, not 5s) — the handler was truly abandoned, which is what keeps a slow dependency from pinning server goroutines.
- `BidiEcho` shows the streaming contract: each side reads and writes independently on one RPC. The server's loop is `Recv` → `Send` until `Recv` returns `io.EOF` (the client called `CloseSend`); the server returning `nil` then closes its own side, which the client observes as `io.EOF` on its final `Recv`. Nothing about the API forces the echo shape — the same primitives support chat, sync protocols, or long-lived subscriptions with sporadic traffic in both directions.
- The tests boot the *full* server — interceptors included — on `bufconn`, and parameterize the client on `withToken`, covering: authorized unary, rejected unary, rejected stream (note: stream auth errors surface on the first `Recv`, not on stream open), the bidi round trip, and the deadline cancellation with the same structural elapsed-time assertion.

## Common Pitfalls

- **Registering only the unary interceptor.** Streams silently skip it. Every cross-cutting concern needs both registrations (or a helper that wires the pair).
- **Expecting stream auth failures at stream creation.** Opening a stream is lazy; the `Unauthenticated` status arrives on the first `Recv`/`Send`. Tests that only check the open call pass against a broken server.
- **Handlers that ignore `ctx.Done()`.** The deadline still fires on the client, but the server keeps computing — under load, that's goroutines and downstream calls spent on answers nobody will read. Long-running handlers must select on the context (or pass it down, as every example in this repo does).
- **Returning `ctx.Err()` raw.** It arrives as `codes.Unknown`. `status.FromContextError` maps `context.DeadlineExceeded`/`Canceled` to their proper codes so clients can branch on them.
- **Forgetting `CloseSend` on bidi streams.** The server's `Recv` never returns `io.EOF`, both sides wait forever, and the RPC leaks until the connection dies.
- **Doing per-call work in interceptors that belongs in handlers.** Interceptors run for *every* RPC; heavy logic there is a global tax. Keep them to cross-cutting concerns.

## References

- [gRPC-Go — Interceptors](https://grpc.io/docs/guides/interceptors/)
- [gRPC-Go — Deadlines](https://grpc.io/docs/guides/deadlines/)
- [gRPC-Go docs — metadata package](https://pkg.go.dev/google.golang.org/grpc/metadata)
- [gRPC-Go — Bidirectional streaming tutorial](https://grpc.io/docs/languages/go/basics/#bidirectional-streaming-rpc)

## Next Steps

- [grpc](../grpc/) — the basics this builds on (unary, server-streaming, status codes, bufconn)
- [context](../context/) — the cancellation semantics that deadline propagation extends across processes
- [circuit-breaker](../circuit-breaker/) — the resilience layer that typically wraps these clients
