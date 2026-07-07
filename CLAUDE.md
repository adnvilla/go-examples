# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A collection of standalone Go examples organized by topic (`examples/<topic>/`), each demonstrating one Go feature, pattern, or library integration. There is no shared application, and no shared root module either — every example directory is its own independent Go module (own `go.mod`/`go.sum`), most `package main` (runnable via `go run .`), a few named packages meant to be imported/tested as libraries. The repo root itself has no Go packages; it only holds `go.work` (the workspace listing every example module) and shared tooling (`Makefile`, `.golangci.yml`, `docker-compose.yml`). Requires Go 1.25+ (each module targets `go 1.25.0`; CI also tests Go 1.26).

## Commands

```bash
make build                    # go build ./...
make test                     # go test -race -count=1 ./...
make vet                      # go vet ./...
make lint                     # golangci-lint run ./... (requires golangci-lint v2 in PATH)
make vuln                     # govulncheck ./... (requires govulncheck in PATH)
make tidy                     # go mod tidy && go mod verify
make run EXAMPLE=context      # go run ./examples/context/  — run a single example
make ci                       # build + vet + test + lint, i.e. the full local pipeline before a PR
make infra-up / infra-down    # docker compose up -d / down (see Infra table below)
make help                     # list all targets
```

Raw `go` equivalents when you need finer control:

```bash
go test ./examples/<name>/...               # test a single example
go test -run TestName ./examples/<name>/... # run a single test
go test -fuzz=FuzzAdd ./examples/testing-patterns/   # fuzz test (runs until interrupted)
```

## Running commands (IMPORTANT)

Run every command as a SEPARATE step. Do NOT chain commands with `&&`,
`||`, `;`, or `|`. Each command must run on its own so it can be matched
against the permission allowlist independently.

- NEVER chain commands. Run them one per line, as individual tool calls.
  `go mod tidy` then `git diff`, NOT `go mod tidy && git diff`.
- NEVER use `cd`. All commands run from the repo root, where you already
  are. To scope to one example, pass the path as an argument
  (`go build ./examples/foo/...`) or use the Makefile
  (`make check EXAMPLE=foo`), never `cd examples/foo && ...`.
- Chaining anything with `git` (e.g. `... && git diff`) triggers a
  git-hook safety guard that forces manual approval and cannot be
  persisted — another reason to keep commands separate.
- For validation use the Makefile targets, one at a time:
  `make build`, `make vet`, `make test`, `make lint`, `make fmt`
  (append `EXAMPLE=<name>` to scope to one example). The Makefile
  encapsulates any needed `cd` inside its recipes, so you never chain.

### Validating an example (canonical flow)
New example → `make use EXAMPLE=<name>` (once), then
`make check EXAMPLE=<name>`.

### CI enforces build, tidy, lint, and vulnerabilities — not just build/test

`.github/workflows/build.yml` runs four independent jobs on push/PR to `master` (Go 1.25 + 1.26 matrix for the test job only; the other jobs run on 1.26):
1. **test** — `go build`, `go vet`, `go test -race -count=1 ./...`
2. **tidy** — runs `go mod tidy` and fails if `go.mod`/`go.sum` diverge from the committed version
3. **lint** — `golangci-lint` (v2 config in `.golangci.yml`: `errcheck`, `govet` with all analyzers except `fieldalignment`, `staticcheck`, `unused`, `misspell`, `unconvert`, `gosec`, `bodyclose`, `noctx`, `rowserrcheck`). Several examples (`benchmark`, `mysql`, `reflection-bench`, `protobuf`) have per-path `exclude-rules` in `.golangci.yml` for pre-context-API or deprecated-SDK code predating this repo's current conventions — don't "fix" those without checking why the exclusion exists first. These path-based exclude-rules are resolved relative to the repo root regardless of each example's own module boundary, so they keep working even though every example is its own module.
4. **security** — `govulncheck ./...`

Run `make ci` locally before opening a PR; it mirrors the test+lint portion of CI. Formatting (`gofmt`) is additionally enforced locally by a pre-commit hook (`githooks/pre-commit`, wired via `git config core.hooksPath githooks` / `InitHooks.bat`) — CI itself does not check formatting, so always `gofmt -w` changed files before committing.

### Integration tests need Docker Compose services

Some examples talk to a real backing service rather than a mock:

| Service | Port | Used by | Notes |
|---------|------|---------|-------|
| Redis 7 | 6379 | `examples/redis` | |
| MySQL 8 | 3306 | `examples/mysql` | user `root`, password `secret`, db `examples` |
| DynamoDB Local | 8000 | `examples/dynamodb` | tests require `DYNAMODB_LOCAL=1` env var set |
| StatsD | 8125/udp | `examples/metric` | prints received metrics to stdout |
| Jaeger | 16686 (UI), 4318/4317 (OTLP) | `examples/otel` | optional — the example defaults to a stdout exporter and only needs Jaeger when `OTEL_EXPORTER_OTLP_ENDPOINT` is set |

Start what you need with `docker compose up -d [service]` (or all of them with no service name) before running tests/examples that depend on them; `docker compose down` to tear down. Without the service running, those examples' tests will fail on connection errors, not skip.

## Architecture / conventions across examples

- **One example = one directory = one Go module = one concern.** Every directory under `examples/` (including nested ones like `examples/concurrency/worker-pool` and `examples/oop/composition`) has its own `go.mod`/`go.sum`, is listed in the root `go.work`, and can be copy-pasted out of this repo as a self-contained unit. Don't add cross-example shared packages. When adding a new example, create a new `examples/<topic>/` directory (`go mod init github.com/adnvilla/go-examples/examples/<topic>` then `go work use ./examples/<topic>`) rather than extending an existing one, unless it's a direct variant of that topic.
- **The repo root has no Go packages of its own.** `go build/vet/test ./...` run bare from the repo root fail ("no packages to match") — always go through the Makefile (`make build`/`make test`/etc., which loop over every module in `go.work`), or `cd`/path-scope into a specific example's directory. `go.work`'s `use` list has one entry per example module and does **not** include `.` (removed once the last example — `examples/typecast` — was migrated out).
- **Package naming is inconsistent by design**: most examples are `package main` (directly runnable with `go run .`), but a few are named packages meant to be imported/tested as libraries (e.g. `examples/pool` is `package pool`, `examples/benchmark` is `package benchmark`, `examples/typecast` is `package typecast`, `examples/oop/composition` is `package composition`). A library-style example with no `main` uses a Go testable `Example` function (or its `Test*`/`Benchmark*` functions) as its runnable demonstration instead of `go run` — check the example's `Makefile`'s `run` target to see which applies. Match the existing pattern in a directory rather than assuming `package main`.
- **Generated code exists in-tree**: `examples/wire/wire_gen.go` is generated by `google/wire` from `examples/wire/wire.go` (regenerate via `make generate` in that directory); `examples/protobuf/addressbook.pb.go` is generated by `protoc` from `addressbook.proto`. Don't hand-edit either generated file.
- **README.md (root) is the cross-example index**, not per-example documentation. It is organized as a difficulty-ordered learning path — one table per Difficulty level (Beginner, Intermediate, Advanced), each row carrying the example's Category (Concurrency, Error Handling, HTTP, Language Features, Observability & Performance, Testing, Patterns & Design, Serialization, Security, Cloud & Infrastructure, Standard Library) and a one-line description; within a level, rows are sequenced pedagogically, so insert new examples where they fit conceptually, not at the end. Every example additionally has its own full `README.md` (Objective, Concepts Covered, Prerequisites, Project Structure, How to Run, Expected Output, Code Walkthrough, Common Pitfalls, References, Next Steps, plus a Category and Difficulty tag — see CONTRIBUTING.md). Update the Docker Compose service table too if infra is involved.
- **Full example-authoring standard (one concept per example, error handling, `context.Context` usage, stdlib-first, naming/comments, determinism, PR checklist, etc.) lives in [CONTRIBUTING.md](CONTRIBUTING.md)** — read it before adding or substantially changing an example.
