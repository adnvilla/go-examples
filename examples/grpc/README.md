# gRPC

**Category:** grpc
**Difficulty:** Intermediate

## Objective

Show a gRPC service end to end from a single `.proto` contract: a unary RPC, a server-streaming RPC, and errors as status codes — with server and client in one process so `go run .` demonstrates the full round trip and terminates on its own. Tests use `bufconn` for in-memory client/server testing (the gRPC analogue of `httptest`).

## Concepts Covered

- The `.proto` file as the API contract; `protoc` generating both messages (`greeter.pb.go`) and client/server stubs (`greeter_grpc.pb.go`)
- Unary RPC (`Greet`) vs server-streaming RPC (`Countdown` — `Send` per value, `nil` return closes the stream, client reads until `io.EOF`)
- `status.Error(codes.InvalidArgument, ...)` on the server and `status.FromError` on the client — typed error codes that survive the wire, no string parsing
- Embedding `UnimplementedGreeterServer` for forward compatibility
- `grpc.NewClient` with lazy dialing and `insecure.NewCredentials()` for localhost
- `GracefulStop` draining in-flight RPCs before `Serve` returns
- `bufconn` in tests: full gRPC stack over an in-memory listener — no TCP, no ports

## Prerequisites

- Go 1.25+
- Nothing extra to *run* it — the generated code is committed
- To *regenerate* after editing `greeter.proto`: `protoc` plus both Go plugins on `PATH`:
  ```bash
  brew install protobuf   # or your platform's package manager
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  make generate
  ```

## Project Structure

```
grpc/
├── go.mod
├── greeter.proto            # the source of truth: service + messages
├── greeterpb/
│   ├── greeter.pb.go        # generated: message types (do not edit)
│   └── greeter_grpc.pb.go   # generated: client + server stubs (do not edit)
├── main.go                  # service implementation + server + client demo
├── main_test.go             # bufconn-based tests
├── Makefile
└── README.md
```

## How to Run

```bash
make run    # server + client round trip in one process
make test   # bufconn tests, no network
```

## Expected Output

```
server: Greeter service up on a loopback port
client: Greet("Ada") -> "Hello, Ada!"
client: Greet("") -> code=InvalidArgument desc="name is required"
client: Countdown(3) -> 3 2 1 0
server: stopped cleanly
```

## Code Walkthrough

- `greeter.proto` declares the whole API: two RPCs and four messages. Everything in `greeterpb/` is derived from it by `make generate` — the proto file is the thing you edit, the `.pb.go` files are build artifacts kept in-tree (same convention as [protobuf](../protobuf/)) so users don't need `protoc` to build.
- `greeterServer` embeds `greeterpb.UnimplementedGreeterServer`, so adding an RPC to the proto later degrades to a runtime `codes.Unimplemented` instead of a compile break across every implementation.
- `Greet` returns `status.Error(codes.InvalidArgument, ...)` for a missing name. On the client, `status.FromError` recovers the code — the demo's second call shows `InvalidArgument` arriving intact, which is the gRPC idiom that replaces matching on error strings.
- `Countdown` receives `grpc.ServerStreamingServer[greeterpb.CountdownReply]` (the generics-based stream API): each `Send` pushes one message; returning `nil` ends the stream, which the client observes as `io.EOF` from `Recv` — the loop-until-EOF shape mirrors the `io.Reader` contract.
- `run` wires both halves: listen on `127.0.0.1:0` (random port, via `net.ListenConfig` so the listen carries a context), serve in a goroutine, then `grpc.NewClient` + calls, then `GracefulStop` — which waits for in-flight RPCs and makes `Serve` return `nil`, so the demo can assert a clean exit.
- `newTestClient` in the tests swaps the TCP listener for `bufconn.Listen`: same server, same generated client, zero network. `grpc.WithContextDialer` routes the client's connections into the in-memory pipe.

## Common Pitfalls

- **Hand-editing `*.pb.go` files.** They're regenerated wholesale by `make generate`; edits silently vanish. Change `greeter.proto` instead.
- **Renumbering or reusing proto field numbers.** Field numbers are the wire format. Changing `string name = 1` to `= 2` breaks every deployed client; deleted fields' numbers should be `reserved`.
- **Returning plain `errors.New` from handlers.** It reaches the client as `codes.Unknown` with the message flattened to a string — losing the machine-checkable code. Use `status.Error`/`status.Errorf` with the right `codes` value.
- **Treating `io.EOF` from `Recv` as a failure.** It's the normal end-of-stream marker; only non-EOF errors are real (and those carry status codes).
- **Forgetting `insecure.NewCredentials()` is for demos.** `grpc.NewClient` refuses to dial without transport credentials; localhost examples use insecure, production uses TLS (`credentials.NewTLS`).
- **Using `grpc.NewServer().Stop()` where `GracefulStop()` is meant.** `Stop` closes connections immediately, aborting in-flight RPCs — the same distinction as `http.Server.Close` vs `Shutdown`.

## References

- [gRPC-Go — Basics tutorial](https://grpc.io/docs/languages/go/basics/)
- [google.golang.org/grpc package docs](https://pkg.go.dev/google.golang.org/grpc)
- [gRPC status codes](https://grpc.io/docs/guides/status-codes/)
- [Protocol Buffers — Language guide (proto3)](https://protobuf.dev/programming-guides/proto3/)
- [bufconn package docs](https://pkg.go.dev/google.golang.org/grpc/test/bufconn)

## Next Steps

- [protobuf](../protobuf/) — the serialization layer underneath, without the RPC framework
- [httptest](../httptest/) — the same test-without-the-network idea for plain HTTP
- [graceful-shutdown-signals](../graceful-shutdown-signals/) — wiring `GracefulStop` to SIGTERM in a real service
