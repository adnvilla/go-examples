# io Readers and Writers

**Category:** basics
**Difficulty:** Beginner

## Objective

Show the `io.Reader`/`io.Writer` contract — the two one-method interfaces that everything byte-shaped in Go flows through (files, sockets, HTTP bodies, gzip, hashes) — and the standard-library combinators that compose them: `io.Copy`, `io.LimitReader`, `bufio.Scanner`, `io.TeeReader`, `io.MultiWriter`, and `io.Pipe`.

## Concepts Covered

- The raw `Read(p []byte) (n int, err error)` contract: caller-owned buffers, partial reads, and why the `n > 0` bytes must be processed *before* checking the error
- `io.Copy` — moving bytes from any reader to any writer without a hand-written loop
- `io.LimitReader` — capping how much downstream code can read (e.g. bounding untrusted input)
- `bufio.Scanner` — buffered, line-oriented consumption of a stream
- `io.TeeReader` — observing a stream as it's consumed (payload + sha256 checksum in a single pass)
- `io.MultiWriter` — fanning one write out to several sinks (the writer-side dual of `TeeReader`)
- `io.Pipe` — connecting writer-shaped code (`json.Encoder`) to reader-shaped code (`io.ReadAll`) across goroutines, and propagating failures with `CloseWithError`

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
io-readers-writers/
├── go.mod
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
--- manual Read loop (the io.Reader contract) ---
read 8 bytes: "the quic"
read 8 bytes: "k brown "
read 3 bytes: "fox"
reached EOF

--- io.Copy and io.LimitReader ---
copied 15 bytes: "streams compose"
limited copy: "only the "

--- bufio.Scanner (buffered line reading) ---
line 1: alpha
line 2: beta
line 3: gamma

--- io.TeeReader (observe a stream while consuming it) ---
consumed: "hash me as I stream"
sha256:   6d62014a7187de5a31e41868d862792782dcd09958f8f79c4c29f9e3657b691d

--- io.MultiWriter (one write, several sinks) ---
buffer 1: "fan out this write"
buffer 2: "fan out this write"
sha256:   810f3c31e3e431437aaf5e1a2521178bf406656678d029ae3d972c7991990156

--- io.Pipe (connect writer-shaped code to reader-shaped code) ---
received: {"name":"ping","seq":1}
```

## Code Walkthrough

- `readInChunks` drives a `strings.Reader` by hand with an 8-byte buffer: each `Read` fills as much of the buffer as it can and reports `n`; the final call returns the leftover 3 bytes, and the one after that returns `0, io.EOF`. The loop's shape — *use the `n` bytes first, then check the error* — is the part that matters: a reader is allowed to return data and `io.EOF` in the same call.
- `copyStreams` replaces that loop with `io.Copy`, which works for any reader/writer pair and returns the byte count. Wrapping the source in `io.LimitReader(r, 9)` makes the copy stop after 9 bytes — the standard way to bound how much you'll accept from an untrusted stream.
- `bufferedLines` shows why you rarely call `Read` directly for text: `bufio.Scanner` buffers the underlying reader and hands you whole lines. Its errors surface via `scanner.Err()` *after* the loop, not per-iteration.
- `teeWhileReading` computes a checksum without a second pass: `io.TeeReader(src, hasher)` copies every byte that flows through it into the hash writer, so `io.ReadAll` yields the payload and the hasher ends up with the digest simultaneously — the same trick used to hash an upload while saving it to disk.
- `writeOnceToMany` is the mirror image: `io.MultiWriter` returns a writer that duplicates each `Write` into every sink (two buffers and a hash here — identical content, and the digest differs from the previous demo only because the payload does).
- `pipeAcrossGoroutines` bridges an API that *wants a writer* (`json.NewEncoder(w).Encode`) with one that *wants a reader* (`io.ReadAll`) using `io.Pipe`, no intermediate buffer. Each `Write` blocks until the other side `Read`s, which is why the encoder runs in its own goroutine; `CloseWithError` forwards any encoding failure to the reading side (with `nil` it's a normal close).

## Common Pitfalls

- **Checking the error before consuming the bytes.** `Read` can return `n > 0` *and* `io.EOF` together — handling the error first silently drops the final chunk of the stream. This is the most common misuse of the interface.
- **Assuming `Read` fills the buffer.** A reader may return fewer bytes than requested at any time, not just at the end (network readers do this constantly). If you need exactly `len(buf)` bytes, use `io.ReadFull`, not a single `Read`.
- **Using `io.Pipe` from one goroutine.** Writes block until matched by reads, so writing and then reading sequentially in the same goroutine deadlocks. If you don't need streaming, a `bytes.Buffer` is the simpler tool — it's a non-blocking reader *and* writer.
- **Forgetting `scanner.Err()`.** `Scan` returning `false` means *stopped*, not necessarily *finished* — the loop looks identical for EOF and for a read error. `bufio.Scanner` also silently stops at lines longer than 64 KiB unless you raise the limit with `scanner.Buffer`.
- **`io.ReadAll` on unbounded input.** It buffers everything in memory; for large or untrusted streams, prefer `io.Copy` into a bounded destination or wrap the source in `io.LimitReader` first.

## References

- [io package docs](https://pkg.go.dev/io)
- [bufio package docs](https://pkg.go.dev/bufio)
- [io package docs — Pipe](https://pkg.go.dev/io#Pipe)
- [Effective Go — Interfaces](https://go.dev/doc/effective_go#interfaces)

## Next Steps

- [http-client](../http-client/) — response bodies are `io.ReadCloser`s; the contract shown here is why they must be drained and closed
- [embed](../embed/) — embedded files expose the same `io/fs` reader interfaces
- [serialization](../serialization/) — `encoding/json` works over these same reader/writer seams
