// Context propagation: cancellation, deadlines, and value passing.
// context.Context is the standard mechanism for controlling goroutine lifetimes
// and threading request-scoped values across API boundaries.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	cancelExample()
	timeoutExample()
	valueExample()
}

// cancelExample: parent cancels work before it finishes.
func cancelExample() {
	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan string, 1)
	go func() {
		if err := slowOp(ctx, 500*time.Millisecond, result); err != nil {
			result <- fmt.Sprintf("cancelled: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	cancel() // signal cancellation before slowOp finishes

	fmt.Println("cancelExample:", <-result)
}

// timeoutExample: operation races against an automatic deadline.
func timeoutExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := make(chan string, 1)
	go func() {
		if err := slowOp(ctx, 500*time.Millisecond, result); err != nil {
			result <- fmt.Sprintf("timeout: %v", err)
		}
	}()

	fmt.Println("timeoutExample:", <-result)
}

// slowOp simulates work that respects ctx cancellation.
func slowOp(ctx context.Context, d time.Duration, out chan<- string) error {
	select {
	case <-time.After(d):
		out <- "completed"
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// valueExample: thread a request ID through a call chain without changing signatures.
// Use context values sparingly — only for request-scoped data, not function parameters.
type ctxKey string

const requestIDKey ctxKey = "requestID"

func valueExample() {
	ctx := context.WithValue(context.Background(), requestIDKey, "req-abc-123")
	processRequest(ctx)
}

func processRequest(ctx context.Context) {
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		fmt.Println("valueExample: no request ID in context")
		return
	}

	// Cancellation propagates: a child context inherits the parent's deadline.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := fetchData(ctx, id); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("valueExample: [%s] deadline exceeded\n", id)
			return
		}
		fmt.Printf("valueExample: [%s] error: %v\n", id, err)
		return
	}
	fmt.Printf("valueExample: [%s] done\n", id)
}

func fetchData(ctx context.Context, id string) error {
	select {
	case <-time.After(50 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
