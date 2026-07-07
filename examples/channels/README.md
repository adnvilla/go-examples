# Channels

**Category:** concurrency
**Difficulty:** Beginner

## Objective

Show the minimal pattern for one goroutine producing values and another consuming them over an unbuffered channel, and how closing the channel is what lets the consumer's `range` loop — and therefore the program — terminate.

## Concepts Covered

- Starting a goroutine with `go`
- Sending and receiving on an unbuffered `chan int`
- Consuming a channel with `range` until it's closed
- Closing a channel from the sender side to signal "no more values"
- `sync.WaitGroup` to block `main` until the consumer goroutine finishes

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
channels/
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

```
Received 1 Received 2 Received 3 Received 4 Received 5 Received 6 Received 7 Received 8 Received 9 Received 10
```

## Code Walkthrough

- `main` registers one unit of work on a `sync.WaitGroup`, then starts `printer` as a goroutine, handing it the channel `c`.
- `printer` blocks in `for i := range c`, printing every value it receives, until the channel is closed — at which point the `range` loop exits on its own and `printer` calls `wg.Done()`.
- Back in `main`, the loop sends the integers 1 through 10 on `c`. Each send blocks until `printer` is ready to receive, since `c` is unbuffered.
- `close(c)` is called once all values have been sent. Closing is what allows `printer`'s `range` loop to end — without it, `printer` would block forever waiting for one more value.
- `wg.Wait()` blocks `main` until `printer` signals completion, guaranteeing all output is flushed before the program exits.

## Common Pitfalls

- **Forgetting to close the channel.** The consumer's `range` loop never sees an "end of stream" signal and blocks forever — a goroutine leak (and, here, a deadlock since `main` waits on `wg`).
- **Closing a channel from the receiver side.** Only the sender should close a channel; a receiver doesn't know if more sends are coming.
- **Closing a channel twice, or sending on a closed channel.** Both panic at runtime — close exactly once, after the last send.
- **Sending on an unbuffered channel with no active receiver.** The send blocks indefinitely; unbuffered channels require a goroutine on the other end ready to receive.

## References

- [Effective Go — Channels](https://go.dev/doc/effective_go#channels)
- [Go Tour — Channels](https://go.dev/tour/concurrency/2)
- [Go Blog — Share Memory By Communicating](https://go.dev/blog/codelab-share)

## Next Steps

- [share-memory-by-communicating](../share-memory-by-communicating/) — a more realistic channel-based worker (URL poller)
- [concurrency/worker-pool](../concurrency/worker-pool/) — multiple goroutines consuming a shared channel
- [concurrency/fan-out-timeout](../concurrency/fan-out-timeout/) — combining channels with `select` and `time.After`
