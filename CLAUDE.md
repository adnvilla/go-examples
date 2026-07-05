# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A collection of standalone Go examples organized by topic (`examples/<topic>/`), each demonstrating one Go feature, pattern, or library integration. There is no shared application — every subdirectory under `examples/` is its own independent `main` package (or a small package with tests), runnable and testable in isolation. Requires Go 1.24+ (module targets Go 1.25.0, module path `github.com/adnvilla/go-examples`).

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

This is a Go multi-module workspace. All commands run from the repo root.

- NEVER chain commands with `cd`. Do not run compound commands like
  `cd examples/foo && go build ...`. Multiple directory changes in one
  command trigger a permission prompt and cannot be persisted to the
  allowlist. Run each step from the repo root instead.
- Use the Makefile targets for validation — never raw chained `go`
  commands. Run them as separate steps, one per line:
  - `make build`
  - `make vet`
  - `make test`
  - `make lint`
  - `make fmt`
- To scope any check to a single example, pass `EXAMPLE=<short-name>`:
  `make build EXAMPLE=http-server` (omit `EXAMPLE` to run across the
  whole workspace). This matches the existing `make run EXAMPLE=` convention.
- Registering a NEW example is a one-time step: `make use EXAMPLE=<name>`
  (wraps `go work use`). It is committed to `go.work` and must NOT be
  repeated on every check.
- `make check EXAMPLE=<name>` runs build + vet + test + lint + fmt for one
  example in a single command.
- NEVER prefix commands with `cd` to the repo root — you are already there.
  Do not write `cd /Users/adnvilla/code/go-examples && ...`. The `cd` is
  redundant and, when combined with `git`, blocks permission persistence
  (git-hook safety guard). Run the command directly from the current directory.

### Validating an example (canonical flow)
New example → `make use EXAMPLE=<name>` (once), then
`make check EXAMPLE=<name>`.

### CI enforces build, tidy, lint, and vulnerabilities — not just build/test

`.github/workflows/build.yml` runs four independent jobs on push/PR to `master` (Go 1.24 + 1.25 matrix for the test job only):
1. **test** — `go build`, `go vet`, `go test -race -count=1 ./...`
2. **tidy** — runs `go mod tidy` and fails if `go.mod`/`go.sum` diverge from the committed version
3. **lint** — `golangci-lint` (v2 config in `.golangci.yml`: `errcheck`, `govet` with all analyzers except `fieldalignment`, `staticcheck`, `unused`, `misspell`, `unconvert`, `gosec`, `bodyclose`, `noctx`, `rowserrcheck`). Several legacy examples (`benchmark`, `mysql`, `reflection-bench`, `dynamodb`, `protobuf`) have per-path `exclude-rules` in `.golangci.yml` for pre-context-API code — don't "fix" those without checking why the exclusion exists first.
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

Start what you need with `docker compose up -d [service]` (or all of them with no service name) before running tests/examples that depend on them; `docker compose down` to tear down. Without the service running, those examples' tests will fail on connection errors, not skip.

## Architecture / conventions across examples

- **One example = one directory = one concern.** Don't add cross-example shared packages — the point of this repo is that each folder is self-contained and can be read/run/copied independently. When adding a new example, create a new `examples/<topic>/` directory rather than extending an existing one, unless it's a direct variant of that topic.
- **Two structural eras coexist.** Existing examples (all of them as of this writing) share the root `go.mod`/`go.sum` and are indexed only in the root `README.md` table. Going forward, **new** examples are standalone Go modules (own `go.mod`/`go.sum`/`Makefile`/full `README.md`, registered in the root `go.work` via `go work use ./examples/<name>`) per the standard in [CONTRIBUTING.md](CONTRIBUTING.md) — don't assume every example shares the root module, and don't migrate a legacy example to the new standard as a drive-by change. `go.work` currently only lists the root module (`use .`); it gains an entry each time a new-standard example is added.
- **Package naming is inconsistent by design** among legacy examples: most are `package main` (directly runnable with `go run`), but a few are named packages meant to be imported/tested as libraries (e.g. `examples/pool` is `package pool`, `examples/benchmark` is `package benchmark`, `examples/typecast` is `package typecast`). Match the existing pattern in a directory rather than assuming `package main`.
- **Generated code exists in-tree**: `examples/wire/wire_gen.go` is generated by `google/wire` from `examples/wire/wire.go` — don't hand-edit `wire_gen.go`; regenerate it if the providers change.
- **README.md is the cross-example index**, not per-example documentation. It has a topic-organized table (Concurrency, Error Handling, HTTP, Language Features, Observability & Performance, Testing, Patterns & Design, Serialization, Cloud & Infrastructure, Standard Library) linking every example with a one-line description. Every example — legacy or new-standard — gets a row here; new-standard examples additionally get their own full `README.md` (see CONTRIBUTING.md for required sections, category, and difficulty tagging). Update the Docker Compose service table too if infra is involved.
- **Full example-authoring standard (one concept per example, error handling, `context.Context` usage, stdlib-first, naming/comments, determinism, PR checklist, etc.) lives in [CONTRIBUTING.md](CONTRIBUTING.md)** — read it before adding or substantially changing an example.
