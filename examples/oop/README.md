# OOP: Methods, Interfaces, and Polymorphism

**Category:** basics
**Difficulty:** Beginner

## Objective

Show how Go covers the common "OOP" patterns without classes or inheritance: methods on structs, and runtime polymorphism through interfaces. See [oop/composition](composition/) for struct embedding, Go's alternative to inheritance.

## Concepts Covered

- Declaring methods on a value receiver (`func (r Rect) Area() float64`)
- Defining a small interface (`Shape`) describing behavior, not implementation
- Polymorphism: a `[]Shape` holding different concrete types (`Rect`, `Circle`), each satisfying the interface independently
- `%T` in `fmt.Printf` to print a value's dynamic (concrete) type

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
oop/
├── go.mod
├── main.go
├── composition/
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
main.Rect — area: 50.00, perimeter: 30.00
main.Circle — area: 153.94, perimeter: 43.98
```

## Code Walkthrough

- `Shape` declares two methods, `Area()` and `Perimeter()`, with no mention of which types implement it — in Go, a type satisfies an interface simply by having matching methods, with no explicit `implements` declaration.
- `Rect` and `Circle` are unrelated structs, each with their own `Area`/`Perimeter` implementations — there's no shared base type or inheritance between them.
- `printShape(s Shape)` accepts anything satisfying `Shape` and calls its methods through the interface — it has no idea, at compile time, whether it's holding a `Rect` or a `Circle`.
- `main` builds a `[]Shape` mixing both concrete types; ranging over it and calling `printShape` on each demonstrates runtime polymorphism — the correct `Area`/`Perimeter` implementation is dispatched based on each value's actual (dynamic) type.
- `%T` in the format string prints that dynamic type (`main.Rect`, `main.Circle`), which is otherwise invisible once a value is boxed into the `Shape` interface.

## Common Pitfalls

- **Looking for `class`/`extends`/`implements` keywords.** Go has none — behavior comes from methods on any type, and interface satisfaction is structural (implicit), not declared.
- **Defining a fat interface with many methods "just in case."** `Shape` has exactly the two methods `printShape` needs — see [inject](../inject/) for more on why small, consumer-defined interfaces are idiomatic.
- **Choosing a pointer receiver vs. value receiver inconsistently.** `Rect`/`Circle` use value receivers here because they don't need to mutate their fields; a type mixing value and pointer receivers across its methods can fail to satisfy an interface in surprising ways (a value doesn't automatically satisfy an interface requiring a pointer-receiver method).
- **Expecting inheritance-style code reuse.** There's no `Rect extends Shape` — shared behavior across types in Go usually comes from either interfaces (behavioral contract) or struct embedding (see [composition](composition/), field/method reuse).

## References

- [Effective Go — Interfaces and other types](https://go.dev/doc/effective_go#interfaces_and_types)
- [Go Tour — Methods and interfaces](https://go.dev/tour/methods/1)
- [Go Proverbs — "The bigger the interface, the weaker the abstraction"](https://go-proverbs.github.io/)

## Next Steps

- [oop/composition](composition/) — struct embedding as Go's alternative to inheritance
- [inject](../inject/) — consumer-defined interfaces used for dependency injection
