# SQLite

**Category:** database
**Difficulty:** Beginner

## Objective

Show the `database/sql` fundamentals — parameterized `Exec`, the `rows.Next`/`Scan`/`Err` discipline, `QueryRow` with `sql.ErrNoRows`, and transactions with commit and rollback — against an embedded SQLite database. The driver (`modernc.org/sqlite`) is pure Go: no CGo, no server, no Docker, so this is the lowest-friction database example in the repo and a contrast to [mysql](../mysql/).

## Concepts Covered

- `sql.Open` is lazy (validate with `PingContext`), and the `*sql.DB` is a connection *pool*, not a connection
- The `:memory:` gotcha: each pool connection gets its own empty in-memory database — `SetMaxOpenConns(1)` makes it behave as one
- `ExecContext` with `?` placeholders — parameterization instead of string concatenation, plus `LastInsertId`
- Multi-row queries: `QueryContext` → `rows.Next` → `rows.Scan` → `rows.Close` → **`rows.Err`** (the step everyone forgets, and the one `rowserrcheck` lints for)
- `QueryRowContext(...).Scan` and branching on `errors.Is(err, sql.ErrNoRows)` as a normal condition
- `BeginTx` / `Commit` / `Rollback` — including proving that a rolled-back insert leaves no trace

## Prerequisites

- Go 1.24+
- No external services, no CGo toolchain, no environment variables — the single dependency is justified because SQLite itself is the topic and `modernc.org/sqlite` is the canonical pure-Go driver

## Project Structure

```
sqlite/
├── go.mod
├── go.sum
├── main.go
├── Makefile
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
--- schema created ---

--- inserts with placeholders ---
inserted id=1 "The Go Programming Language" (2015)
inserted id=2 "Go in Action" (2015)
inserted id=3 "Learning Go" (2021)

--- multi-row query ---
published in 2015: "Go in Action"
published in 2015: "The Go Programming Language"

--- single-row query and ErrNoRows ---
published after 2019: "Learning Go"
published after 2100: none (sql.ErrNoRows)

--- transactions: commit and rollback ---
after commit + rollback: 4 books (the rolled-back insert left no trace)
```

## Code Walkthrough

- The blank import `_ "modernc.org/sqlite"` runs the driver's `init()`, which registers the `"sqlite"` name with `database/sql` — the app code then talks only to the standard interface, which is the whole design of `database/sql`: swap the driver, keep the code.
- `sql.Open` doesn't touch the database; it returns a lazy pool. `db.PingContext` is the idiomatic "fail now, not on first query" check. `SetMaxOpenConns(1)` is load-bearing here: with the `:memory:` DSN, every new pool connection would open its *own* empty database, so a second connection would see no tables at all.
- `insertBooks` uses `?` placeholders so values never get spliced into SQL text — this is the injection defense, and it also sidesteps quoting/escaping. `LastInsertId` returns SQLite's generated `AUTOINCREMENT` key.
- `queryBooks` walks the full result-set discipline. The subtle part is `rows.Err()` after the loop: `rows.Next()` returns `false` both at end-of-results *and* when the connection dies mid-iteration — only `rows.Err()` tells you which. (This repo's lint config enables `rowserrcheck` precisely for this.)
- `querySingleBook` shows `QueryRow`'s contract: errors are deferred to `Scan`, and an empty result is `sql.ErrNoRows` — a value to branch on with `errors.Is`, not a failure to propagate blindly.
- `transactions` runs two `BeginTx` blocks: one commits, one rolls back after a successful insert. The final `COUNT(*)` of 4 (3 seeds + 1 committed) demonstrates atomicity observably. Note the `_ = tx.Rollback()` on error paths — once a transaction failed, the rollback error adds nothing actionable.

## Common Pitfalls

- **Treating `*sql.DB` as a single connection.** It's a pool. Session state (temp tables, `PRAGMA`s, `:memory:` contents) set on "the connection" may land on a different one next query. Use `SetMaxOpenConns(1)`, a `*sql.Conn`, or a transaction when you need connection affinity.
- **Skipping `rows.Err()`.** The `for rows.Next()` loop exits silently on a broken connection; without the `Err` check you'll report partial results as complete ones.
- **Forgetting `defer rows.Close()`.** Each open result set pins a pool connection; leak enough of them and every query blocks waiting for a free connection.
- **Building SQL with `fmt.Sprintf`.** Placeholders exist for both safety (injection) and correctness (quoting). Note the placeholder syntax is driver-specific: `?` for SQLite/MySQL, `$1` for Postgres.
- **Assuming SQLite handles concurrent writers like a server database.** SQLite serializes writes; under write contention you'll see `SQLITE_BUSY`-style errors, and pure-Go vs CGo drivers differ in how they surface it. For write-heavy concurrent loads, use a server database — that's a design boundary, not a driver bug.

## References

- [Go — Accessing a relational database (tutorial)](https://go.dev/doc/tutorial/database-access)
- [database/sql package docs](https://pkg.go.dev/database/sql)
- [modernc.org/sqlite docs](https://pkg.go.dev/modernc.org/sqlite)
- [Go Wiki — SQLInterface](https://go.dev/wiki/SQLInterface)

## Next Steps

- [mysql](../mysql/) — the same `database/sql` interface against a real server (Docker Compose)
- [errors](../errors/) — the `errors.Is` machinery used for `sql.ErrNoRows`
- [context](../context/) — why every query here takes a `ctx`
