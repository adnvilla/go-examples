# Contributing

## Adding a new example

Every example is a standalone Go module, isolated from every other example — the repo root itself has no Go packages, only `go.work` (listing every example module) and shared tooling. A new example must be copy-pasteable on its own, with its own dependency graph.

1. Create a directory under `examples/` with a lowercase, hyphen-separated name, then init its own module and register it with the workspace:
   ```bash
   mkdir -p examples/my-topic && (cd examples/my-topic && go mod init github.com/adnvilla/go-examples/examples/my-topic)
   make use EXAMPLE=my-topic
   ```

2. One example = one primary concept (e.g. "Worker Pool", "Circuit Breaker", "Graceful Shutdown"). Don't combine unrelated concepts (e.g. Kafka + Postgres + retry + worker pool) into a single example — split into separate directories instead.

3. Structure, whenever applicable:
   ```
   examples/my-topic/
   ├── README.md
   ├── go.mod
   ├── go.sum
   ├── main.go
   ├── Makefile
   ├── internal/     (optional)
   ├── pkg/          (optional)
   ├── testdata/     (optional)
   └── *_test.go
   ```
   The `Makefile` should provide `run`, `test`, `lint`, `vet`, and `fmt` targets so the example is runnable with a single documented command (`make run` or `go run .`) — no undocumented manual setup.

4. Every example needs a `README.md` with these sections, in order:
   ```
   # Title
   ## Objective
   ## Concepts Covered
   ## Prerequisites
   ## Project Structure
   ## How to Run
   ## Expected Output
   ## Code Walkthrough
   ## Common Pitfalls
   ## References
   ## Next Steps
   ```
   Explain **why** the code exists, not just what it does. State the learning objectives explicitly (e.g. "goroutines, channels, context cancellation"), tag a single primary **Category** (basics, concurrency, channels, context, synchronization, networking, http, grpc, database, sql, kafka, messaging, cli, testing, benchmarking, observability, security, performance, design-patterns, architecture) and a **Difficulty** (Beginner / Intermediate / Advanced). Document the expected output so readers can verify the example works. Cite the source when the example implements an established pattern or recommendation (Go Blog, Effective Go, Go Code Review Comments, Go Proverbs, Uber Go Style Guide, Google Go Style Decisions, Go Memory Model, official package docs).

5. Follow these conventions:
   - First line of `main.go`: a single-line comment explaining what the example demonstrates and **why** the pattern matters.
   - No Spanish text in code, comments, or README entries.
   - Idiomatic Go, standard-library first — add an external dependency only when it's the canonical/de-facto library for the topic (e.g. `go-redis/v9` for Redis) or the dependency itself is the thing being taught. Justify every dependency.
   - Handle errors explicitly (`if err != nil { return err }`); no ignored errors, no empty `recover` blocks, no `panic`/`log.Fatal` inside library code — unless panic/recover is the concept the example teaches.
   - `context.Context` as the first parameter for any operation that blocks, does I/O, may be cancelled, or talks to a network/database/external system.
   - Prefer readable code over clever code: avoid premature optimization, reflection, `unsafe`, and metaprogramming unless that is the example's actual topic.
   - Descriptive names (`worker`, `job`, `repository`) over placeholders (`x`, `tmp`, `foo`). Comments explain intent/why, not what the code already says.
   - `log/slog` for structured output, `fmt.Println` for simple demonstrations — keep logging minimal.
   - Tests must be deterministic — no flaky tests, no unseeded randomness or timing-dependent assertions, unless the example is specifically about non-determinism.
   - If the example needs a real service (database, cache, queue), add a service entry to `docker-compose.yml` and document the requirement in the README.

6. Register the example in the root `README.md` learning path, in the table matching its **Difficulty** tag, with its **Category** in the second column — placed where it fits pedagogically within the level, not appended at the end. The root README stays the cross-example discovery index; the per-example `README.md` holds the depth.

7. Run the full CI pipeline locally before opening a PR:
   ```bash
   make ci
   ```

### Pull request checklist

Before opening a PR, verify:
- The example teaches one primary concept.
- It compiles and runs successfully (`go run .` or `make run`).
- Tests pass and are deterministic.
- `README.md` is complete, including the root `README.md` index table entry.
- Expected output is documented.
- Code is idiomatic Go and every dependency is justified.
- `make lint` and `gofmt` pass.
- The example is understandable without external explanation — when in doubt, prefer clarity over cleverness.

## Running examples

```bash
# Run a single example
make run EXAMPLE=context

# Run all tests
make test

# Start infrastructure services
make infra-up
```

## Code style

- `gofmt` / `goimports` formatting is enforced by the pre-commit hook (`githooks/pre-commit`). Install it with:
  ```bash
  # macOS / Linux
  cp githooks/pre-commit .git/hooks/pre-commit
  chmod +x .git/hooks/pre-commit
  ```
- The CI pipeline runs `golangci-lint`. Run `make lint` locally to catch issues before pushing.
- Use `//nolint:<linter>` with a comment explaining **why** only when suppressing a false-positive on pre-existing or intentionally-legacy code (e.g. an example predating `context.Context` support, or using a deprecated SDK on purpose). New examples should not need nolint directives.

## Pull requests

- One PR per logical change or phase.
- Commit messages in English, imperative mood (`add context example`, not `added` or `adds`).
- PR description should include a brief summary and a test plan checklist.
