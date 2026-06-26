# go-examples

A collection of Go examples organized by topic. Requires Go 1.24+.

## Examples

| Directory | Description | Topics |
|-----------|-------------|--------|
| [benchmark](examples/benchmark/) | Benchmarks with `testing.B` | testing, performance |
| [channels](examples/channels/) | Goroutine communication with channels | concurrency |
| [concurrency/worker-pool](examples/concurrency/worker-pool/) | Fixed pool of goroutines consuming a shared job channel | concurrency |
| [concurrency/scatter-gather](examples/concurrency/scatter-gather/) | Fan-out: one goroutine per task, collect all results | concurrency |
| [concurrency/fan-out-timeout](examples/concurrency/fan-out-timeout/) | Fan-out with per-worker deadline via `time.After` | concurrency |
| [config](examples/config/) | Reading JSON configuration with `encoding/json` | stdlib |
| [datetime-parse](examples/datetime-parse/) | Parsing dates in multiple formats | libraries |
| [dynamodb](examples/dynamodb/) | DynamoDB CRUD with AWS SDK v1 | AWS, databases |
| [flags](examples/flags/) | Command-line flags with the `flag` package | stdlib, CLI |
| [gin](examples/gin/) | Minimal HTTP API with Gin | HTTP, frameworks |
| [inject](examples/inject/) | Dependency injection via constructor functions | DI, patterns |
| [lambda](examples/lambda/) | AWS Lambda function in Go | AWS, serverless |
| [metric](examples/metric/) | Sending metrics to Datadog via StatsD | observability |
| [mysql](examples/mysql/) | MySQL connection and queries | databases |
| [oop](examples/oop/) | Methods, interfaces and polymorphism in Go | patterns |
| [oop/composition](examples/oop/composition/) | Struct embedding as Go's alternative to inheritance | patterns |
| [pool](examples/pool/) | Worker pool with per-task error tracking | concurrency, patterns |
| [profiling](examples/profiling/) | CPU/memory profiling with `pkg/profile` | performance |
| [protobuf](examples/protobuf/) | Binary serialization with Protocol Buffers | serialization |
| [recover](examples/recover/) | Panic handling with `recover` | error handling |
| [redis](examples/redis/) | Task queue with Redis and Gin (go-redis v9) | Redis, HTTP |
| [reflection-bench](examples/reflection-bench/) | Benchmarking reflection vs direct types | performance, reflection |
| [serialization](examples/serialization/) | Flexible JSON: field that is either an object or array | serialization |
| [share-memory-by-communicating](examples/share-memory-by-communicating/) | HTTP URL poller using channels — from the Go blog | concurrency |
| [testing](examples/testing/) | Basic unit tests | testing |
| [typecast](examples/typecast/) | Benchmarks: type switch vs type assertion vs interface | performance |
| [wire](examples/wire/) | Compile-time dependency injection with Wire | DI, code generation |

## Running an example

```bash
go run ./examples/channels/
go run ./examples/gin/
go run ./examples/concurrency/worker-pool/
```

## Tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# DynamoDB integration tests (requires local DynamoDB on :8000)
DYNAMODB_LOCAL=1 go test ./examples/dynamodb/...
```

## External service requirements

| Example | Requirement |
|---------|-------------|
| `dynamodb/` | DynamoDB Local or AWS credentials |
| `redis/` | Redis on `localhost:6379` |
| `mysql/` | Accessible MySQL instance |
| `metric/` | StatsD listener on `localhost:8125` |
| `protobuf/` | `protoc` + `protoc-gen-go` to regenerate `.pb.go` |
