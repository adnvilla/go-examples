# httptest

**Category:** testing
**Difficulty:** Intermediate

## Objective

Show the two halves of `net/http/httptest` and when each applies: `httptest.NewRecorder` unit-tests an `http.Handler` entirely in memory (no ports, no network), while `httptest.NewServer` stands up a real loopback server so *client* code can be tested against the full HTTP stack — including the failure modes a client must survive.

## Concepts Covered

- `httptest.NewRequestWithContext` + `httptest.NewRecorder` + `mux.ServeHTTP` — the in-memory handler-test triad
- Testing through the mux (not the bare handler function), so method matching and `r.PathValue` path parameters are exercised too (Go 1.22 routing)
- `httptest.NewServer` — a real server on a random loopback port, with `ts.URL` and `ts.Client()` replacing hardcoded addresses
- Testing client error paths with throwaway `http.HandlerFunc` servers (404s, 500s, malformed bodies)
- `t.Context()` (Go 1.24) for request contexts in tests, and `t.Parallel()` throughout — httptest servers don't conflict, every test gets its own

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
httptest/
├── go.mod
├── main.go          # UserServer (handler under test) + FetchUser (client under test)
├── main_test.go     # the actual demonstration
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
--- PASS: TestHealthz (0.00s)
--- PASS: TestCreateAndGetUser (0.00s)
--- PASS: TestCreateUserRejectsBadBody (0.00s)
    --- PASS: TestCreateUserRejectsBadBody/not_json (0.00s)
    --- PASS: TestCreateUserRejectsBadBody/empty_name (0.00s)
    --- PASS: TestCreateUserRejectsBadBody/empty_body (0.00s)
--- PASS: TestFetchUserErrors (0.00s)
    --- PASS: TestFetchUserErrors/server_error (0.00s)
    --- PASS: TestFetchUserErrors/not_found (0.00s)
    --- PASS: TestFetchUserErrors/garbage_body (0.00s)
--- PASS: TestFetchUser (0.00s)
PASS
```

## Code Walkthrough

- `UserServer` is a deliberately small but real API: JSON in/out, a path parameter, a 404 path, and a mutex-guarded in-memory store — enough surface for the tests to be representative without drowning the httptest techniques in application logic.
- **Recorder tests** (`TestHealthz`, `TestCreateAndGetUser`, `TestCreateUserRejectsBadBody`): `httptest.NewRequestWithContext` builds an `*http.Request` without a network (carrying `t.Context()`, so the request dies with the test); `httptest.NewRecorder` is an `http.ResponseWriter` that captures status, headers, and body for assertions. Serving through `mux.ServeHTTP` (rather than calling `s.handleGetUser` directly) means the test also covers routing — a request with the wrong method or a missing path parameter fails here, exactly as it would in production.
- **Server tests** (`TestFetchUser`, `TestFetchUserErrors`): `httptest.NewServer` binds a real listener on `127.0.0.1:0` (a random free port), so `FetchUser` — the code under test — runs unmodified against `ts.URL` with `ts.Client()`. This is the tool for testing *clients*; using it to test your own handlers is usually overkill when a recorder suffices.
- `TestFetchUserErrors` swaps in one-line `http.HandlerFunc` servers to force the failures a client must handle: a 404, a 500, and a 200 with a garbage body. Fault injection this way needs no mocking framework — a handler closure *is* the mock.
- Everything runs under `t.Parallel()`: recorders are pure memory, and each `httptest.NewServer` gets its own port, so nothing collides.

## Common Pitfalls

- **`httptest.NewRequestWithContext` vs `http.NewRequestWithContext`.** The httptest variant is for *server-side* tests: it panics on error (fine in tests), needs no absolute URL, and produces a request ready to pass to `ServeHTTP`. Don't use it to build requests for a real client — and don't use the plain `httptest.NewRequest` in linted code; `noctx` (rightly) wants the context-carrying form.
- **Testing the handler function instead of the mux.** Calling `s.handleGetUser(rec, req)` directly skips routing, so `r.PathValue("id")` returns `""` and method mismatches go untested. Serve through the mux you ship.
- **Forgetting `defer ts.Close()`.** Each `httptest.NewServer` holds a listener and goroutines; leaking them across many tests slows the suite and can exhaust file descriptors.
- **Hardcoding ports instead of `ts.URL`.** The whole point of httptest servers is the random port — tests that assume `:8080` conflict with each other and with whatever else is running on the machine.
- **Using `http.DefaultClient` against a TLS test server.** `httptest.NewTLSServer` uses a self-signed certificate; only `ts.Client()` is preconfigured to trust it. This is why `FetchUser` accepts an `*http.Client` instead of creating its own — testability drove the signature.

## References

- [net/http/httptest package docs](https://pkg.go.dev/net/http/httptest)
- [net/http docs — ServeMux routing patterns (Go 1.22)](https://pkg.go.dev/net/http#ServeMux)
- [Go Wiki — TableDrivenTests](https://go.dev/wiki/TableDrivenTests)

## Next Steps

- [http-server](../http-server/) — the production counterpart of the handler side (middleware, graceful shutdown)
- [http-client](../http-client/) — a fuller client (retries, backoff) that these same techniques can test
- [testing-patterns](../testing-patterns/) — the table-driven/subtest/parallel machinery used throughout this suite
