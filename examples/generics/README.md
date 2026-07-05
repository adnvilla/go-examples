# Generics

**Category:** basics
**Difficulty:** Intermediate

## Objective

Show what type parameters (Go 1.18+) buy you: collection functions (`Map`/`Filter`/`Reduce`) that work over any element type without duplicating code per type, constraints that restrict what a type parameter can be, and a generic `Option[T]` type that makes "value absent" explicit instead of relying on a nil pointer or a zero value.

## Concepts Covered

- Unconstrained type parameters (`[T any]`) for `Map`, `Filter`, `Reduce`
- Two type parameters in one function (`Map[T, U any]`) when input and output types differ
- Built-in constraints: `cmp.Ordered` (works with `slices.Min`) and `comparable` (required for map keys)
- A generic struct (`Option[T]`) with generic methods, as an alternative to nil-pointer-means-absent

## Prerequisites

- Go 1.24+ (generics require Go 1.18+; this repo targets 1.25)
- No external services or environment variables required

## Project Structure

```
generics/
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

`Keys` is sorted explicitly before printing, so this output is fully deterministic:

```
Map (double): [2 4 6 8 10 12]
Filter (even): [2 4 6]
Reduce (sum): 21
Min string: apple
Keys: [a b c]
Some(42).OrElse(0): 42
None[int]().OrElse(0): 0
```

## Code Walkthrough

- `Map[T, U any]` takes a `[]T` and a `func(T) U`, returning `[]U` — two type parameters because the input and output element types can differ (here they don't, but the signature doesn't assume that).
- `Filter[T any]` and `Reduce[T, U any]` follow the same shape: generic over the element type(s), with the transformation supplied as a function argument.
- `Min[T cmp.Ordered]` constrains `T` to types `cmp`'s ordering operators support (all built-in numeric types and strings) — this is what lets it call `slices.Min(s)` and compile-time-guarantee `s`'s elements are ordered.
- `Keys[K comparable, V any]` constrains the map's key type to `comparable`, which every valid Go map key type already satisfies — the constraint exists so the compiler can verify `K` is usable as a map key at all.
- `Option[T]` wraps a `*T`: `Some` allocates and points at a copy of the value, `None` leaves the pointer nil. `IsPresent`, `Unwrap`, and `OrElse` are ordinary methods, just parameterized by `T` from the struct they're defined on — the caller never has to name `T` explicitly (`Some(42)` infers `T = int`).

## Common Pitfalls

- **Calling `Unwrap()` on a `None` value.** It dereferences a nil pointer and panics — always check `IsPresent()` first, or prefer `OrElse` when you have a sensible default.
- **Reaching for generics before reaching for `any` + type assertions, or before just writing the specific version.** Generics remove *duplication*, not necessarily complexity — a single non-generic function is still simpler when there's no second type ever going to use it.
- **Forgetting a constraint is needed at all.** `T any` compiles even where you actually need ordering or comparison — the compiler only catches this at the call site of an operation `any` doesn't support (e.g. `<` or `==` inside the generic function itself), which can be confusing without understanding constraints.
- **`Keys` returning map keys in "unspecified order.**" Go intentionally randomizes map iteration — `Keys` must be sorted (as done here with `slices.Sort`) before the result can be printed or compared deterministically.

## References

- [Go Blog — An Introduction to Generics](https://go.dev/blog/intro-generics)
- [Go Blog — When To Use Generics](https://go.dev/blog/when-generics)
- [cmp package docs](https://pkg.go.dev/cmp)
- [slices package docs](https://pkg.go.dev/slices)

## Next Steps

- [iterator](../iterator/) — range-over-func iterators, often combined with generic helpers like these
- [pool](../pool/) — a generic worker pool with per-task error tracking
