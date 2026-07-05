# Errors

**Category:** basics
**Difficulty:** Beginner

## Objective

Show the standard-library error-handling toolkit: sentinel errors, custom error types, wrapping with `%w`, unwrapping with `errors.Is`/`errors.As`, and combining multiple errors with `errors.Join`.

## Concepts Covered

- Sentinel errors (`errors.New`) compared by identity with `errors.Is`
- A custom error type (`*ValidationError`) implementing the `error` interface, retrieved with `errors.As`
- Wrapping an error with `fmt.Errorf("...: %w", err)` so the original is still reachable through the chain
- `errors.Join` (Go 1.20+) to combine multiple independent errors into one, still matchable individually with `errors.Is`

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
errors/
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
is ErrNotFound: true
error: loadProfile: findUser(999): not found

validation error on field "id": must be positive

no error: <nil>

joined: not found
forbidden
is ErrNotFound: true
is ErrForbidden: true
```

## Code Walkthrough

- `findUser` returns one of two error shapes depending on input: a `*ValidationError` for a bad ID, or the wrapped sentinel `ErrNotFound` for an ID that's out of range.
- `loadProfile` wraps whatever `findUser` returns with `fmt.Errorf("loadProfile: %w", err)` — the `%w` verb (as opposed to `%v`) is what keeps the original error reachable via `Unwrap`, even though it's now nested two levels deep (`loadProfile` → `findUser` → `ErrNotFound`).
- `errors.Is(err, ErrNotFound)` walks the whole wrap chain looking for a match by identity — it returns `true` here despite `ErrNotFound` being wrapped twice.
- `errors.As(err, &valErr)` walks the chain looking for an error whose *type* matches `*ValidationError`, and if found, assigns it into `valErr` so its fields (`Field`, `Message`) are accessible.
- `errors.Join(errs...)` combines `ErrNotFound` and `ErrForbidden` into a single error whose `Error()` string concatenates both messages — but `errors.Is` still matches either original sentinel against the joined result.

## Common Pitfalls

- **Wrapping with `%v` instead of `%w`.** `%v` formats the error into a new string with no link back to the original — `errors.Is`/`errors.As` can no longer find it in the chain.
- **Comparing errors with `==` instead of `errors.Is`.** `==` only matches if the error wasn't wrapped at all; `errors.Is` is required once any wrapping is involved.
- **Passing a non-pointer to `errors.As`.** The target must be a pointer to a type implementing `error` (here, `&valErr` where `valErr` is `*ValidationError`) — passing the value itself is a compile-time or runtime error.
- **Defining sentinel errors as unexported variables when callers need to check for them.** `ErrNotFound`/`ErrForbidden` are exported here specifically so calling code can `errors.Is` against them.

## References

- [errors package docs](https://pkg.go.dev/errors)
- [Go Blog — Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Go Code Review Comments — Error Strings](https://go.dev/wiki/CodeReviewComments#error-strings)

## Next Steps

- [recover](../recover/) — handling panics instead of returned errors
- [context](../context/) — `errors.Is(err, context.DeadlineExceeded)` used in a real cancellation scenario
