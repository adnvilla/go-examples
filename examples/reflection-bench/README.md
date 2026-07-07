# Reflection Bench

**Category:** performance
**Difficulty:** Intermediate

## Objective

Quantify the cost of `reflect`-based slice construction versus a plain `append` loop, and show `runtime/pprof` wired up for optional CPU/memory profiling of the comparison.

## Concepts Covered

- `reflect.MakeSlice` + `reflect.Append` to build a slice generically, without knowing the element type at compile time
- Benchmarking two implementations of the same operation (`CreateSlice` vs. `CreateSliceReflect`) to measure the reflection overhead directly
- `runtime/pprof.StartCPUProfile`/`WriteHeapProfile` gated behind `-cpuprofile`/`-memprofile` flags, so profiling is opt-in
- `_ "net/http/pprof"` blank import, which (if the program also started an HTTP server) would expose live profiling endpoints — here it's imported but unused beyond its `init()` side effect, since this program has no server

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
reflection-bench/
├── go.mod
├── main.go
├── main_test.go
└── README.md
```

## How to Run

```bash
make run    # builds a 10,000,000-element slice via reflection (takes under a second)
make test   # go test -race -count=1 ./... — runs TestSlice/TestReflectionSlice
make bench  # single-iteration comparison of CreateSlice vs. CreateSliceReflect
```

To capture profiles directly:
```bash
go run . -cpuprofile=cpu.pprof -memprofile=mem.pprof
go tool pprof cpu.pprof
```

## Expected Output

`make bench` (single iteration each, so `ns/op` is the cost of building one slice of `b.N == 1` elements at whatever size the benchmark's inner loop reaches):

```
BenchmarkSlice-14           	       1	       625.0 ns/op
BenchmarkSliceReflect-14    	       1	     35334 ns/op
```

Reflection-based construction was **~56x slower** in this run — the exact multiplier will vary by machine, but the direction (reflection is significantly slower) is consistent.

## Code Walkthrough

- `CreateSlice(n)` is the baseline: a plain `[]D` with ordinary `append` in a loop — no reflection involved.
- `CreateSliceReflect(n)` does the same job through `reflect.TypeOf`, `reflect.MakeSlice`, and `reflect.Append` — every element is boxed into a `reflect.Value` and appended through the reflection API, which is where the overhead comes from (type-safety checks and indirection that the compiler would otherwise handle statically).
- `BenchmarkSlices` sweeps both implementations across powers of two (1, 2, 4, ... 1024) using `b.Run` subtests, useful for seeing whether the relative overhead changes with size (it doesn't change qualitatively here — reflection stays proportionally slower at every size).
- `main` optionally starts a CPU profile (if `-cpuprofile` is set), builds a 10-million-element slice via `CreateSliceReflect` unconditionally, then optionally writes a heap profile (if `-memprofile` is set) after forcing a GC for up-to-date stats.

## Common Pitfalls

- **Reaching for `reflect` when a concrete type or generics would do.** This example exists specifically to demonstrate why: see [generics](../generics/) for the type-safe, compile-time alternative that avoids this overhead entirely for known-shape code.
- **Profiling without `runtime.GC()` first for heap profiles.** `WriteHeapProfile` reflects whatever garbage collector state exists at the time it's called — calling `runtime.GC()` immediately before (as `main` does) ensures the numbers reflect actual live memory, not stale pre-GC state.
- **Importing `net/http/pprof` expecting profiling endpoints without actually running an HTTP server.** The blank import here only registers pprof's handlers on `http.DefaultServeMux` — since this program never calls `http.ListenAndServe`, those endpoints are registered but unreachable; it's included to show the import, not because this particular program serves them.
- **Committing generated profile output (`.pprof`/`.pdf` files) to version control.** These are point-in-time artifacts of running the benchmark on a specific machine, not part of the example's source — they don't belong in the repository (removed during this migration).

## References

- [reflect package docs](https://pkg.go.dev/reflect)
- [runtime/pprof package docs](https://pkg.go.dev/runtime/pprof)
- [Go Blog — Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Blog — The Laws of Reflection](https://go.dev/blog/laws-of-reflection)

## Next Steps

- [generics](../generics/) — the compile-time-safe, zero-reflection-overhead alternative for generic collection code
- [profiling](../profiling/) — a more complete profiling setup using `github.com/pkg/profile`
- [typecast](../typecast/) — a related benchmark comparing dispatch strategies (type switch vs. type assertion vs. interface)
