# Contributing

## Adding a new example

1. Create a directory under `examples/` with a lowercase, hyphen-separated name:
   ```
   examples/my-topic/
   ```

2. Add a `main.go` (for runnable examples) or a package with `_test.go` (for library examples). Every example must compile and, where possible, include tests.

3. Follow these conventions:
   - First line of `main.go`: a single-line comment explaining what the example demonstrates and **why** the pattern matters.
   - No Spanish text in code, comments, or README entries.
   - Prefer stdlib over external dependencies. Add a dependency only when it's the canonical library for the topic (e.g., `go-redis/v9` for Redis).
   - If the example needs a real service (database, cache, queue), add a service entry to `docker-compose.yml` and document the requirement in the README table.

4. Register the example in the README table under the appropriate section.

5. Run the full CI pipeline locally before opening a PR:
   ```bash
   make ci
   ```

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
- Use `//nolint:<linter>` with a comment explaining **why** only when suppressing a false-positive in legacy code. New examples should not need nolint directives.

## Pull requests

- One PR per logical change or phase.
- Commit messages in English, imperative mood (`add context example`, not `added` or `adds`).
- PR description should include a brief summary and a test plan checklist.
