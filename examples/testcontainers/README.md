# Testcontainers

**Category:** testing
**Difficulty:** Advanced

## Objective

Show integration tests that **own their infrastructure**: each test starts a throwaway PostgreSQL container with `testcontainers-go`, waits for real readiness, runs real SQL against it, and cleans everything up — including the safety net (Ryuk) that reaps containers even if the test process dies. This is the alternative to the pattern used elsewhere in this repo (docker-compose service + `DYNAMODB_LOCAL=1`-style env guards): no external setup, no shared state, and the tests run anywhere a Docker daemon exists — including this repo's CI, making this the first example whose integration tests actually execute in the pipeline.

## Concepts Covered

- `tcpostgres.Run` — the Postgres module: image, credentials, and a wait strategy in one call
- Wait strategies: `wait.ForLog(...).WithOccurrence(2)` — why "the port is open" isn't "the database is ready", and why this image logs readiness twice
- `testcontainers.CleanupContainer(t, ...)` + `t.Cleanup` — teardown tied to the test's own lifecycle
- `testcontainers.SkipIfProviderIsNotHealthy(t)` — skip (not fail) on machines without Docker
- Container-per-test isolation, proven by a test that asserts emptiness while another test writes in parallel
- The trade-off versus compose+env-guard: hermetic and CI-friendly, at the cost of a few seconds of startup per test
- `pgx` as a `database/sql` driver (`stdlib`), and the `rows.Next/Scan/Err` discipline from [sqlite](../sqlite/) reused against Postgres

## Prerequisites

- Go 1.25+
- A running Docker daemon (Docker Desktop, or the Docker engine on CI runners). No compose services, no env vars — the tests skip cleanly if Docker is absent.
- Dependencies justified: `testcontainers-go` (+ its Postgres module) is the topic; `pgx/v5` provides the `database/sql` driver

## Project Structure

```
testcontainers/
├── go.mod
├── main.go        # NoteStore — the repository under test
├── main_test.go   # the actual demonstration
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

Abridged — testcontainers logs container lifecycle events around the results:

```
🐳 Creating container for image postgres:16-alpine
🐳 Creating container for image testcontainers/ryuk:0.14.0
⏳ Waiting for container ... Waiting for: log message "database system is ready to accept connections" (occurrence: 2)
🔔 Container is ready: ...
--- PASS: TestAddAndList (3.96s)
--- PASS: TestEachTestGetsAFreshDatabase (4.00s)
PASS
```

## Code Walkthrough

- `startPostgres` is the whole pattern in one helper: skip if Docker is unhealthy, `tcpostgres.Run` with image + credentials + wait strategy, `CleanupContainer` for teardown, `ConnectionString` (host and port are dynamic — the container maps to a random free port, so nothing collides), then open `database/sql` over the `pgx` stdlib driver and run the schema. Everything a test needs, nothing shared between tests.
- The wait strategy is the part people get wrong: Postgres's entrypoint starts the server twice (once for `initdb`, once for real), and both print `database system is ready to accept connections`. Waiting for the *second* occurrence is the documented readiness signal — waiting for the first (or just the TCP port) yields "connection refused" flakes that look like driver bugs.
- `TestEachTestGetsAFreshDatabase` runs in parallel with `TestAddAndList`, which inserts rows at the same time — and still must see zero notes. That assertion is the isolation property, demonstrated rather than claimed: no shared database, no truncate-between-tests fixtures, no ordering sensitivity.
- The first log lines show Ryuk (`testcontainers/ryuk`) starting alongside the test containers: it watches the session and force-reaps anything left behind if the test process crashes — the answer to "what if `t.Cleanup` never runs".
- `NoteStore` itself is deliberately plain (`Init`/`Add`/`List`, Postgres `RETURNING`, the `rows.Err` discipline). The example's subject is the test harness; the code under test only needs to be real enough to prove the database is.

## Common Pitfalls

- **Waiting on the port instead of readiness.** Port-open happens before Postgres finishes `initdb`; log- or query-based wait strategies exist because "listening" and "ready" are different states.
- **One shared container mutated by every test.** It's faster, but tests now depend on cleanup discipline and ordering. If startup cost hurts, prefer one container per *package* (via `TestMain`) with per-test schemas or truncation — and know what you traded away.
- **Hardcoding the mapped port.** The whole point of the random host port is parallel safety; always build the DSN from `ConnectionString`/`MappedPort`.
- **Failing instead of skipping without Docker.** `SkipIfProviderIsNotHealthy` keeps the suite honest on laptops without a daemon while still running fully on CI.
- **Forgetting these are real integration tests in CI.** They pull images and take seconds; keep them in modules where that cost is the point (like this one), and keep unit tests ([mocking](../mocking/), [httptest](../httptest/)) carrying the fast bulk of the pyramid.

## References

- [Testcontainers for Go — documentation](https://golang.testcontainers.org/)
- [Testcontainers for Go — PostgreSQL module](https://golang.testcontainers.org/modules/postgres/)
- [Testcontainers — Ryuk (resource reaper)](https://github.com/testcontainers/moby-ryuk)

## Next Steps

- [postgres](../postgres/) — the compose+env-guard pattern this example is the alternative to, and deeper Postgres content
- [mocking](../mocking/) — the other end of the test-double spectrum; most suites need both
- [testing-patterns](../testing-patterns/) — the `t.Parallel`/subtest machinery these tests build on
