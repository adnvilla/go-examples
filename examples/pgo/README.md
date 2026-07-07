# Profile-Guided Optimization (PGO)

**Category:** performance
**Difficulty:** Advanced

## Objective

Walk the complete PGO workflow the Go toolchain has shipped since 1.21 and almost nobody exercises deliberately: run a representative workload under the CPU profiler, save the profile as **`default.pgo`** next to `main.go`, and every subsequent `go build`/`run`/`test` silently recompiles with profile-guided decisions (most notably, more aggressive inlining of the call sites the profile proved hot). The binary *verifies* its own PGO status from build info — no guessing from timings — and `make compare` benchmarks the same code with PGO off and on.

## Concepts Covered

- The PGO loop: profile a representative run (`runtime/pprof`) → commit `default.pgo` → the toolchain applies it automatically (`-pgo=auto` is the default)
- `debug.ReadBuildInfo()` and the `-pgo` build setting — the ground truth for "was this binary PGO-built?", printed by the program itself
- `-pgo=off` for baselines and for profiling runs (profile the *un*-optimized binary)
- `default.pgo` as a committed build input — the same generated-artifact-in-tree convention as [wire](../wire/)'s `wire_gen.go` and [protobuf](../protobuf/)'s `.pb.go`, regenerated with `make profile`
- Honest expectations: PGO's published wins are 2–7% on real services; on a microbenchmark the delta can be within noise, so the workflow and the verification are the lesson, not a magic number
- `b.Loop()` (Go 1.24) in the benchmark that `make compare` drives

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
pgo/
├── go.mod
├── main.go        # workload + -cpuprofile flag + PGO self-report
├── main_test.go   # correctness tests + the benchmark compare drives
├── default.pgo    # committed CPU profile (regenerate: make profile)
├── Makefile
└── README.md
```

## How to Run

```bash
make run        # built WITH default.pgo (auto-detected)
make run-nopgo  # same code, PGO explicitly off — compare the first line
make profile    # regenerate default.pgo from a representative run
make compare    # benchmark PGO off vs on (3 rounds each)
```

## Expected Output

`make run` (the `-pgo` path is absolute on your machine; the workload result is deterministic):

```
this binary was built with -pgo=/…/examples/pgo/default.pgo
hottest word: "select" (2536 occurrences)
```

`make run-nopgo`:

```
this binary was built with -pgo=off
hottest word: "select" (2536 occurrences)
```

`make compare` prints `ns/op` for both builds; the numbers are machine-dependent and on a workload this small may sit within run-to-run noise — see the walkthrough for why that's the honest result.

## Code Walkthrough

- The workflow lives in the Makefile, and it's three lines: `make profile` runs the binary with `-pgo=off` (profile the unoptimized code) and `-cpuprofile=default.pgo`; from then on, plain `go build`/`go run`/`go test` in this directory find `default.pgo` and apply it — `-pgo=auto` has been the default since Go 1.21. Nothing else changes: no build tags, no source edits.
- `pgoSetting` is the part worth stealing: it reads the `-pgo` build setting from `debug.ReadBuildInfo()`. Teams that "enabled PGO" without verifying it frequently ship un-PGO'd binaries (profile not found in CI's build directory, wrong `-pgo` flag in a Dockerfile); a binary that *prints its own answer* ends that debate.
- The workload (`corpus` + `hottestWord`) is deliberately shaped for PGO relevance: small hot functions in a tight loop, dominated by map operations and string slicing. It's deterministic (a hand-rolled LCG generates the text; ties break alphabetically), so profile runs, test runs, and benchmark runs all agree on the answer — `"select"`, 2536 times.
- `make compare` runs the same benchmark with `-pgo=off` and with the committed profile, three rounds each. On this microbenchmark the delta is often small or within noise — which is the honest, documented reality: PGO's gains (Google reports 2–7%) come from real programs with deep call graphs, where the profile teaches the compiler which of thousands of call sites deserve inlining budget. A demo that promised 30% would be lying; benchstat over many rounds is how you'd measure a real claim.
- `default.pgo` is committed, mirroring the repo's generated-code convention: build inputs derived by a tool live in-tree so `go build` works from a fresh clone, with a `make` target to regenerate.

## Common Pitfalls

- **Profiling with an unrepresentative workload.** PGO optimizes what the profile says is hot; a profile from unit tests or a synthetic corner case can pessimize production paths. Collect from production (or a faithful load test) — `net/http/pprof` in production services exists for exactly this.
- **Assuming PGO applied.** `-pgo=auto` only finds `default.pgo` in the *main package's* directory. Renamed files, CI checkouts that skip it, or builds from another directory silently produce a normal binary. Verify via build info, as this example does.
- **Profiling the PGO-built binary and re-committing forever.** Iterative PGO is supported and stable, but each profile should come from a build you understand; profile the baseline (`-pgo=off`, as `make profile` does) when in doubt.
- **Expecting microbenchmark-visible wins.** The gains are aggregate and workload-dependent. Judge PGO on service-level metrics (CPU/QPS) with `benchstat`-grade statistics, not on one `ns/op` line.
- **Letting `default.pgo` go stale.** A profile from last year's code still "works" (PGO tolerates drift) but optimizes yesterday's hot paths. Regenerate it on a cadence, like any other generated artifact.

## References

- [Go docs — Profile-guided optimization](https://go.dev/doc/pgo)
- [Go Blog — Profile-guided optimization in Go 1.21](https://go.dev/blog/pgo)
- [runtime/pprof package docs](https://pkg.go.dev/runtime/pprof)

## Next Steps

- [profiling](../profiling/) — collecting the profiles this workflow consumes
- [benchmark](../benchmark/) — the `testing.B` machinery `make compare` drives
- [reflection-bench](../reflection-bench/) / [typecast](../typecast/) — other lenses on what the compiler does with your code
