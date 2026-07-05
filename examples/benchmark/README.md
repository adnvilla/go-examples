# Benchmark: MySQL Connection Pool Tuning

**Category:** benchmarking
**Difficulty:** Intermediate

## Objective

Show `testing.B` benchmarks used to compare configuration choices empirically — here, `database/sql`'s connection-pool settings (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`) against a real MySQL server, rather than guessing at the "right" values.

## Concepts Covered

- `testing.B` and `b.RunParallel` for benchmarking concurrent workloads
- Comparing multiple configurations of the same operation (`Ping`) by writing one `Benchmark*` function per configuration
- `sql.DB` connection-pool tuning: `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`
- Why this file predates `context` in `database/sql`'s API — see Common Pitfalls

## Prerequisites

- Go 1.24+
- A running MySQL instance matching this repo's `docker-compose.yml` (`root`/`secret`@`localhost:3306`/`examples`):
  ```bash
  docker compose up -d mysql
  ```

## Project Structure

```
benchmark/
├── go.mod
├── main.go
├── main_test.go
└── README.md
```

## How to Run

```bash
make bench   # go test -bench=. -benchtime=1x ./...  (requires mysql running — see Prerequisites)
make test    # go test -race -count=1 ./... — passes without MySQL running (see Expected Output)
```

## Expected Output

`make test` (no MySQL required — there are no `Test*` functions in this file, only `Benchmark*`, so `go test` without `-bench` never calls `Ping`):
```
ok  	.../examples/benchmark	1.561s [no tests to run]
```

`make bench` (with MySQL running) prints one line per benchmark, each showing `ns/op` for that connection-pool configuration — actual numbers depend entirely on your MySQL instance and hardware, which is the point: compare relative results across configurations, not absolute numbers across machines. A verified run against the docker-compose MySQL service (`go test -bench=. -benchtime=1x ./...`):

```
BenchmarkMaxOpenConns1-14               	       1	   4925542 ns/op
BenchmarkMaxOpenConns2-14               	       1	   2787292 ns/op
BenchmarkMaxOpenConns5-14               	       1	   2805958 ns/op
BenchmarkMaxOpenConns10-14              	       1	   1891542 ns/op
BenchmarkMaxOpenConnsUnlimited-14       	       1	   1584083 ns/op
BenchmarkMaxIdleConnsNone-14            	       1	   1537791 ns/op
BenchmarkMaxIdleConns1-14               	       1	   1382958 ns/op
BenchmarkMaxIdleConns2-14               	       1	   1510042 ns/op
BenchmarkMaxIdleConns5-14               	       1	   1743291 ns/op
BenchmarkMaxIdleConns10-14              	       1	   1456583 ns/op
BenchmarkConnMaxLifetimeUnlimited-14    	       1	   1264042 ns/op
BenchmarkConnMaxLifetime1000-14         	       1	   1294250 ns/op
BenchmarkConnMaxLifetime500-14          	       1	   1320750 ns/op
BenchmarkConnMaxLifetime200-14          	       1	   1473208 ns/op
BenchmarkConnMaxLifetime100-14          	       1	   1617208 ns/op
BenchmarkBestResults-14                 	       1	   1373583 ns/op
PASS
```

## Code Walkthrough

- `Ping(b, db)` is the operation under test: a single `db.Ping()` round-trip, panicking on error (acceptable in a benchmark helper, not in production code).
- Each `Benchmark*` function opens its own `*sql.DB` (note `sql.Open` doesn't actually connect — it's lazy; the pool is established on first use), applies one pool setting, then benchmarks `Ping` under `b.RunParallel`, which runs the body concurrently across multiple goroutines to simulate concurrent load.
- The naming convention encodes what's being varied: `BenchmarkMaxOpenConns1` through `BenchmarkMaxOpenConnsUnlimited` sweep the max-open-connections setting; `BenchmarkMaxIdleConns*` sweeps idle connections kept warm; `BenchmarkConnMaxLifetime*` sweeps how long a connection can be reused before being recycled.
- `BenchmarkBestResults` combines the specific settings the original author found fastest (unlimited open conns, 10 idle conns, 200ms max lifetime) — a snapshot of a conclusion reached by running the other benchmarks, not a universal recommendation for every MySQL server.

## Common Pitfalls

- **Running `make bench` without MySQL up.** Every benchmark's first `Ping` will fail and the whole `Benchmark*` function panics — start `docker compose up -d mysql` first.
- **This file predates `context.Context` in `database/sql`.** All calls use the context-less API (`db.Ping()` instead of `db.PingContext(ctx)`); it's excluded from `noctx`/`errcheck`/`unused`/`staticcheck` linting in `.golangci.yml` for exactly this reason — don't "fix" it to add context plumbing as a drive-by change, since that would change what's being measured.
- **Treating absolute benchmark numbers as portable.** Connection-pool tuning results are entirely dependent on the MySQL server, network latency, and machine running the benchmark — compare relative results between the `Benchmark*` variants on the *same* run, not the numbers from someone else's machine.
- **Forgetting `defer db.Close()`.** Every benchmark here closes its pool, releasing connections back to MySQL — omitting it across many benchmark runs would exhaust the server's connection limit.

## References

- [testing package docs — Benchmarks](https://pkg.go.dev/testing#hdr-Benchmarks)
- [database/sql package docs](https://pkg.go.dev/database/sql)
- [Go database/sql tutorial — Setting SetMaxOpenConns et al.](http://go-database-sql.org/connection-pool.html)

## Next Steps

- [mysql](../mysql/) — the same MySQL connection, without the benchmark/tuning layer
- [testing-patterns](../testing-patterns/) — table-driven `Test*` functions, the correctness counterpart to these performance benchmarks
