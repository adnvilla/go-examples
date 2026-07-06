package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain replaces the default test runner for this package — the place for
// package-level setup/teardown around m.Run(). Here it demonstrates gating on
// coverage, a pattern with real caveats: testing.CoverMode() is only non-empty
// when tests run with -cover, and testing.Coverage() doesn't necessarily match
// the percentage `go test -cover` reports (see README Common Pitfalls).
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	rc := m.Run()

	// rc 0 means we've passed,
	// and CoverMode will be non empty if run with -cover
	if rc == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.8 {
			fmt.Println("Tests passed but coverage failed at", c)
			rc = -1
		}
	}
	os.Exit(rc)
}
