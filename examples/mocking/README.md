# Mocking with Interfaces

**Category:** testing
**Difficulty:** Intermediate

## Objective

Show how far plain Go interfaces go for test doubles — no framework required. A small service with two dependency seams (`UserStore`, `Mailer`) is tested with the three classic double flavors: a **stub** (canned answers), a **mock** (records calls for interaction assertions), and a **fake** (a real implementation with a shortcut), plus the function-adapter trick for one-off inline doubles.

## Concepts Covered

- Consumer-defined interfaces: the service declares the `UserStore`/`Mailer` it *needs*, so implementations (real or test) satisfy it implicitly — the seam that makes substitution free
- Stub vs mock vs fake, and which each is for: output you need vs interaction you assert vs stateful behavior many tests share
- The `http.HandlerFunc` adapter trick (`MailerFunc`) — a closure as a complete test double
- Asserting the *negative* interaction: on lookup failure, the mailer must not have been called
- Error wrapping (`%w`) verified with `errors.Is` through the service boundary
- Constructor injection (`NewWelcomeService(store, mailer)`) as the wiring that makes all of this possible

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
mocking/
├── go.mod
├── main.go          # WelcomeService and its two dependency seams
├── main_test.go     # stub, mock, fake, func-adapter — the actual demonstration
├── Makefile
└── README.md
```

## How to Run

The demonstration lives in the tests (`main.go`'s `main` only points you there):

```bash
make test
# or
go test -race -count=1 -v ./...
```

## Expected Output

Abridged — tests run in parallel, so ordering interleaves:

```
--- PASS: TestWelcomeSendsEmail (0.00s)
--- PASS: TestWelcomeStoreFailureSkipsEmail (0.00s)
--- PASS: TestWelcomeWrapsMailerError (0.00s)
--- PASS: TestWelcomeWithFuncDouble (0.00s)
--- PASS: TestWelcomeWithFake (0.00s)
PASS
```

## Code Walkthrough

- `WelcomeService.Welcome` is deliberately ordinary application code: look up, compose, send, wrap errors. What makes it testable is entirely in the signatures — it depends on two small interfaces it defines itself, injected through the constructor. This is the Go norm: **interfaces live with the consumer**, so the service compiles against exactly the behavior it uses and nothing more.
- `stubUserStore` (a *stub*) returns whatever `user`/`err` the test configured. Value receiver, no state, three lines — this is the workhorse double for "arrange" data.
- `mockMailer` (a *mock*) appends every `Send` to a slice so tests can assert interactions: `TestWelcomeSendsEmail` checks recipient and subject; `TestWelcomeStoreFailureSkipsEmail` checks the more interesting property that **no** email goes out when the lookup fails. Its `err` field also makes it a failure injector (`TestWelcomeWrapsMailerError`).
- `fakeUserStore` (a *fake*) actually works — a map with the same semantics a database adapter would have, including `ErrUserNotFound`. Fakes cost more to write but pay off when many tests need consistent stateful behavior instead of per-test canned answers.
- `MailerFunc` adapts a function to the `Mailer` interface exactly like `http.HandlerFunc` adapts to `http.Handler`. For a single-method interface, that means an inline closure is a full double (`TestWelcomeWithFuncDouble`) — often the lightest option of all.
- Error paths are asserted with `errors.Is` against the sentinel/wrapped error, not string matching — the same wrapping discipline shown in the [errors](../errors/) example.

## Common Pitfalls

- **Defining interfaces next to the implementation and mocking those.** If the interface belongs to the producer, it accretes methods the consumer never uses, and every double must implement all of them. Keep interfaces small and consumer-side; a one- or two-method interface makes hand-rolled doubles trivial.
- **Over-asserting on interactions.** Checking that the mailer was called once with the right recipient is the contract; asserting the exact body string in *every* test couples all of them to copy changes. Assert interactions where the interaction *is* the behavior, values where the value is.
- **Mocking what you don't own.** Wrapping `*sql.DB` or an AWS SDK client in your own small interface (`UserStore`) and mocking *that* is robust; trying to fake the SDK's entire surface is not. Adapt third-party types behind your own seam first.
- **Reaching for a framework by default.** `gomock`/`mockery` (generated mocks) and `testify/mock` earn their keep with large interfaces, many call expectations, or codebase-wide consistency — but they add generation steps and a DSL. For seams like this one, the hand-rolled version is shorter than the framework's setup code.
- **Races in doubles.** If the code under test calls a double from multiple goroutines, the double needs the same synchronization a real implementation would (`mockMailer` guards its slice with a mutex) — otherwise `-race` flags your test double instead of your code.

## References

- [Go Code Review Comments — Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [Google Go Style Decisions — Interfaces](https://google.github.io/styleguide/go/decisions#interfaces)
- [Martin Fowler — Mocks Aren't Stubs](https://martinfowler.com/articles/mocksArentStubs.html)
- [gomock](https://github.com/uber-go/mock) / [mockery](https://github.com/vektra/mockery) — generation-based alternatives for large interfaces

## Next Steps

- [inject](../inject/) — the constructor-injection pattern these seams rely on
- [httptest](../httptest/) — the specialized doubles the stdlib ships for HTTP
- [errors](../errors/) — the wrapping/`errors.Is` discipline the assertions here use
