# go-examples

A collection of Go examples organized by topic. Requires Go 1.24+.

## Examples

### Concurrency

| Directory | Description |
|-----------|-------------|
| [channels](examples/channels/) | Goroutine communication with channels |
| [concurrency/worker-pool](examples/concurrency/worker-pool/) | Fixed pool consuming a shared job channel |
| [concurrency/scatter-gather](examples/concurrency/scatter-gather/) | One goroutine per task, collect all results |
| [concurrency/fan-out-timeout](examples/concurrency/fan-out-timeout/) | Per-worker deadline with `time.After` |
| [concurrency/pipeline](examples/concurrency/pipeline/) | Staged channels, fan-in `merge`, cancellation without goroutine leaks |
| [concurrency/rate-limiting](examples/concurrency/rate-limiting/) | `time.Ticker` vs `x/time/rate`: `Allow` to shed load, `Wait` to pace |
| [errgroup](examples/errgroup/) | `errgroup.WithContext` vs manual WaitGroup; `SetLimit` for bounded concurrency |
| [pool](examples/pool/) | Generic worker pool with per-task error tracking |
| [graceful-shutdown-signals](examples/graceful-shutdown-signals/) | `signal.NotifyContext`: SIGINT/SIGTERM as context cancellation, drain with a deadline |
| [share-memory-by-communicating](examples/share-memory-by-communicating/) | URL poller via channels — from the Go blog |
| [sync-primitives](examples/sync-primitives/) | `sync.Once`, `sync.Map`, and `atomic` operations |

### Error Handling

| Directory | Description |
|-----------|-------------|
| [errors](examples/errors/) | Sentinel errors, custom types, `errors.Is`/`As`, wrapping with `%w`, `errors.Join` |
| [recover](examples/recover/) | Panic handling with `recover` |

### HTTP

| Directory | Description |
|-----------|-------------|
| [http-server](examples/http-server/) | stdlib HTTP server with middleware chain and graceful shutdown |
| [http-client](examples/http-client/) | HTTP client with retries, exponential backoff, and context cancellation |
| [gin](examples/gin/) | Minimal API with Gin framework |
| [grpc](examples/grpc/) | Unary + server-streaming RPCs from one `.proto`; status codes; `bufconn` tests |
| [redis](examples/redis/) | Task queue over Redis with Gin (go-redis v9) |

### Language Features

| Directory | Description |
|-----------|-------------|
| [context](examples/context/) | Cancellation, deadlines, and `context.Value` propagation |
| [generics](examples/generics/) | `Map`, `Filter`, `Reduce`, `cmp.Ordered`, `Option[T]` |
| [iterator](examples/iterator/) | Range-over-func iterators (`iter.Seq`) — Go 1.23 |
| [embed](examples/embed/) | Compile static files into the binary with `//go:embed` |
| [functional-options](examples/functional-options/) | Configure structs without exporting fields or multiplying constructors |

### Observability & Performance

| Directory | Description |
|-----------|-------------|
| [slog](examples/slog/) | Structured logging with `log/slog` (Go 1.21) |
| [metric](examples/metric/) | StatsD metrics to Datadog |
| [otel](examples/otel/) | OpenTelemetry tracing: nested spans, attributes, error recording; stdout or Jaeger |
| [profiling](examples/profiling/) | CPU/memory profiling with `pkg/profile` |
| [benchmark](examples/benchmark/) | Benchmarks with `testing.B` |
| [reflection-bench](examples/reflection-bench/) | Benchmark: `reflect`-based slice construction vs plain `append`, with optional pprof profiles |
| [typecast](examples/typecast/) | Benchmark: dispatch strategies compared (method call, interface, type switch, type assertion) |

### Testing

| Directory | Description |
|-----------|-------------|
| [testing-patterns](examples/testing-patterns/) | Table-driven tests, `t.Parallel`, subtests, fuzz testing, `TestMain` |
| [httptest](examples/httptest/) | Handler tests with `httptest.NewRecorder`, client tests with `httptest.NewServer` |
| [mocking](examples/mocking/) | Hand-rolled test doubles via interfaces: stub, mock, fake, func adapter |

### Patterns & Design

| Directory | Description |
|-----------|-------------|
| [inject](examples/inject/) | Constructor dependency injection with interfaces |
| [wire](examples/wire/) | Compile-time DI code generation with Wire |
| [oop](examples/oop/) | Methods, interfaces, and polymorphism |
| [oop/composition](examples/oop/composition/) | Struct embedding as alternative to inheritance |

### Serialization

| Directory | Description |
|-----------|-------------|
| [serialization](examples/serialization/) | Flexible JSON: field that is either an object or array |
| [protobuf](examples/protobuf/) | Binary serialization with Protocol Buffers |

### Security

| Directory | Description |
|-----------|-------------|
| [crypto-basics](examples/crypto-basics/) | `sha256`, HMAC + constant-time compare, `crypto/rand` tokens, AES-GCM |

### Cloud & Infrastructure

| Directory | Description |
|-----------|-------------|
| [dynamodb](examples/dynamodb/) | DynamoDB CRUD with AWS SDK v2 |
| [lambda](examples/lambda/) | AWS Lambda function |
| [mysql](examples/mysql/) | MySQL connection and queries |
| [sqlite](examples/sqlite/) | `database/sql` fundamentals on embedded SQLite (pure Go, no Docker) |

### Standard Library

| Directory | Description |
|-----------|-------------|
| [io-readers-writers](examples/io-readers-writers/) | `io.Reader`/`io.Writer` contract and stream combinators (`Copy`, `TeeReader`, `MultiWriter`, `Pipe`) |
| [config](examples/config/) | Reading JSON config with `encoding/json` |
| [datetime-parse](examples/datetime-parse/) | Parsing dates in multiple formats |
| [flags](examples/flags/) | Command-line flags with `flag` |
| [os-exec](examples/os-exec/) | Run external processes: capture output, stdin, exit codes, env, kill on timeout |
| [templates](examples/templates/) | `text/template` vs `html/template`: actions, FuncMap, composition, auto-escaping |

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
