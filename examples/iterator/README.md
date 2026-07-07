# Iterator

**Category:** basics
**Difficulty:** Intermediate

## Objective

Show range-over-func iterators (Go 1.23+): functions matching `iter.Seq[V]` — `func(yield func(V) bool)` — can be used directly in a `for range` loop, composed lazily, and even represent infinite sequences without ever materializing a full slice.

## Concepts Covered

- The `iter.Seq[V]` shape and the `yield` callback contract (`yield` returns `false` to signal "stop early")
- Composable, lazy iterator transforms: `Filter`, `Map`, `Take`
- An infinite iterator (`Naturals`) that only produces as many values as a downstream `Take`/consumer actually asks for
- `slices.Collect` to materialize a finite iterator into a slice
- Ranging directly over an `iter.Seq[V]` with a plain `for ... range` — no slice in between

## Prerequisites

- Go 1.25+ (range-over-func requires Go 1.23+; this repo targets 1.25)
- No external services or environment variables required

## Project Structure

```
iterator/
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
first 5 evens: [2 4 6 8 10]
squares of first 4 odds: [1 9 25 49]
inline range: 10 11 12 13 14
```

## Code Walkthrough

- `Naturals(start)` returns an `iter.Seq[int]` that yields `start, start+1, start+2, ...` forever — it only stops when its `yield` call returns `false`, which happens when the consumer (directly, or through `Take`) decides it has enough.
- `Filter(seq, pred)` wraps another iterator, yielding only the elements `pred` accepts — it still forwards `yield`'s return value, so a downstream "stop early" signal propagates back through the filter to the original sequence.
- `Map(seq, f)` is the same shape, transforming each value with `f` instead of filtering.
- `Take(seq, n)` is what actually terminates an otherwise-infinite sequence: it counts yields and calls `return` once `n` have been produced, which causes the underlying `for range seq` in `Naturals` (or whatever it's wrapping) to stop too.
- `slices.Collect(iter.Seq[V])` drains a *finite* iterator into a `[]V` — calling it on an un-`Take`n `Naturals` would hang forever, since nothing ever tells it to stop.
- The final loop, `for n := range Take(Naturals(10), 5)`, shows that an `iter.Seq[V]` can be ranged over directly, without ever calling `Collect` — useful when you don't need a slice at all, just to process each value as it's produced.

## Common Pitfalls

- **Calling `slices.Collect` on an infinite iterator with no `Take`.** `Filter(Naturals(1), ...)` alone never terminates — always bound it (`Take`, or a `break` inside a direct `range`) before collecting to a slice.
- **Ignoring `yield`'s return value inside a custom iterator.** If `yield(v)` returns `false` (the consumer stopped early, e.g. via `break`) and the iterator keeps calling `yield` anyway, it wastes work and can misbehave — every custom iterator here checks `if !yield(v) { return }`.
- **Composing iterators expecting eager evaluation.** `Filter`/`Map`/`Take` are lazy — each element is only pulled through the whole chain when the final consumer asks for it, not precomputed up front. This is what lets `Naturals` (infinite) compose safely with `Take` (bounded) at all.

## References

- [Go Blog — Range over Function Types](https://go.dev/blog/range-functions)
- [iter package docs](https://pkg.go.dev/iter)
- [slices package docs](https://pkg.go.dev/slices)

## Next Steps

- [generics](../generics/) — the `Map`/`Filter`/`Reduce` functions here are the slice-based counterparts to this iterator style
