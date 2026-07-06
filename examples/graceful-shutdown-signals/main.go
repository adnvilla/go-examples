// Demonstrates graceful shutdown driven by OS signals: signal.NotifyContext
// turns SIGINT/SIGTERM into context cancellation, workers drain in-flight work,
// and main enforces a deadline so a stuck worker can't hang the exit forever.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	numWorkers      = 3
	demoSignalAfter = 300 * time.Millisecond
	shutdownTimeout = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// ctx is canceled on the first SIGINT/SIGTERM. stop() then restores the
	// default signal handling, so a second Ctrl+C kills the process
	// immediately instead of being swallowed by a shutdown that might itself
	// be stuck — always defer it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for id := 1; id <= numWorkers; id++ {
		// WaitGroup.Go (Go 1.25) wraps the Add(1)/go/defer Done() dance.
		wg.Go(func() {
			worker(ctx, id)
		})
	}
	fmt.Printf("main: %d workers running; press Ctrl+C to stop\n", numWorkers)

	// So the demo terminates on its own, simulate the operator's Ctrl+C by
	// sending SIGINT to our own process after a short delay. A real service
	// would simply block on <-ctx.Done() until Kubernetes, systemd, or a
	// human delivers the signal.
	go func() {
		time.Sleep(demoSignalAfter)
		fmt.Println("main: simulating Ctrl+C (sending SIGINT to self)")
		if err := signalSelf(); err != nil {
			fmt.Fprintln(os.Stderr, "demo: could not send SIGINT:", err)
		}
	}()

	<-ctx.Done()
	fmt.Printf("main: shutdown signal received; waiting up to %s for workers\n", shutdownTimeout)

	// wg.Wait has no timeout, so wait for it in a goroutine and race the
	// completion channel against a deadline.
	workersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		fmt.Println("main: all workers stopped cleanly")
		return nil
	case <-time.After(shutdownTimeout):
		return fmt.Errorf("shutdown timed out after %s: exiting with workers still running", shutdownTimeout)
	}
}

// worker processes jobs until its context is canceled, then finishes the job
// in flight before returning — cancellation is a request to stop taking new
// work, not permission to drop what's already started.
func worker(ctx context.Context, id int) {
	fmt.Printf("worker %d: processing jobs\n", id)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("worker %d: shutdown requested, draining current job\n", id)
			time.Sleep(50 * time.Millisecond) // simulate finishing in-flight work
			fmt.Printf("worker %d: stopped\n", id)
			return
		case <-ticker.C:
			// simulate picking up and completing a job
		}
	}
}

// signalSelf delivers SIGINT to this process — the programmatic equivalent of
// the operator pressing Ctrl+C in the terminal.
func signalSelf() error {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return proc.Signal(os.Interrupt)
}
