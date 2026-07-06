// Demonstrates rate limiting: a time.Ticker as a fixed-cadence limiter, and
// golang.org/x/time/rate's token bucket for the real thing — Allow for
// shed-load-now decisions and Wait for pacing work without dropping it.
// Rate limiting is how clients respect API quotas and servers survive bursts.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tickerCadence()
	tokenBucketAllow()
	return tokenBucketWait(ctx)
}

// tickerCadence is the simplest limiter: one request per tick. It works, but
// it's rigid — no bursts, one token "capacity", and a stopped consumer still
// has the ticker firing into a buffered channel of size 1.
func tickerCadence() {
	fmt.Println("--- time.Ticker: fixed cadence, one request per tick ---")
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for req := 1; req <= 4; req++ {
		<-ticker.C // block until the next slot opens
		fmt.Printf("request %d processed on its tick\n", req)
	}
}

// tokenBucketAllow shows the load-shedding side of x/time/rate: Allow takes a
// token if one is available and reports the decision immediately — the shape
// used inside servers that answer 429 instead of queueing.
func tokenBucketAllow() {
	fmt.Println("\n--- rate.Limiter Allow: refill 10/s, burst 3 — shed what overflows ---")
	// burst is the bucket size: up to 3 tokens can accumulate, so 3 back-to-
	// back requests succeed even though the steady-state rate is 10/s.
	limiter := rate.NewLimiter(rate.Limit(10), 3)

	for req := 1; req <= 5; req++ {
		if limiter.Allow() {
			fmt.Printf("request %d: allowed (token consumed)\n", req)
		} else {
			fmt.Printf("request %d: rejected — bucket empty, a real server would answer 429\n", req)
		}
	}
}

// tokenBucketWait shows the pacing side: Wait blocks until a token is
// available (or the context ends), so nothing is dropped — the shape used by
// clients staying under a provider's quota.
func tokenBucketWait(ctx context.Context) error {
	fmt.Println("\n--- rate.Limiter Wait: refill 50/s, burst 1 — pace instead of reject ---")
	limiter := rate.NewLimiter(rate.Limit(50), 1)

	const requests = 5
	start := time.Now()
	for req := 1; req <= requests; req++ {
		// Wait is where context integration pays off: cancellation or a
		// deadline aborts the sleep with an error instead of hanging.
		if err := limiter.Wait(ctx); err != nil {
			return fmt.Errorf("waiting for token (request %d): %w", req, err)
		}
		fmt.Printf("request %d sent\n", req)
	}

	// At 50 tokens/s with burst 1, requests 2..5 each wait ~20ms, so the loop
	// cannot legally finish faster than ~80ms. Verify the pacing actually
	// happened rather than printing machine-dependent timings.
	elapsed := time.Since(start)
	minExpected := time.Duration(requests-1) * 20 * time.Millisecond
	if elapsed < minExpected {
		return fmt.Errorf("pacing violated: %d requests in %v (< %v)", requests, elapsed, minExpected)
	}
	fmt.Printf("%d requests paced over at least %v — nothing dropped, nothing early\n", requests, minExpected)
	return nil
}
