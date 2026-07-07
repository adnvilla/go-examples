# Profiling

**Category:** performance
**Difficulty:** Beginner

## Objective

Show the minimal way to add CPU profiling to a Go program using [`github.com/pkg/profile`](https://github.com/pkg/profile), a thin wrapper around the standard `runtime/pprof` that handles starting/stopping and file placement in one `defer` line.

## Concepts Covered

- `defer profile.Start().Stop()` — CPU profiling for the lifetime of `main`, with automatic cleanup
- Where the profile file ends up (a temp directory, printed to stderr) and how to turn it into a visual report with `go tool pprof`
- A concurrent workload (1000 workers racing against a 5-second timeout each) as the thing being profiled

## Prerequisites

- Go 1.25+
- No external services or environment variables required
- To render the profile as a PDF/graph: [Graphviz](https://www.graphviz.org/download/) installed and on `PATH`

## Project Structure

```
profiling/
├── go.mod
├── main.go
├── SamplePdf.PNG   (example of the rendered call-graph output)
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

This takes roughly 5-10 seconds (workers sleep a random delay up to 10s, racing a 5s timeout) and prints a line telling you where the CPU profile was written, e.g.:
```
profile: cpu profiling enabled, /tmp/profile.../cpu.pprof
```

To turn that into a visual graph (requires Graphviz):
```bash
go tool pprof --pdf ./profiling /tmp/profile.../cpu.pprof > file.pdf
```

## Expected Output

1000 `Worker: N` startup lines, followed by 1000 `Worker N Time: Rms` completion lines (in whatever order goroutines finish), followed by 1000 `Response: N` / `TimeOut: N` lines — roughly half of each, since delays are uniform over 0-10s against a 5s cutoff:

```
Worker: 1
Worker: 2
...
Worker 708 Time: 5127
Worker 265 Time: 5135
...
profile: cpu profiling disabled, /tmp/profile.../cpu.pprof
```
(with `Response: N` and `TimeOut: N` lines mixed into the final block — the exact counts and ordering vary between runs.)

## Code Walkthrough

- `main` wraps the entire program in `defer profile.Start().Stop()` — `profile.Start()` begins CPU profiling immediately (its default mode) and returns a stopper whose `Stop()` (deferred) writes the profile file when `main` returns.
- `Init` fans out 1000 `worker` goroutines, each wrapping a `workerTask` with a 5-second timeout via `select`/`time.After` — the same timeout-racing shape as [concurrency/fan-out-timeout](../concurrency/fan-out-timeout/), just at a much larger scale (1000 goroutines instead of 10).
- `workerTask` sleeps a random duration (0-10s) to simulate variable-latency work, then sends a response — about half will exceed the 5s timeout given the uniform random range.
- All responses (real or timeout) are collected into a channel, drained into a slice once every worker's `wg.Done()` has fired, and printed.
- Profiling this program's CPU usage under `go tool pprof` mostly reveals goroutine scheduling and channel operations, since the actual "work" is just `time.Sleep` — a more CPU-bound example would show hotter allocation/computation paths in the resulting graph.

## Common Pitfalls

- **Forgetting `defer profile.Start().Stop()`, or placing it after other setup.** `profile.Start()` should typically be the very first line of `main`, and its `Stop()` deferred immediately, so as much of the program's execution as possible is captured.
- **Not having Graphviz installed when running `go tool pprof --pdf`.** The `--pdf` (and other graph-image) output formats shell out to `dot` from Graphviz; without it, `pprof` can still run in interactive/text mode (`go tool pprof cpu.pprof`) but can't render a graph.
- **Profiling a program dominated by `time.Sleep`/I/O wait instead of CPU work.** CPU profiling samples only CPU-bound time — a program mostly blocked on sleeps or network calls (like this one) won't show much in a CPU profile; use `profile.Start(profile.MemProfile)` or a blocking/mutex profile instead when investigating non-CPU bottlenecks.
- **Committing generated profile output (`.pprof`/`.pdf` files) to version control.** These are point-in-time artifacts of a specific run — only the illustrative `SamplePdf.PNG` (a static screenshot referenced by this README) is meant to be checked in; a stray generated `file.pdf` was removed during this migration.

## References

- [github.com/pkg/profile GitHub repository](https://github.com/pkg/profile)
- [Go Blog — Profiling Go Programs](https://go.dev/blog/pprof)
- [runtime/pprof package docs](https://pkg.go.dev/runtime/pprof)
- [Graphviz](https://www.graphviz.org/)

## Next Steps

- [reflection-bench](../reflection-bench/) — `runtime/pprof` used directly (without the `pkg/profile` wrapper) to compare two implementations
- [concurrency/fan-out-timeout](../concurrency/fan-out-timeout/) — the same timeout-racing pattern at a smaller, easier-to-read scale
