# Composition (Struct Embedding)

**Category:** basics
**Difficulty:** Beginner

## Objective

Show struct embedding — Go's alternative to inheritance. An embedded type's fields and methods are promoted to the embedding struct, but the embedding struct can still override a promoted method, and the embedded value remains directly accessible.

## Concepts Covered

- Embedding a struct (`Person`) inside another (`Citizen`) by field with no name
- Promotion: `Citizen` gets `Person`'s fields (`Name`) and methods (`Location`) as if they were its own
- Overriding a promoted method (`Citizen.Talk` shadows `Person.Talk`)
- A testable `Example` function (`example_test.go`) as this package's runnable demonstration, since it's a library package with no `main`

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
composition/
├── composition.go
├── example_test.go
├── go.mod
└── README.md
```

## How to Run

This is a library package (`package composition`), not a `main` package, so there's nothing to `go run`. Its runnable demonstration is a Go [testable example](https://go.dev/blog/examples):

```bash
make run
# or
go test -v ./...
```

## Expected Output

```
=== RUN   Example
--- PASS: Example (0.00s)
PASS
ok  	github.com/adnvilla/go-examples/examples/oop/composition
```

(`go test` compares the `Example` function's actual stdout against the `// Output:` comment in `example_test.go` and passes only if they match exactly — the printed lines themselves are `Hello, my name is T'Challa and I'm from Wakanda`, `I'm at 1 Palace Way, Birnin Zana N/A 00000`, and `T'Challa is a citizen of Wakanda`.)

## Code Walkthrough

- `Person` has two methods, `Talk` and `Location`, plus a `Name` and an `Address`.
- `Citizen` embeds `Person` by declaring the field with only its type name (`Person`, not `p Person`) — this is what makes it an *embedded* field rather than an ordinary named one.
- Embedding promotes `Person`'s fields and methods: `c.Name` and `c.Location()` work directly on a `Citizen`, even though neither is declared on `Citizen` itself — Go resolves them through the embedded `Person`.
- `Citizen` declares its own `Talk` method, which **shadows** the promoted `Person.Talk` — calling `c.Talk()` invokes `Citizen`'s version, not `Person`'s. The original is still reachable explicitly as `c.Person.Talk()` if needed.
- `Citizen` also adds `Nationality`, a method with no equivalent on `Person` at all — embedding doesn't require overriding everything, only what needs to differ.

## Common Pitfalls

- **Expecting embedding to behave like class inheritance with dynamic dispatch.** If `Person.Location` internally called `p.Talk()`, it would always call `Person.Talk`, never `Citizen`'s override — there's no virtual dispatch through the embedded type, unlike inheritance in many OOP languages.
- **Forgetting the embedded value is still directly accessible.** `c.Person` and `c.Person.Talk()` remain valid — embedding promotes members, it doesn't hide the embedded type.
- **Naming collisions between embedded types.** Embedding two types that both define the same method/field name requires explicit qualification (`c.TypeA.Field`) at the call site — Go won't guess which one you meant.
- **Reaching for embedding when a named field would be clearer.** Embedding is best when the embedding type genuinely "is-a-kind-of" or "has-all-the-behavior-of" the embedded type; a named field (`person Person`) is often more readable when you just need to hold a related value.

## References

- [Effective Go — Embedding](https://go.dev/doc/effective_go#embedding)
- [Go Blog — The Go Blog: Testable Examples in Go](https://go.dev/blog/examples)

## Next Steps

- [oop](../) — methods and interfaces (polymorphism), the sibling pattern to this one
- [inject](../../inject/) — interfaces used for dependency injection rather than data reuse
