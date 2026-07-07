package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrOpen is returned by Do when the breaker short-circuits a call: the
// downstream is presumed unhealthy and is not contacted at all.
var ErrOpen = errors.New("circuit breaker is open")

// State is the breaker's position in its lifecycle.
type State int

const (
	// StateClosed lets calls through and counts consecutive failures.
	StateClosed State = iota
	// StateOpen fails fast without calling downstream until the cooldown
	// elapses.
	StateOpen
	// StateHalfOpen lets a limited number of probe calls through to test
	// whether the downstream has recovered.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("State(%d)", int(s))
	}
}

// Breaker is a minimal consecutive-failure circuit breaker.
//
// Closed: calls pass through; failureThreshold consecutive failures trip it
// to open. Open: calls fail fast with ErrOpen; after cooldown the next call
// transitions to half-open. Half-open: up to successThreshold probes run one
// at a time — any failure re-opens, successThreshold successes close.
type Breaker struct {
	failureThreshold int
	successThreshold int
	cooldown         time.Duration

	// now is injected so tests can drive time deterministically.
	now func() time.Time
	// OnStateChange, if set, is called synchronously on every transition
	// (with the breaker's lock held — it must not call back into Breaker).
	OnStateChange func(from, to State)

	mu        sync.Mutex
	state     State
	failures  int // consecutive failures while closed
	successes int // successful probes while half-open
	openedAt  time.Time
	probing   bool // a half-open probe is in flight
}

// New returns a closed Breaker. failureThreshold consecutive failures open
// it; after cooldown, successThreshold successful probes close it again.
func New(failureThreshold, successThreshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldown:         cooldown,
		now:              time.Now,
	}
}

// State reports the breaker's current state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Do runs fn under the breaker's policy. It returns ErrOpen without invoking
// fn when the breaker is open (or a half-open probe is already in flight);
// otherwise it returns fn's error and records the outcome.
func (b *Breaker) Do(fn func() error) error {
	if err := b.allow(); err != nil {
		return err
	}
	err := fn()
	b.record(err == nil)
	return err
}

// allow decides whether a call may proceed, performing the open -> half-open
// transition when the cooldown has elapsed.
func (b *Breaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		if b.now().Sub(b.openedAt) < b.cooldown {
			return ErrOpen
		}
		b.transition(StateHalfOpen)
		b.probing = true
		return nil
	case StateHalfOpen:
		// One probe at a time: concurrent callers fail fast rather than
		// stampeding a barely-recovered downstream.
		if b.probing {
			return ErrOpen
		}
		b.probing = true
		return nil
	default:
		return fmt.Errorf("unknown breaker state %v", b.state)
	}
}

// record folds a call outcome into the state machine.
func (b *Breaker) record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		if success {
			b.failures = 0
			return
		}
		b.failures++
		if b.failures >= b.failureThreshold {
			b.openedAt = b.now()
			b.transition(StateOpen)
		}
	case StateHalfOpen:
		b.probing = false
		if !success {
			// The downstream is still sick: re-open and restart the cooldown.
			b.openedAt = b.now()
			b.transition(StateOpen)
			return
		}
		b.successes++
		if b.successes >= b.successThreshold {
			b.transition(StateClosed)
		}
	case StateOpen:
		// A call that started before the trip finished after it; its outcome
		// no longer changes the decision.
	}
}

// transition switches states, resets counters, and fires the callback.
// Callers must hold b.mu.
func (b *Breaker) transition(to State) {
	from := b.state
	if from == to {
		return
	}
	b.state = to
	b.failures = 0
	b.successes = 0
	if to != StateHalfOpen {
		b.probing = false
	}
	if b.OnStateChange != nil {
		b.OnStateChange(from, to)
	}
}
