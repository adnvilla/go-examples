// Demonstrates the circuit breaker resilience pattern: after a run of
// consecutive failures the breaker opens and fails fast (protecting both the
// caller's latency and the struggling downstream), then probes cautiously in
// half-open state and only closes again once the downstream proves healthy.
// The full closed -> open -> half-open -> open -> half-open -> closed
// lifecycle is walked deterministically.
package main

import (
	"errors"
	"fmt"
	"time"
)

// flakyService simulates a downstream dependency with an on/off switch.
type flakyService struct {
	healthy bool
	calls   int // how many times the breaker actually let a call through
}

var errUnavailable = errors.New("503 service unavailable")

func (s *flakyService) call() error {
	s.calls++
	if !s.healthy {
		return errUnavailable
	}
	return nil
}

func main() {
	const cooldown = 50 * time.Millisecond

	service := &flakyService{healthy: false}
	breaker := New(3, 2, cooldown) // trip after 3 failures, close after 2 good probes
	breaker.OnStateChange = func(from, to State) {
		fmt.Printf("  >> breaker: %s -> %s\n", from, to)
	}

	attempt := func(label string) {
		err := breaker.Do(service.call)
		switch {
		case err == nil:
			fmt.Printf("%s: ok\n", label)
		case errors.Is(err, ErrOpen):
			fmt.Printf("%s: short-circuited (%v)\n", label, err)
		default:
			fmt.Printf("%s: downstream error (%v)\n", label, err)
		}
	}

	fmt.Println("--- closed: three consecutive failures trip the breaker ---")
	attempt("call 1")
	attempt("call 2")
	attempt("call 3")

	fmt.Println("\n--- open: calls fail fast, the downstream is left alone ---")
	before := service.calls
	attempt("call 4")
	attempt("call 5")
	fmt.Printf("downstream calls during open state: %d\n", service.calls-before)

	fmt.Println("\n--- half-open probe while still broken: back to open ---")
	time.Sleep(cooldown + 10*time.Millisecond)
	attempt("call 6 (probe)")

	fmt.Println("\n--- half-open probes after recovery: two successes close it ---")
	service.healthy = true
	time.Sleep(cooldown + 10*time.Millisecond)
	attempt("call 7 (probe)")
	attempt("call 8 (probe)")

	fmt.Println("\n--- closed again: traffic flows normally ---")
	attempt("call 9")
	fmt.Printf("final state: %s, total downstream calls: %d (of 9 attempts)\n",
		breaker.State(), service.calls)
}
