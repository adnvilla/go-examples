# WebSocket

**Category:** networking
**Difficulty:** Advanced

## Objective

Build the canonical WebSocket architecture — an HTTP endpoint that upgrades connections into a **broadcast hub** — with `coder/websocket`, and get the lifecycle details right: per-client reader goroutines (control frames like pongs are only processed while a `Read` is in flight, a real gotcha discovered live while building this), ping for liveness, read limits, per-write deadlines so one slow client can't wedge the room, and clean closes with status codes. Server and two clients run in one process; `go run .` shows the full chat round trip and terminates on its own.

## Concepts Covered

- `websocket.Accept` in a plain `http.Handler` — the upgrade is just HTTP, so the hub mounts on any mux or server
- The hub shape: register/unregister under a mutex, fan-out `broadcast` with a **per-write timeout** (the slow-consumer policy made explicit)
- The client reader pump: a goroutine that `Read`s continuously into a channel — not an optimization, but the requirement for pings, pongs, and close frames to be processed at all
- `Ping(ctx)` as real liveness (an open TCP connection says nothing about the peer)
- `SetReadLimit` bounding what one frame can cost; `Close(StatusNormalClosure, ...)` vs `CloseNow`
- Graceful teardown ordering: clients close → hub unregisters → `http.Server.Shutdown`

## Prerequisites

- Go 1.25+
- No external services or environment variables required — `coder/websocket` (the maintained successor of `nhooyr.io/websocket`) is the canonical minimal WebSocket library and the topic being taught

## Project Structure

```
websocket/
├── go.mod
├── hub.go        # Hub: accept, register, read loop, broadcast (the reusable part)
├── main.go       # server + two clients: ping, broadcasts, clean close
├── hub_test.go   # broadcast fan-out and unregister-on-close over httptest
├── Makefile
└── README.md
```

## How to Run

```bash
make run
make test
```

## Expected Output

```
--- hub up; two clients connect ---
hub reports 2 connected clients
alice: ping round trip ok

--- broadcast: every message reaches every client ---
alice received: "alice: hola"
bob received: "alice: hola"
alice received: "bob: qué tal"
bob received: "bob: qué tal"

--- clean close: status codes end the conversation ---
hub reports 0 connected clients after closes
server: stopped cleanly
```

## Code Walkthrough

- `Hub.ServeHTTP` is an ordinary `http.Handler`: `websocket.Accept` performs the upgrade (and writes the HTTP error itself if it fails), then the handler *is* the connection's read loop — register, read-and-broadcast until error, unregister via `defer`. One goroutine per connection, lifecycle tied to the handler, nothing leaked.
- `broadcast` snapshots the client list under the mutex, then writes outside it with a 2-second deadline per client. That deadline is the **slow-consumer policy**: a wedged client loses messages instead of blocking the room. Real systems choose between dropping (here), buffering per client, or disconnecting laggards — but they must choose; the default (block everyone) is the worst option.
- `client`/`connect` on the demo side runs the **reader pump**: a goroutine that `Read`s continuously and forwards data frames into a channel. This is the detail that bit during development: `Ping` timed out until the pump existed, because pongs (like all control frames) are only processed *inside* a `Read` call. A client that reads "when it expects something" has a connection that is only alive when it expects something.
- `alice.conn.Ping(ctx)` completes only because both sides are reading — the server's loop processes the ping and replies; alice's pump processes the pong. That's the liveness check to run on idle connections, on a timer, in production.
- Teardown is ordered and verified: `Close(StatusNormalClosure)` performs the close handshake, the hub's counts drain to zero (asserted, not assumed), then `server.Shutdown` ends the HTTP side.

## Common Pitfalls

- **Reading only when you expect data.** Control frames (ping/pong/close) are processed during `Read`; without a continuous reader, pings time out and closes go unnoticed. Every long-lived client needs a read pump — this example's original draft didn't, and its `Ping` deadlocked.
- **Broadcasting while holding the lock, with no write deadline.** One slow client backpressures the entire hub. Snapshot the list, write with deadlines, and pick an explicit slow-consumer policy.
- **No `SetReadLimit`.** The default protects you (32 KiB in this library), but know the number: a malicious client sending a giant frame is an allocation attack.
- **Confusing `Close` and `CloseNow`.** `Close` performs the status-code handshake (use it on the happy path); `CloseNow` tears down the TCP connection (use it in `defer` as the failsafe).
- **Treating an open connection as a live user.** NAT boxes and phones drop silently; only ping round trips (or application heartbeats) distinguish "connected" from "gone".
- **Skipping `OriginPatterns` in real browsers-facing servers.** `Accept` enforces same-origin by default; configure it consciously rather than disabling verification to "make it work".

## References

- [coder/websocket documentation](https://pkg.go.dev/github.com/coder/websocket)
- [RFC 6455 — The WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455)
- [MDN — WebSockets API](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API)

## Next Steps

- [http-server](../http-server/) — the plain-HTTP server machinery the upgrade endpoint mounts on
- [concurrency/pipeline](../concurrency/pipeline/) — the channel patterns behind per-client send buffers
- [graceful-shutdown-signals](../graceful-shutdown-signals/) — wiring the teardown ordering to SIGTERM
