# go-examples

A collection of standalone Go examples. Requires Go 1.24+.

## Learning path

Every example carries a **Category** and a **Difficulty** tag (Beginner / Intermediate / Advanced — the three levels defined in [CONTRIBUTING.md](CONTRIBUTING.md)) in its own README. The tables below order the whole collection as a learning path: start at Beginner and work down. Within each level, examples are sequenced so each one builds on ideas introduced before it.

### 🟢 Beginner — language fundamentals, stdlib, first tools

| Example | Category | Description |
|---------|----------|-------------|
| [oop](examples/oop/) | Patterns & Design | Methods, interfaces, and polymorphism |
| [oop/composition](examples/oop/composition/) | Patterns & Design | Struct embedding as alternative to inheritance |
| [errors](examples/errors/) | Error Handling | Sentinel errors, custom types, `errors.Is`/`As`, wrapping with `%w`, `errors.Join` |
| [channels](examples/channels/) | Concurrency | Goroutine communication with channels |
| [concurrency/worker-pool](examples/concurrency/worker-pool/) | Concurrency | Fixed pool consuming a shared job channel |
| [concurrency/scatter-gather](examples/concurrency/scatter-gather/) | Concurrency | One goroutine per task, collect all results |
| [functional-options](examples/functional-options/) | Language Features | Configure structs without exporting fields or multiplying constructors |
| [inject](examples/inject/) | Patterns & Design | Constructor dependency injection with interfaces |
| [flags](examples/flags/) | Standard Library | Command-line flags with `flag` |
| [config](examples/config/) | Standard Library | Reading JSON config with `encoding/json` |
| [datetime-parse](examples/datetime-parse/) | Standard Library | Parsing dates in multiple formats |
| [io-readers-writers](examples/io-readers-writers/) | Standard Library | `io.Reader`/`io.Writer` contract and stream combinators (`Copy`, `TeeReader`, `MultiWriter`, `Pipe`) |
| [templates](examples/templates/) | Standard Library | `text/template` vs `html/template`: actions, FuncMap, composition, auto-escaping |
| [embed](examples/embed/) | Language Features | Compile static files into the binary with `//go:embed` |
| [slog](examples/slog/) | Observability & Performance | Structured logging with `log/slog` (Go 1.21) |
| [sqlite](examples/sqlite/) | Cloud & Infrastructure | `database/sql` fundamentals on embedded SQLite (pure Go, no Docker) |
| [mysql](examples/mysql/) | Cloud & Infrastructure | MySQL connection and queries |
| [gin](examples/gin/) | HTTP | Minimal API with Gin framework |
| [lambda](examples/lambda/) | Cloud & Infrastructure | AWS Lambda function |
| [metric](examples/metric/) | Observability & Performance | StatsD metrics to Datadog |
| [profiling](examples/profiling/) | Observability & Performance | CPU/memory profiling with `pkg/profile` |

### 🟡 Intermediate — concurrency patterns, HTTP, testing, tooling

| Example | Category | Description |
|---------|----------|-------------|
| [context](examples/context/) | Language Features | Cancellation, deadlines, and `context.Value` propagation |
| [sync-primitives](examples/sync-primitives/) | Concurrency | `sync.Once`, `sync.Map`, and `atomic` operations |
| [share-memory-by-communicating](examples/share-memory-by-communicating/) | Concurrency | URL poller via channels — from the Go blog |
| [concurrency/pipeline](examples/concurrency/pipeline/) | Concurrency | Staged channels, fan-in `merge`, cancellation without goroutine leaks |
| [concurrency/fan-out-timeout](examples/concurrency/fan-out-timeout/) | Concurrency | Per-worker deadline with `time.After` |
| [concurrency/rate-limiting](examples/concurrency/rate-limiting/) | Concurrency | `time.Ticker` vs `x/time/rate`: `Allow` to shed load, `Wait` to pace |
| [errgroup](examples/errgroup/) | Concurrency | `errgroup.WithContext` vs manual WaitGroup; `SetLimit` for bounded concurrency |
| [pool](examples/pool/) | Concurrency | Generic worker pool with per-task error tracking |
| [graceful-shutdown-signals](examples/graceful-shutdown-signals/) | Concurrency | `signal.NotifyContext`: SIGINT/SIGTERM as context cancellation, drain with a deadline |
| [recover](examples/recover/) | Error Handling | Panic handling with `recover` |
| [generics](examples/generics/) | Language Features | `Map`, `Filter`, `Reduce`, `cmp.Ordered`, `Option[T]` |
| [iterator](examples/iterator/) | Language Features | Range-over-func iterators (`iter.Seq`) — Go 1.23 |
| [os-exec](examples/os-exec/) | Standard Library | Run external processes: capture output, stdin, exit codes, env, kill on timeout |
| [http-server](examples/http-server/) | HTTP | stdlib HTTP server with middleware chain and graceful shutdown |
| [http-client](examples/http-client/) | HTTP | HTTP client with retries, exponential backoff, and context cancellation |
| [testing-patterns](examples/testing-patterns/) | Testing | Table-driven tests, `t.Parallel`, subtests, fuzz testing, `TestMain` |
| [httptest](examples/httptest/) | Testing | Handler tests with `httptest.NewRecorder`, client tests with `httptest.NewServer` |
| [mocking](examples/mocking/) | Testing | Hand-rolled test doubles via interfaces: stub, mock, fake, func adapter |
| [serialization](examples/serialization/) | Serialization | Flexible JSON: field that is either an object or array |
| [protobuf](examples/protobuf/) | Serialization | Binary serialization with Protocol Buffers |
| [grpc](examples/grpc/) | HTTP | Unary + server-streaming RPCs from one `.proto`; status codes; `bufconn` tests |
| [crypto-basics](examples/crypto-basics/) | Security | `sha256`, HMAC + constant-time compare, `crypto/rand` tokens, AES-GCM |
| [benchmark](examples/benchmark/) | Observability & Performance | Benchmarks with `testing.B` |
| [reflection-bench](examples/reflection-bench/) | Observability & Performance | Benchmark: `reflect`-based slice construction vs plain `append`, with optional pprof profiles |
| [typecast](examples/typecast/) | Observability & Performance | Benchmark: dispatch strategies compared (method call, interface, type switch, type assertion) |
| [wire](examples/wire/) | Patterns & Design | Compile-time DI code generation with Wire |
| [otel](examples/otel/) | Observability & Performance | OpenTelemetry tracing: nested spans, attributes, error recording; stdout or Jaeger |

### 🔴 Advanced — integrations against real backing services

| Example | Category | Description |
|---------|----------|-------------|
| [redis](examples/redis/) | Cloud & Infrastructure | Task queue over Redis with Gin (go-redis v9) |
| [dynamodb](examples/dynamodb/) | Cloud & Infrastructure | DynamoDB CRUD with AWS SDK v2 |

---

## Running an example

Every example is its own Go module (see [CONTRIBUTING.md](CONTRIBUTING.md)), so there's no single root module to build against. Use the Makefile:

```bash
make run EXAMPLE=context
make run EXAMPLE=generics
make run EXAMPLE=http-server
make run EXAMPLE=concurrency/worker-pool
```

or `go run .` from inside the example's own directory.

## Tests

```bash
# Every example, in its own module
make test

# A specific example
make test EXAMPLE=testing-patterns
# or, from inside that directory:
go test -race -count=1 ./...

# Fuzz test (runs until interrupted)
go test -fuzz=FuzzAdd ./examples/testing-patterns/

# DynamoDB integration tests (requires the dynamodb service below)
DYNAMODB_LOCAL=1 make test EXAMPLE=dynamodb
```

## Infrastructure (Docker Compose)

Several examples need a real service to connect to. Start all services at once:

```bash
docker compose up -d
```

Or bring up only what you need:

```bash
docker compose up -d redis          # examples/redis
docker compose up -d mysql          # examples/mysql
docker compose up -d dynamodb       # examples/dynamodb
docker compose up -d statsd         # examples/metric
docker compose up -d jaeger         # examples/otel (optional — it runs standalone too)
```

Stop and clean up:

```bash
docker compose down
```

| Service | Port | Used by |
|---------|------|---------|
| Redis 7 | `6379` | `examples/redis` |
| MySQL 8 | `3306` | `examples/mysql` — user `root`, password `secret`, db `examples` |
| DynamoDB Local | `8000` | `examples/dynamodb` — set `DYNAMODB_LOCAL=1` for tests |
| StatsD | `8125/udp` | `examples/metric` |
| Jaeger | `16686` (UI), `4318`/`4317` (OTLP) | `examples/otel` — set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` |
