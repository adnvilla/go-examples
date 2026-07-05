# Testing

**Category:** testing
**Difficulty:** Beginner

## Objective

Show the basic shape of a Go unit test — a `_test.go` file, a `Test*` function using `*testing.T` — plus `TestMain` as a hook for package-level setup/teardown, illustrated here with a (flawed) coverage-threshold gate. See [testing-patterns](../testing-patterns/) for table-driven tests, subtests, and fuzzing.

## Concepts Covered

- The `_test.go` naming convention and `TestXxx(t *testing.T)` signature
- `t.Errorf` to report a failing assertion without stopping the test immediately
- `go test -cover` / `go test --coverprofile=...` / `go tool cover -html=...` for coverage reports
- `TestMain(m *testing.M)` as a package-level hook that wraps every test run
- Why gating on `testing.Coverage()` inside `TestMain` is fragile (see Common Pitfalls)

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
testing/
├── go.mod
├── main.go
├── main_test.go
└── README.md
```

## How to Run

```bash
make run   # runs the program itself (go run .)
make test  # go test -race -count=1 ./...
# or directly:
go test -v ./...
go test -cover ./...
go test --coverprofile=cover.out ./... && go tool cover -html=cover.out -o coverage.html
```

## Expected Output

`go run .`:
```
10
```

`go test -v ./...` (no `-cover` flag — see Common Pitfalls for why this matters):
```
=== RUN   TestSum
--- PASS: TestSum (0.00s)
PASS
ok  	.../examples/testing	0.005s
```

## Code Walkthrough

- `Sum(x, y int) int` is the function under test — deliberately trivial, so the focus stays on the testing mechanics rather than the code being tested.
- `TestSum` calls `Sum(5, 5)`, asserts the result is `10`, and reports a failure with `t.Errorf` (which marks the test failed but lets it keep running, unlike `t.Fatalf`) if not.
- `TestMain(m *testing.M)` replaces the default test runner for this package: it calls `m.Run()` to actually execute every `Test*` function, then inspects the result. If tests passed (`rc == 0`) *and* the run was invoked with `-cover` (`testing.CoverMode() != ""`), it additionally fails the whole run (`rc = -1`) when coverage is below 80%.
- `os.Exit(rc)` propagates whatever exit code `TestMain` decided on to the `go test` process itself.

## Common Pitfalls

- **`testing.Coverage()` inside `TestMain` doesn't necessarily match the percentage `go test -cover` reports.** Running this example with `go test -cover ./...` prints `coverage: 50.0% of statements` from the standard tool, but `testing.Coverage()` called inside `TestMain` returns a different (lower) fraction here — enough to trip the 80% gate even though the tool-reported figure is higher. Don't assume the two numbers agree; verify before wiring a coverage gate into CI this way.
- **The coverage gate is silently skipped without `-cover`.** `testing.CoverMode()` is only non-empty when tests are run with `-cover` — plain `go test`/`go test -race` (what this repo's `make test` runs) never triggers the threshold check at all, so relying on this pattern in ordinary CI runs gives a false sense of enforcement.
- **`t.Errorf` vs. `t.Fatalf`.** `Errorf` marks the test failed but continues executing the rest of the test function; `Fatalf` stops immediately. Using `Fatalf` after a check that later code depends on avoids acting on invalid state.

## References

- [testing package docs](https://pkg.go.dev/testing)
- [Go Blog — The cover story](https://go.dev/blog/cover)
- [testing package docs — Main](https://pkg.go.dev/testing#hdr-Main)

## Next Steps

- [testing-patterns](../testing-patterns/) — table-driven tests, `t.Parallel`, subtests, and fuzz testing
- [benchmark](../benchmark/) — `testing.B` benchmarks, the performance-measurement counterpart to `testing.T`
