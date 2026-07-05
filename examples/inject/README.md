# Inject

**Category:** design-patterns
**Difficulty:** Beginner

## Objective

Show idiomatic Go dependency injection: passing interfaces into constructors explicitly, with no reflection, struct tags, or DI framework. Contrast with [wire](../wire/), which generates this same wiring at compile time.

## Concepts Covered

- Defining small, consumer-side interfaces (`Namer`, `Planter`) for what a dependency needs to do, not what a concrete type is
- Constructor functions (`NewNameAPI`, `NewApp`, ...) that accept dependencies as parameters and return a ready-to-use value
- Composing a dependency graph by hand in `main`, one constructor call at a time
- Depending on `http.RoundTripper` (an interface) rather than `*http.Client` (a concrete type), so a test could substitute a fake transport

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
inject/
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
Spock is from Vulcan
```

## Code Walkthrough

- `Namer` and `Planter` are minimal interfaces describing exactly one method each — the shape `App` actually needs, not the shape of any particular implementation.
- `NameAPI` and `PlanetAPI` are concrete types that happen to satisfy those interfaces; each is constructed via `NewNameAPI`/`NewPlanetAPI`, taking an `http.RoundTripper` (here, `http.DefaultTransport`) as an explicit dependency.
- `App` depends only on the two interfaces (`names Namer`, `planets Planter`), never on the concrete `*NameAPI`/`*PlanetAPI` types — so a test could hand `NewApp` a fake `Namer`/`Planter` without touching `App`'s code at all.
- `main` wires the whole graph by hand: construct the leaves first (`nameAPI`, `planetAPI`), then the thing that depends on them (`app`). This is the entire "framework" — there isn't one.

## Common Pitfalls

- **Depending on concrete types instead of interfaces.** If `App` held `*NameAPI` directly, it could never be tested without a real (or heavily mocked) `NameAPI` — depending on `Namer` instead means any type satisfying that one-method interface works.
- **Defining interfaces on the producer side.** `Namer`/`Planter` are declared where they're *used* (by `App`), not where they're *implemented* (`NameAPI`/`PlanetAPI`) — this is idiomatic Go and keeps interfaces minimal and consumer-driven.
- **Reaching for a DI framework or reflection-based container before outgrowing manual wiring.** Manual constructor injection, as shown here, scales fine until the dependency graph gets large or repetitive enough that generating the wiring (see [wire](../wire/)) starts paying for itself.
- **Interfaces with more methods than a consumer needs.** Keep interfaces as small as the consumer requires (`Namer` has exactly one method) rather than mirroring a large concrete type's full method set.

## References

- [Effective Go — Interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go Proverbs — "The bigger the interface, the weaker the abstraction"](https://go-proverbs.github.io/)

## Next Steps

- [wire](../wire/) — the same dependency graph, wired at compile time by code generation instead of by hand
- [functional-options](../functional-options/) — a complementary pattern for configuring one of these constructed types
