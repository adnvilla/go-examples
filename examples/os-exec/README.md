# os/exec

**Category:** cli
**Difficulty:** Intermediate

## Objective

Show the five things every program that shells out needs from `os/exec`: capturing a child's output, feeding its stdin, reading its exit code (and telling "ran and failed" apart from "never ran"), controlling its environment, and killing it when a context deadline expires. This is the plumbing behind build tools, git wrappers, and anything that orchestrates other binaries.

## Concepts Covered

- `exec.CommandContext` — always preferred over `exec.Command`: the context is what lets you kill a runaway child
- `Output()` vs `CombinedOutput()` — keeping stderr separate so warnings don't corrupt parsed output
- `cmd.Stdin` accepting any `io.Reader` — piping data into a child
- `*exec.ExitError` via `errors.As` — the exit code lives there, and its absence means the command never started (bad path, permissions)
- `cmd.Env` replacing (not extending) the inherited environment — the `append(os.Environ(), ...)` idiom
- Verifying the kill structurally: the 30-second sleep must die near the 200ms deadline

## Prerequisites

- Go 1.24+
- A Unix-like system (`echo`, `tr`, `sh`, `sleep` on `PATH` — macOS and Linux both qualify; this matches the repo's CI)

## Project Structure

```
os-exec/
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

```
--- Output: run and capture stdout ---
captured: "hello from a child process"

--- Stdin: pipe data into the child ---
tr said: "SHOUT THIS"

--- ExitError: read the child's exit code ---
command failed as expected with exit code 3

--- Env: control what the child sees ---
child saw: "greeting is: hola"

--- CommandContext: kill a runaway child ---
30s sleep killed after the 200ms deadline (context: context deadline exceeded, child: signal: killed)
```

## Code Walkthrough

- `captureOutput` uses `.Output()`, which runs the command to completion and returns stdout only. Reach for `CombinedOutput` when you want everything a human would see, but never when you intend to *parse* the result — one deprecation warning on stderr and your parser breaks.
- `feedStdin` assigns `strings.NewReader` to `cmd.Stdin`; the child (`tr a-z A-Z`) reads it exactly as if it were a terminal pipe. Because `Stdin` is an `io.Reader`, files, network connections, or another command's `StdoutPipe` slot in without ceremony — the same composability shown in [io-readers-writers](../io-readers-writers/).
- `readExitCode` is the error-handling core: `Run()` returning an `*exec.ExitError` means the process started and exited non-zero — `ExitCode()` carries the tool's verdict (here `3`). Any *other* error means the process never ran at all. `errors.As` is what separates the two; treating them the same throws away the difference between "tests failed" and "go not installed".
- `controlEnvironment` demonstrates the `cmd.Env` trap and its idiom in one line: setting `Env` **replaces** the whole environment, so the child would lose `PATH` and everything else unless you start from `os.Environ()` and append your overrides.
- `killOnTimeout` wraps the command in a 200ms context and runs `sleep 30`. When the deadline fires, `CommandContext` sends SIGKILL; the demo asserts the elapsed time proves the kill happened rather than trusting the error message. `context.Cause` reports *why* (deadline exceeded) while the command's own error reports *how* (`signal: killed`).

## Common Pitfalls

- **Building shell strings instead of argument lists.** `exec.Command("sh", "-c", "ls "+userInput)` is command injection. Pass arguments as separate parameters — `exec.Command("ls", userInput)` — and the child gets them verbatim, unparsed. (This example uses `sh -c` only with fixed, constant strings.)
- **`exec.Command` without a context for anything that can hang.** A child blocked on a dead network mount blocks your program forever; `CommandContext` is the same call with an exit strategy.
- **Setting `cmd.Env` to just your variables.** You silently wiped `PATH`, `HOME`, `TMPDIR`... and the failure shows up as "executable not found" somewhere downstream. Always `append(os.Environ(), ...)` unless isolation is the goal.
- **Parsing `CombinedOutput`.** Stderr interleaves with stdout nondeterministically. Parse `Output()`, log stderr (available on the `ExitError.Stderr` field when using `Output`).
- **Forgetting that SIGKILL gives the child no chance to clean up.** `CommandContext`'s default is a hard kill; children that need graceful shutdown want `cmd.Cancel`/`cmd.WaitDelay` (Go 1.20+) to send SIGTERM first and escalate.

## References

- [os/exec package docs](https://pkg.go.dev/os/exec)
- [os/exec docs — CommandContext, Cancel, WaitDelay](https://pkg.go.dev/os/exec#Cmd)
- [Go Blog — Command PATH security in Go](https://go.dev/blog/path-security)

## Next Steps

- [context](../context/) — the deadline machinery that kills the runaway child
- [io-readers-writers](../io-readers-writers/) — the reader/writer seams `cmd.Stdin`/`cmd.Stdout` are built on
- [graceful-shutdown-signals](../graceful-shutdown-signals/) — signals from the receiving side
