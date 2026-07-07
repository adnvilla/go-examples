# Config

**Category:** basics
**Difficulty:** Beginner

## Objective

Show the minimal pattern for reading a JSON configuration file from disk into a typed Go struct using `encoding/json`.

## Concepts Covered

- Defining a struct that mirrors a JSON document's shape
- `os.Open` + `json.NewDecoder(...).Decode(...)` to stream-parse a file (as opposed to reading it all into memory first with `json.Unmarshal`)
- Handling a missing/unreadable file vs. a malformed JSON document as two distinct error cases

## Prerequisites

- Go 1.25+
- No external services; reads `config.json` from its own directory

## Project Structure

```
config/
├── config.json
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

`config.json` is read relative to the working directory, so run it from inside `examples/config/` (both `make run` and `go run .` do this correctly).

## Expected Output

```
[UserA UserB]
NameUser
map[NameUser:asdadas]
```

## Code Walkthrough

- `Configuration` declares exported fields matching the JSON keys in `config.json` (`Users`, `Groups`, `Name`, `ConnectionStrings`) — `encoding/json` matches JSON object keys to struct fields case-insensitively by default.
- `os.Open("config.json")` opens the file; if it doesn't exist or can't be read, the error is printed and `main` returns early instead of proceeding with a zero-value `Configuration`.
- `json.NewDecoder(file).Decode(&configuration)` streams the file's bytes directly into the struct, rather than loading the whole file into a `[]byte` first and calling `json.Unmarshal` — preferable for larger files or streams.
- The three `fmt.Println` calls print a subset of the decoded fields to confirm the parse succeeded; `Groups` is decoded but not printed, showing that unused fields aren't an error.

## Common Pitfalls

- **Relative file paths break when the binary runs from a different directory.** `os.Open("config.json")` only finds the file if the working directory is `examples/config/` — a compiled binary run from elsewhere would fail. Real applications typically resolve config paths explicitly (flag, env var, or a path relative to the executable) rather than the process's working directory.
- **Ignoring the decode error.** The example prints it but continues — in real code, a decode failure usually means the config is unusable and the program should stop rather than proceed with a partially-populated struct.
- **Struct fields not matching JSON keys.** `encoding/json` only populates fields it can match (case-insensitively, or via a `json:"..."` tag); a typo'd field name silently stays at its zero value instead of erroring.

## References

- [encoding/json package docs](https://pkg.go.dev/encoding/json)
- [Effective Go — JSON](https://go.dev/blog/json)

## Next Steps

- [flags](../flags/) — configure a program from the command line instead of a file
- [functional-options](../functional-options/) — configure a struct programmatically without exporting fields
- [embed](../embed/) — compile a config/static file into the binary instead of reading it from disk at runtime
