# Embed

**Category:** basics
**Difficulty:** Beginner

## Objective

Show the three shapes of `//go:embed`: a single file as a `string`, a single file as `[]byte`, and a whole directory as an `embed.FS` — all compiled directly into the binary, with no filesystem access needed at runtime.

## Concepts Covered

- `//go:embed` directives on package-level `string`, `[]byte`, and `embed.FS` variables
- `embed.FS` implementing `io/fs.FS`, so it works with `fs.WalkDir`, `ReadFile`, and any `io/fs`-aware API
- Decoding embedded JSON bytes with `encoding/json`
- Why embedded assets remove a runtime dependency on the working directory (contrast with [config](../config/), which reads `config.json` from disk)

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
embed/
├── go.mod
├── main.go
├── static/
│   ├── config.json
│   └── hello.txt
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

The two feature-flag lines print in random order — Go intentionally randomizes map iteration — everything else is fixed:

```
=== embedded string ===
Hello from an embedded file!
This file is compiled into the binary at build time.
=== embedded JSON ===
version: 1.0.0
  new_ui: true
  dark_mode: false
=== embedded directory ===
static/config.json (94 bytes): {
  "version": "1.0.0",
  "feature_flags
static/hello.txt (82 bytes): Hello from an embedded file!
This file i
```

## Code Walkthrough

- `//go:embed static/hello.txt` above a `string` variable embeds that one file's contents directly as text.
- `//go:embed static/config.json` above a `[]byte` variable embeds the raw bytes, which are then decoded with `json.Unmarshal` into `appConfig`.
- `//go:embed static` above an `embed.FS` variable embeds the entire directory tree; `fs.WalkDir` walks it exactly like it would a real directory on disk, and `staticFS.ReadFile(path)` reads an individual embedded file's bytes.
- All three embeds happen at *compile time* — the resulting binary has no dependency on `static/` existing at runtime, unlike [config](../config/)'s `os.Open("config.json")`.

## Common Pitfalls

- **`//go:embed` paths are relative to the source file's directory**, not the working directory at runtime — this is the opposite of `os.Open`, and is what makes the embedded binary portable.
- **The directive must have no blank line between it and the variable declaration.** A blank line silently turns it into an ordinary comment instead of an embed directive (the build then fails because the variable doesn't have the right embedded content, or fails to compile if the type doesn't match).
- **Only `string`, `[]byte`, and `embed.FS` are valid embed targets** — no other types.
- **Map iteration order is randomized by Go on purpose** (`FeatureFlags` here) — never rely on it for output order; sort keys explicitly if deterministic output is required.

## References

- [embed package docs](https://pkg.go.dev/embed)
- [Go Blog — Go 1.16 embed](https://go.dev/blog/go1.16)

## Next Steps

- [config](../config/) — the same JSON-config idea, but read from disk at runtime instead of embedded
- [http-server](../http-server/) — pair this with `http.FileServerFS(staticFS)` to serve embedded static assets over HTTP
