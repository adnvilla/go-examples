# Datetime Parse

**Category:** standard library
**Difficulty:** Beginner

## Objective

Show `github.com/araddon/dateparse` parsing dozens of real-world date/time formats — RFC formats, US and international date orders, Unix timestamps, and more — without the caller specifying a layout string up front, rendered in a table with `github.com/apcera/termtables`.

## Concepts Covered

- Why Go's standard `time.Parse` requires an exact reference-time layout per format, and what a format-guessing library like `dateparse` buys you when input formats are unpredictable (e.g. user-submitted data, logs from multiple systems)
- `dateparse.ParseLocal` — parses a string into a `time.Time`, applying `time.Local` when the input has no explicit timezone
- `time.LoadLocation` + assigning `time.Local` to change what "local" means for the whole process — set here via a `-timezone` flag
- Ambiguous date orders (`mm/dd/yy` vs `dd/mm/yy`) and why `dateparse` can only guess, not know for certain

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
datetime-parse/
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
go run . -timezone=America/Los_Angeles
```

## Expected Output

A table with one row per example input string and its parsed `time.Time` (`%v` formatting). Abridged (see `main.go`'s `examples` slice for the full ~65-entry list):

```
+-------------------------------------------------------+----------------------------------------+
| Input                                                  | Parsed, and Output as %v                |
+-------------------------------------------------------+----------------------------------------+
| May 8, 2009 5:57:51 PM                                 | 2009-05-08 17:57:51 +0000 UTC            |
| Mon Jan  2 15:04:05 2006                               | 2006-01-02 15:04:05 +0000 UTC            |
| Mon, 02 Jan 2006 15:04:05 -0700                        | 2006-01-02 15:04:05 -0700 -0700          |
| 2014-04-26 17:24:37.3186369                            | 2014-04-26 17:24:37.3186369 +0000 UTC     |
| 20140601                                               | 2014-06-01 00:00:00 +0000 UTC            |
| 1332151919                                             | 2012-03-19 10:11:59 +0000 UTC            |
| 1384216367189                                          | 2013-11-12 00:32:47.189 +0000 UTC        |
+-------------------------------------------------------+----------------------------------------+
```

## Code Walkthrough

- `-timezone` (default `"UTC"`) is parsed via `flag`, then `time.LoadLocation` resolves it and is assigned to the package-level `time.Local` — this changes what "local time" means for every subsequent `dateparse.ParseLocal` call in the program (a global, process-wide effect, not scoped to one call).
- `examples` is a large slice of date/time strings covering RFC 1123/3339-style formats, US (`mm/dd/yy`) and international (`yyyy/mm/dd`) orders, formats with/without timezone abbreviations or offsets, fractional seconds, a Chinese-locale date, and raw Unix timestamps (seconds and milliseconds).
- For each string, `dateparse.ParseLocal(dateExample)` guesses the layout and parses it into a `time.Time` — unlike `time.Parse`, no reference-time layout string is supplied by the caller.
- Every result is added as a row to a `termtables.Table` and rendered once at the end, rather than printed line by line.
- The program `panic`s on any parse failure — acceptable for this demo (every included example is known to parse successfully), but real code handling untrusted input should return the error instead.

## Common Pitfalls

- **Assuming `dateparse` always resolves ambiguous dates correctly.** `03/04/2014` could mean March 4th or April 3rd depending on locale convention — a format-guessing library makes a best-effort choice, and that choice can be wrong for a given input's actual intended meaning. Prefer `time.Parse` with an explicit layout whenever the format is known in advance.
- **Not setting `time.Local` before parsing timezone-less input.** `dateparse.ParseLocal` (as opposed to `dateparse.ParseAny`, which defaults to UTC) uses whatever `time.Local` currently is — this program deliberately sets it from a flag first so the behavior is explicit and reproducible.
- **Panicking on parse errors in production code.** This example panics for brevity since every input is known-good; code parsing untrusted or external input should propagate the `error` `ParseLocal` returns instead.
- **Forgetting that `time.Local` is a global.** Assigning to it (as `main` does) affects every subsequent time operation in the process, not just calls in this file — a library should generally avoid mutating it and instead pass locations explicitly.

## References

- [araddon/dateparse GitHub repository](https://github.com/araddon/dateparse)
- [time package docs](https://pkg.go.dev/time)
- [Go Blog — Time Zones](https://go.dev/blog/timezone)

## Next Steps

- [config](../config/) — parsing structured (JSON) input instead of free-form date strings
- [flags](../flags/) — more on the `flag` package used here for `-timezone`
