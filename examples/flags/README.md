# Flags

**Category:** cli
**Difficulty:** Beginner

## Objective

Show the standard-library `flag` package: declaring string/int/bool flags (two different ways), parsing them, and reading trailing positional arguments.

## Concepts Covered

- `flag.String` / `flag.Int` / `flag.Bool` — declare a flag, get a pointer to its value
- `flag.StringVar` — bind a flag to an existing variable instead of a new pointer
- `flag.Parse()` — must be called once, after all flags are declared, before reading any of them
- `flag.Args()` — the positional arguments left over after flag parsing

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
flags/
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or, with explicit flags and positional args:
go run . -word=opt -numb=7 -fork=true extra1 extra2
```

## Expected Output

With no arguments (`make run` / `go run .`):

```
word: foo
numb: 42
fork: false
svar: bar
tail: []
```

With `go run . -word=opt -numb=7 -fork=true extra1 extra2`:

```
word: opt
numb: 7
fork: true
svar: bar
tail: [extra1 extra2]
```

## Code Walkthrough

- `flag.String("word", "foo", "a string")` registers a `-word` flag with default `"foo"` and returns a `*string` — the value isn't known until `flag.Parse()` runs.
- `flag.Int`/`flag.Bool` follow the same shape for their respective types.
- `flag.StringVar(&svar, "svar", "bar", "a string var")` is the alternative form: instead of returning a new pointer, it writes into an existing variable (`svar`) you already declared — useful when the variable needs to be used elsewhere before flags are wired up.
- `flag.Parse()` must run after every flag is declared and before any flag value is read; it scans `os.Args[1:]`, matching `-name=value` (or `-name value`) pairs against the declared flags.
- Anything left over after all recognized flags are consumed — non-flag arguments — is available via `flag.Args()`, e.g. `extra1 extra2` in the example above.

## Common Pitfalls

- **Reading a flag pointer before calling `flag.Parse()`.** The pointer is valid, but it still holds the *default* value until `Parse()` runs.
- **Declaring flags after `flag.Parse()`.** Any flag declared after parsing has already happened won't be recognized on the command line.
- **Assuming positional args come before flags.** The `flag` package stops parsing flags at the first non-flag argument by default — put flags before positional arguments on the command line, or the "flag" will be treated as a positional argument instead.
- **Forgetting to dereference the pointer.** `wordPtr` is a `*string`; printing `wordPtr` prints an address, not the value — you need `*wordPtr`.

## References

- [flag package docs](https://pkg.go.dev/flag)
- [Go by Example — Command-Line Flags](https://gobyexample.com/command-line-flags)

## Next Steps

- [config](../config/) — configure a program from a JSON file instead of the command line
- [functional-options](../functional-options/) — configure a struct programmatically, without a CLI at all
