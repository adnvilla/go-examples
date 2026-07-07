// Demonstrates Profile-Guided Optimization (PGO): collect a CPU profile from
// a representative run, save it as default.pgo next to main.go, and the Go
// toolchain automatically recompiles the hot paths with better decisions
// (notably more aggressive inlining of hot call sites). The binary itself
// reports whether it was built with PGO — the workflow is the lesson; the
// speedup depends on the workload and machine.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cpuprofile := flag.String("cpuprofile", "", "write a CPU profile to this file (used by `make profile`)")
	flag.Parse()

	fmt.Printf("this binary was built with -pgo=%s\n", pgoSetting())

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			return fmt.Errorf("creating profile file: %w", err)
		}
		defer f.Close() //nolint:errcheck
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("collecting CPU profile into %s\n", *cpuprofile)
	}

	// Run the workload enough times for the profiler to see where the time
	// goes; the result is deterministic, so runs are comparable.
	word, count := "", 0
	for range 50 {
		word, count = hottestWord(corpus())
	}
	fmt.Printf("hottest word: %q (%d occurrences)\n", word, count)
	return nil
}

// pgoSetting reads the -pgo build setting stamped into the binary — the
// verifiable answer to "did PGO actually apply?", no guessing from timings.
func pgoSetting() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "-pgo" {
			return s.Value
		}
	}
	return "off"
}

// ── the workload ─────────────────────────────────────────────────────────────
// Small, hot functions called in a tight loop: exactly the shape where PGO's
// profile-driven inlining decisions pay off.

// corpus deterministically generates ~200KB of synthetic text from a fixed
// vocabulary using a hand-rolled LCG (no seeds to get wrong, no lint debates).
func corpus() string {
	vocabulary := []string{
		"context", "channel", "goroutine", "interface", "slice", "map",
		"struct", "error", "defer", "select", "mutex", "atomic",
	}
	var sb strings.Builder
	sb.Grow(220_000)
	state := uint32(42)
	for range 30_000 {
		state = state*1664525 + 1013904223 // LCG: deterministic "randomness"
		sb.WriteString(vocabulary[int(state)%len(vocabulary)])
		sb.WriteByte(' ')
	}
	return sb.String()
}

// hottestWord counts word frequencies and returns the most common word,
// breaking ties alphabetically so the result is stable.
func hottestWord(text string) (string, int) {
	counts := make(map[string]int, 16)
	start := -1
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != ' ' {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			counts[text[start:i]]++
			start = -1
		}
	}

	bestWord, bestCount := "", 0
	for word, count := range counts {
		if count > bestCount || (count == bestCount && word < bestWord) {
			bestWord, bestCount = word, count
		}
	}
	return bestWord, bestCount
}
