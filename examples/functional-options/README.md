# Functional Options

**Category:** design-patterns
**Difficulty:** Beginner

## Objective

Show how to configure a struct with sensible defaults, letting callers override only what they need, without exporting struct fields and without a constructor overload per combination of settings.

## Concepts Covered

- The `Option func(*T)` pattern: each option is a closure that mutates one field
- A constructor (`NewServer`) that starts from defaults and applies a variadic list of options on top
- Keeping the configuration struct (`serverConfig`) unexported while its options (`WithHost`, `WithPort`, ...) are the only exported way to set it
- Why this scales better than adding a new constructor parameter per setting

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
functional-options/
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
default server: addr=0.0.0.0:8080 maxConns=100
custom server:  addr=0.0.0.0:9090 readTimeout=30s maxConns=500
```

## Code Walkthrough

- `Option` is defined as `func(*serverConfig)` — a function type whose only job is to mutate one field of the config it's given.
- Each `With*` function (`WithHost`, `WithPort`, ...) is a small factory that returns an `Option` closing over the value the caller passed in.
- `NewServer(opts ...Option)` builds a `serverConfig` populated with defaults first, then loops over every supplied `Option` and applies it — so an option can only ever override a default, never omit a required field.
- Calling `NewServer()` with no options at all (`s1`) is valid and produces a fully-defaulted server; calling it with a handful of options (`s2`) only changes what's explicitly requested (`port`, `readTimeout`, `maxConns`), leaving `host` and `writeTimeout` at their defaults.

## Common Pitfalls

- **Exporting the config struct's fields directly instead of going through options.** That reintroduces the exact problem this pattern avoids — callers depending on field names/types directly, making it harder to add or rename settings later without breaking them.
- **Making `NewServer` return an error just because options exist.** Options here can't fail; if an option *can* fail (e.g. validating a port range), have it return `(Option, error)` or accumulate errors in the config and check them once inside `NewServer`.
- **Order-dependent options that aren't supposed to be.** Since options apply in the order given, two options that both set the same field will leave the *last* one's value — usually fine, but worth being intentional about if some options should be mutually exclusive.
- **Overusing this pattern for structs with only one or two settings.** A couple of constructor parameters, or a plain config struct literal, is simpler when there's no need for defaults-plus-overrides or backward-compatible growth.

## References

- [Dave Cheney — Functional options for friendly APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
- [Uber Go Style Guide — Functional Options](https://github.com/uber-go/guide/blob/master/style.md#functional-options)

## Next Steps

- [inject](../inject/) — constructor-based dependency injection, a related but distinct configuration pattern
- [oop](../oop/) — methods and interfaces, the building blocks this pattern is built on
