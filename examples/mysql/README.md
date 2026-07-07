# MySQL

**Category:** database
**Difficulty:** Beginner

## Objective

Show the basic `database/sql` + `go-sql-driver/mysql` pattern: connect, prepare statements once and reuse them, insert rows, and query a single value back with `Scan`.

## Concepts Covered

- `sql.Open` with the MySQL driver registered via blank import (`_ "github.com/go-sql-driver/mysql"`)
- `db.Prepare` — compiling a statement once and reusing it across many `Exec`/`QueryRow` calls, rather than re-parsing SQL on every call
- `stmtOut.QueryRow(...).Scan(&dest)` — the standard pattern for fetching a single row into Go variables
- Why this file predates `context` in `database/sql`'s API — see Common Pitfalls (same reasoning as [benchmark](../benchmark/))

## Prerequisites

- Go 1.25+
- A running MySQL instance matching this repo's `docker-compose.yml` (`root`/`secret`@`localhost:3306`/`examples`):
  ```bash
  docker compose up -d mysql
  # or, from this directory:
  make infra-up
  ```

## Project Structure

```
mysql/
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make infra-up   # start MySQL
make run
```

## Expected Output

```
The square number of 13 is: 169
The square number of 1 is: 1
```

## Code Walkthrough

- `main` connects to MySQL, then unconditionally `DROP TABLE IF EXISTS` + `CREATE TABLE squareNum (number INT PRIMARY KEY, squareNumber INT)` — this makes the example self-contained and repeatable: it never depends on a table having been created by some other process beforehand, and running it twice in a row doesn't fail on duplicate keys.
- `stmtIns` is a prepared `INSERT` statement, reused inside a loop that inserts 25 rows (`number = i`, `squareNumber = i*i` for `i` in `0..24`) — preparing once and executing many times avoids re-parsing the SQL for every insert.
- `stmtOut` is a prepared `SELECT` statement with one placeholder (`number = ?`); it's reused for both lookups (`13` and `1`) in the same way.
- `stmtOut.QueryRow(13).Scan(&squareNum)` runs the query, expects exactly one result row, and copies its single column into `squareNum` — `Scan` is how `database/sql` converts a driver-level row into Go types.

## Common Pitfalls

- **This file predates `context.Context` in `database/sql`.** All calls use the context-less API (`db.Prepare`/`Exec`/`QueryRow` instead of their `...Context` counterparts); it's excluded from `noctx`/`errcheck` linting in `.golangci.yml` for exactly this reason — don't "fix" it to add context plumbing as a drive-by change without also updating the surrounding pattern intentionally.
- **`Scan` on a query with zero matching rows returns `sql.ErrNoRows`**, not a zero value — always check the error rather than assuming `Scan` populated something.
- **Forgetting `defer stmt.Close()` on prepared statements.** Each prepared statement holds a server-side resource; not closing it (as this example correctly does via `defer`) leaks it for the lifetime of the connection.
- **Panicking on every error, as this example does.** Fine for a minimal demo; real code should return errors and let the caller decide how to handle a failed connection or query, per [errors](../errors/).
- **Running this against a MySQL instance without the `docker-compose` credentials.** The connection string is hardcoded to match this repo's `docker-compose.yml` exactly (`root`/`secret`/`examples` on `localhost:3306`) — pointing it at a different instance requires editing `main.go`.

## References

- [database/sql package docs](https://pkg.go.dev/database/sql)
- [go-sql-driver/mysql GitHub repository](https://github.com/go-sql-driver/mysql)
- [Go database/sql tutorial](http://go-database-sql.org/)

## Next Steps

- [benchmark](../benchmark/) — benchmarking `database/sql` connection-pool settings against this same MySQL service
- [dynamodb](../dynamodb/) — a NoSQL alternative, for comparison
