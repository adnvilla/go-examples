package main

import (
	"errors"
	"testing"
	"time"
)

// fakeClock lets tests move time forward without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newTestBreaker(clock *fakeClock) *Breaker {
	b := New(3, 2, time.Minute)
	b.now = clock.now
	return b
}

var errBoom = errors.New("boom")

func fail() error    { return errBoom }
func succeed() error { return nil }

func TestTripsAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()
	b := newTestBreaker(&fakeClock{})

	for i := range 3 {
		if err := b.Do(fail); !errors.Is(err, errBoom) {
			t.Fatalf("call %d: err = %v, want errBoom", i+1, err)
		}
	}
	if got := b.State(); got != StateOpen {
		t.Fatalf("state after 3 failures = %v, want open", got)
	}
	if err := b.Do(succeed); !errors.Is(err, ErrOpen) {
		t.Fatalf("call while open: err = %v, want ErrOpen", err)
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	t.Parallel()
	b := newTestBreaker(&fakeClock{})

	_ = b.Do(fail)
	_ = b.Do(fail)
	_ = b.Do(succeed) // resets the consecutive-failure count
	_ = b.Do(fail)
	_ = b.Do(fail)

	if got := b.State(); got != StateClosed {
		t.Fatalf("state = %v, want closed — success must reset the streak", got)
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{}
	b := newTestBreaker(clock)

	for range 3 {
		_ = b.Do(fail)
	}
	clock.advance(2 * time.Minute)

	// Probe runs (downstream still broken) and the breaker re-opens.
	if err := b.Do(fail); !errors.Is(err, errBoom) {
		t.Fatalf("probe err = %v, want errBoom (probe must reach downstream)", err)
	}
	if got := b.State(); got != StateOpen {
		t.Fatalf("state after failed probe = %v, want open", got)
	}
	// And the cooldown restarted: an immediate call is short-circuited.
	if err := b.Do(succeed); !errors.Is(err, ErrOpen) {
		t.Fatalf("call right after re-open: err = %v, want ErrOpen", err)
	}
}

func TestHalfOpenSuccessesClose(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{}
	b := newTestBreaker(clock)

	for range 3 {
		_ = b.Do(fail)
	}
	clock.advance(2 * time.Minute)

	if err := b.Do(succeed); err != nil {
		t.Fatalf("first probe: %v", err)
	}
	if got := b.State(); got != StateHalfOpen {
		t.Fatalf("state after first good probe = %v, want half-open", got)
	}
	if err := b.Do(succeed); err != nil {
		t.Fatalf("second probe: %v", err)
	}
	if got := b.State(); got != StateClosed {
		t.Fatalf("state after two good probes = %v, want closed", got)
	}
}

func TestStateChangesAreObserved(t *testing.T) {
	t.Parallel()
	clock := &fakeClock{}
	b := newTestBreaker(clock)

	var transitions []string
	b.OnStateChange = func(from, to State) {
		transitions = append(transitions, from.String()+"->"+to.String())
	}

	for range 3 {
		_ = b.Do(fail)
	}
	clock.advance(2 * time.Minute)
	_ = b.Do(succeed)
	_ = b.Do(succeed)

	want := []string{"closed->open", "open->half-open", "half-open->closed"}
	if len(transitions) != len(want) {
		t.Fatalf("transitions = %v, want %v", transitions, want)
	}
	for i := range want {
		if transitions[i] != want[i] {
			t.Errorf("transition[%d] = %q, want %q", i, transitions[i], want[i])
		}
	}
}
