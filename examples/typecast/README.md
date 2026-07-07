# Typecast

**Category:** performance
**Difficulty:** Intermediate

## Objective

Benchmark four ways of calling the same operation on a value depending on how it's typed: a direct method call, an interface method call, a type switch, and a type assertion — to see whether dispatch strategy actually matters at the micro level.

## Concepts Covered

- Direct method calls on a concrete type (`*myint`) vs. calling the same method through an interface (`Inccer`)
- A type switch (`switch v := any.(type)`) as one way to recover a concrete type from an interface value
- A type assertion with the "comma ok" form (`if newint, ok := any.(*myint); ok`) as an alternative
- Using `testing.B`/`b.N` to compare the four approaches' per-operation cost

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
typecast/
├── go.mod
├── main_test.go
└── README.md
```

## How to Run

This is a benchmark-only package (`package typecast`, no `main`) — there's nothing to `go run`:

```bash
make bench
# or
go test -bench=. -run=^$ ./...
```

## Expected Output

Absolute numbers depend heavily on the Go version, compiler optimizations (inlining/devirtualization), and machine — at this operation's scale (a single integer increment), differences between strategies are often within noise:

```
BenchmarkIntmethod-14        	1000000000	         0.3315 ns/op
BenchmarkInterface-14        	1000000000	         0.3006 ns/op
BenchmarkTypeSwitch-14       	1000000000	         0.3562 ns/op
BenchmarkTypeAssertion-14    	1000000000	         0.2411 ns/op
```

## Code Walkthrough

- `myint` is a concrete type with one method, `inc()`, which satisfies the `Inccer` interface.
- `incnIntmethod` calls `i.inc()` directly on a `*myint` — no interface involved, the compiler knows the exact type at compile time.
- `incnInterface` takes an `Inccer` and calls `any.inc()` through the interface — this goes through Go's interface method dispatch (an indirect call via the interface's method table).
- `incnSwitch` and `incnAssertion` both start from an `Inccer` and recover the concrete `*myint` before calling `inc()` directly — a type switch and a type assertion respectively, each paying the cost of a runtime type check instead of a virtual dispatch.
- Each `Benchmark*` function wraps one of these four call patterns in `testing.B`'s standard `b.N`-iteration loop, so `go test -bench` reports a stable per-operation cost for each.

## Common Pitfalls

- **Reading too much into sub-nanosecond micro-benchmark numbers.** At this scale, results are extremely sensitive to compiler inlining decisions and can vary between Go versions — don't treat one run's ranking as a permanent, portable conclusion; if dispatch strategy matters for a real hot path, benchmark that actual code path, not an isolated microbenchmark.
- **Assuming interface calls are always meaningfully slower than concrete calls.** Modern Go compilers can devirtualize simple, provably-monomorphic interface calls — the theoretical "interface dispatch is slower" intuition doesn't always show up in practice.
- **Choosing a dispatch strategy for performance before it's shown to matter.** For most code, readability (does this need to be generic over multiple types, or not?) should drive the choice between direct calls, interfaces, and type switches/assertions — only optimize dispatch strategy after profiling shows it's a real bottleneck.

## References

- [testing package docs — Benchmarks](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Go Wiki — Compiler And Runtime Optimizations](https://go.dev/wiki/CompilerOptimizations)

## Next Steps

- [reflection-bench](../reflection-bench/) — a related comparison, contrasting reflection-based code against a direct, non-reflective implementation
- [oop](../oop/) — interfaces and polymorphism, the language feature underlying the interface-dispatch case here
