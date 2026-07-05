# Testing Patterns

**Category:** testing
**Difficulty:** Intermediate

## Objective

Show four testing techniques beyond a basic `Test*` function: table-driven tests, named subtests with `t.Run`, running tests in parallel with `t.Parallel`, and fuzz testing (Go 1.18+) to check an invariant against generated inputs. See [testing](../testing/) for the fundamentals these build on.

## Concepts Covered

- Table-driven tests: one test function, a slice of `{input, want}` cases, one loop
- `t.Run(name, func(t *testing.T) {...})` — named subtests, individually addressable via `go test -run TestAdd/negative`
- `t.Parallel()` at both the top-level test and subtest level, to run independent cases concurrently
- `errors.Is` inside a table-driven test to assert on a specific sentinel error (`ErrDivByZero`)
- `FuzzAdd`: a seed corpus (`f.Add(...)`) plus a property check (`f.Fuzz(func(t *testing.T, a, b int) {...})`) verifying `Add` is commutative for any generated inputs

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
testing-patterns/
├── go.mod
├── main.go
├── math.go
├── math_test.go
└── README.md
```

## How to Run

```bash
make run    # go run . — runs the small FizzBuzz demo in main.go
make test   # go test -race -count=1 ./... — runs every Test* function
make fuzz   # go test -fuzz=FuzzAdd -fuzztime=30s — fuzz for 30s (FUZZTIME=10s to override)
```

## Expected Output

`make run`:
```
FizzBuzz
Fizz
```

`go test -v ./...` (abridged — full output lists every named subtest):
```
--- PASS: TestDivide (0.00s)
    --- PASS: TestDivide/normal (0.00s)
    --- PASS: TestDivide/by_zero (0.00s)
    --- PASS: TestDivide/fraction (0.00s)
--- PASS: TestAdd (0.00s)
    --- PASS: TestAdd/positive (0.00s)
    --- PASS: TestAdd/mixed (0.00s)
    --- PASS: TestAdd/zeros (0.00s)
    --- PASS: TestAdd/negative (0.00s)
--- PASS: TestFizzBuzz (0.00s)
    ...
=== RUN   FuzzAdd
--- PASS: FuzzAdd (0.00s)
    --- PASS: FuzzAdd/seed#0 (0.00s)
    --- PASS: FuzzAdd/seed#1 (0.00s)
    --- PASS: FuzzAdd/seed#2 (0.00s)
PASS
```

`make fuzz` (or `go test -fuzz=FuzzAdd`) runs the seed corpus plus continuously generated inputs for the configured `-fuzztime`, printing nothing on success unless a failing input is found (in which case it's saved under `testdata/fuzz/FuzzAdd/` for `go test` to replay automatically from then on).

## Code Walkthrough

- `TestAdd`/`TestDivide`/`TestFizzBuzz` each declare a `cases` slice of anonymous structs (input fields + `want`), then loop over it calling `t.Run(tc.name, ...)` — this is the table-driven pattern: adding a new case means adding one line to the table, not writing a new test function.
- `t.Parallel()` inside both the outer test function and each subtest signals that independent cases can run concurrently — `go test` pauses each parallel test until every non-parallel test in the same "batch" finishes, then runs all parallel ones together.
- `TestDivide` uses `errors.Is(err, tc.wantErr)` rather than `err == tc.wantErr` — the idiomatic way to check for a specific sentinel error (`ErrDivByZero`), consistent with the wrapping-aware comparisons shown in [errors](../errors/).
- `FuzzAdd`'s seed corpus (`f.Add(0, 0)`, etc.) gives the fuzzer known starting points; `f.Fuzz(...)` then defines the property that must hold for *any* two `int`s the fuzzer generates — here, that addition is commutative. A fuzz run with `-fuzztime` mutates inputs looking for a counterexample; without `-fuzz`, `go test` just runs the seed corpus as regular subtests (as shown in the `FuzzAdd/seed#N` output).

## Common Pitfalls

- **Table-driven test cases that aren't independent.** Since subtests can run in parallel (`t.Parallel()`), each case's test closure must not share mutable state with another case, or the results become nondeterministic.
- **Forgetting `t.Parallel()` needs to be called inside the subtest closure, not just the outer test.** Calling it only in `TestAdd` (not inside each `t.Run` closure) would make `TestAdd` parallel relative to *other* top-level tests, but its subtests would still run sequentially.
- **Fuzzing without a meaningful invariant.** `FuzzAdd` checks commutativity — a fuzz target needs a property that's true for all valid inputs; fuzzing without one (e.g. just calling a function and checking it doesn't panic) is still useful, but far less targeted.
- **Deleting fuzz-discovered failing inputs from `testdata/fuzz/`.** If the fuzzer ever finds a counterexample, it's saved there and replayed on every subsequent `go test` — removing it without first fixing the underlying bug just hides the regression.

## References

- [testing package docs](https://pkg.go.dev/testing)
- [Go Blog — Fuzzing is Beta Ready](https://go.dev/blog/fuzz-beta)
- [Go Wiki — Table Driven Tests](https://go.dev/wiki/TableDrivenTests)

## Next Steps

- [testing](../testing/) — the basics: a single `Test*` function and `TestMain`
- [benchmark](../benchmark/) — `testing.B` for performance measurement instead of correctness
