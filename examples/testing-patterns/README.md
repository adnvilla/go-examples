# Testing Patterns

**Category:** testing
**Difficulty:** Intermediate

## Objective

Show the core Go testing techniques in one place: table-driven tests, named subtests with `t.Run`, running tests in parallel with `t.Parallel`, fuzz testing (Go 1.18+) to check an invariant against generated inputs, and `TestMain` as a package-level setup/teardown hook — illustrated here with a (deliberately flawed) coverage-threshold gate.

## Concepts Covered

- Table-driven tests: one test function, a slice of `{input, want}` cases, one loop
- `t.Run(name, func(t *testing.T) {...})` — named subtests, individually addressable via `go test -run TestAdd/negative`
- `t.Parallel()` at both the top-level test and subtest level, to run independent cases concurrently
- `errors.Is` inside a table-driven test to assert on a specific sentinel error (`ErrDivByZero`)
- `FuzzAdd`: a seed corpus (`f.Add(...)`) plus a property check (`f.Fuzz(func(t *testing.T, a, b int) {...})`) verifying `Add` is commutative for any generated inputs
- `TestMain(m *testing.M)` as a package-level hook that wraps every test run
- `go test -cover` / `testing.CoverMode()` / `testing.Coverage()` — and why gating on the latter inside `TestMain` is fragile (see Common Pitfalls)

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
testing-patterns/
├── go.mod
├── main.go
├── math.go
├── math_test.go
├── testmain_test.go
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
- `TestMain(m *testing.M)` (in `testmain_test.go`) replaces the default test runner for this package: it calls `m.Run()` to actually execute every `Test*` function, then inspects the result. If tests passed (`rc == 0`) *and* the run was invoked with `-cover` (`testing.CoverMode() != ""`), it additionally fails the whole run (`rc = -1`) when coverage is below 80%. `os.Exit(rc)` propagates whatever exit code `TestMain` decided on to the `go test` process itself. In real suites, `TestMain` is where package-wide setup/teardown lives (starting a container, opening a shared connection).

## Common Pitfalls

- **Table-driven test cases that aren't independent.** Since subtests can run in parallel (`t.Parallel()`), each case's test closure must not share mutable state with another case, or the results become nondeterministic.
- **Forgetting `t.Parallel()` needs to be called inside the subtest closure, not just the outer test.** Calling it only in `TestAdd` (not inside each `t.Run` closure) would make `TestAdd` parallel relative to *other* top-level tests, but its subtests would still run sequentially.
- **Fuzzing without a meaningful invariant.** `FuzzAdd` checks commutativity — a fuzz target needs a property that's true for all valid inputs; fuzzing without one (e.g. just calling a function and checking it doesn't panic) is still useful, but far less targeted.
- **Deleting fuzz-discovered failing inputs from `testdata/fuzz/`.** If the fuzzer ever finds a counterexample, it's saved there and replayed on every subsequent `go test` — removing it without first fixing the underlying bug just hides the regression.
- **`testing.Coverage()` inside `TestMain` doesn't necessarily match the percentage `go test -cover` reports.** The two are computed differently, and the `TestMain` figure can come in lower — enough to trip the 80% gate even when the tool-reported number is higher. Don't wire a coverage gate into CI this way without verifying both numbers first.
- **The coverage gate is silently skipped without `-cover`.** `testing.CoverMode()` is only non-empty when tests run with `-cover` — plain `go test`/`go test -race` (what this repo's `make test` runs) never triggers the threshold check at all, giving a false sense of enforcement.
- **`t.Errorf` vs. `t.Fatalf`.** `Errorf` marks the test failed but continues executing the rest of the test function; `Fatalf` stops immediately. Using `Fatalf` after a check that later code depends on (as `TestDivide` does with the error check) avoids acting on invalid state.

## References

- [testing package docs](https://pkg.go.dev/testing)
- [testing package docs — Main](https://pkg.go.dev/testing#hdr-Main)
- [Go Blog — Fuzzing is Beta Ready](https://go.dev/blog/fuzz-beta)
- [Go Blog — The cover story](https://go.dev/blog/cover)
- [Go Wiki — Table Driven Tests](https://go.dev/wiki/TableDrivenTests)

## Next Steps

- [benchmark](../benchmark/) — `testing.B` for performance measurement instead of correctness
