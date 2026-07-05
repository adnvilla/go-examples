# Recover

**Category:** basics
**Difficulty:** Intermediate

## Objective

Show `panic`/`recover`/`defer` working together: how a panic unwinds the call stack running deferred functions along the way, how `recover` inside a deferred function stops the unwind, and why a recovering function should re-panic on errors it doesn't actually know how to handle.

## Concepts Covered

- `defer` running in LIFO order as the stack unwinds
- `panic` propagating up through nested calls until something recovers it
- `recover()`, valid only inside a directly-deferred function, stopping the panic and letting the program continue
- Re-panicking when the recovered value isn't something the recoverer actually knows how to handle (here: a `runtime.Error`)

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
recover/
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
Calling g.
Printing in g 0
Printing in g 1
Printing in g 2
Printing in g 3
Panicking!
Defer in g 3
Defer in g 2
Defer in g 1
Defer in g 0
Recovered in f 4
Returned normally from f.
```

## Code Walkthrough

- `f` sets up a `defer` that calls `recover()` — this is the only place in the program `recover()` will actually catch anything, since it must be called directly inside a deferred function.
- `f` calls `g(0)`, which recurses (`g(1)`, `g(2)`, `g(3)`) — each level prints, then defers a print of its own before recursing further.
- `g(4)` (i.e. `i > 3`) panics instead of recursing again, with the string `"4"` as the panic value.
- The panic immediately starts unwinding the stack: each pending `defer fmt.Println("Defer in g", i)` runs, in reverse order of how they were deferred (3, 2, 1, 0) — this is why "Defer in g" prints count down even though "Printing in g" counted up.
- The unwind reaches `f`'s deferred function, where `recover()` catches the panic value (`"4"`). Since it's a plain string, not a `runtime.Error`, the type assertion `r.(runtime.Error)` fails, so `f` just prints it and returns normally — the panic is fully handled, and `main` continues to its next line as if nothing happened.

## Common Pitfalls

- **Calling `recover()` outside a deferred function.** It only has an effect when called directly by a function that was itself deferred — calling it from anywhere else (including a function *called by* a deferred function) always returns `nil`.
- **Swallowing every panic unconditionally.** This example specifically re-panics when the recovered value is a `runtime.Error` (e.g. a nil-pointer dereference or index-out-of-range) — those usually indicate a real bug, and silently continuing past one can hide corrupted state rather than fix anything.
- **Using panic/recover for ordinary error handling.** Idiomatic Go returns `error` values for expected failure cases (see [errors](../errors/)); `panic` is reserved for programmer errors or truly unrecoverable situations, recovered only at a boundary (e.g. an HTTP middleware, as in [http-server](../http-server/)) that can log it and continue serving other requests.
- **Assuming deferred functions run in the order they were declared.** They run in LIFO (last-in-first-out) order — the opposite of declaration order, as this example's "Defer in g" output demonstrates.

## References

- [Effective Go — Panic and Recover](https://go.dev/doc/effective_go#recover)
- [Go Blog — Defer, Panic, and Recover](https://go.dev/blog/defer-panic-and-recover)

## Next Steps

- [errors](../errors/) — the idiomatic alternative to panic for expected error conditions
- [http-server](../http-server/) — recovering from a handler panic at a middleware boundary in a real server
