// errgroup runs a group of goroutines and collects the first error.
// It replaces the common WaitGroup + error channel pattern with less boilerplate.
// errgroup.WithContext also cancels the shared context when any goroutine fails,
// so sibling goroutines can stop early.
package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

type result struct {
	id    int
	value string
}

// fetchAll runs N fetches concurrently and returns all results or the first error.
func fetchAll(ctx context.Context, ids []int) ([]result, error) {
	results := make([]result, len(ids))

	g, ctx := errgroup.WithContext(ctx)
	for i, id := range ids {
		i, id := i, id // capture loop variables
		g.Go(func() error {
			r, err := fetch(ctx, id)
			if err != nil {
				return fmt.Errorf("fetch %d: %w", id, err)
			}
			results[i] = r
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func fetch(ctx context.Context, id int) (result, error) {
	select {
	case <-ctx.Done():
		return result{}, ctx.Err()
	case <-time.After(time.Duration(id*10) * time.Millisecond):
		return result{id: id, value: fmt.Sprintf("data-%d", id)}, nil
	}
}

// fetchWithLimit uses errgroup's SetLimit to cap concurrent goroutines.
// Useful when calling a rate-limited downstream service.
func fetchWithLimit(ctx context.Context, ids []int, concurrency int) ([]result, error) {
	results := make([]result, len(ids))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, id := range ids {
		i, id := i, id
		g.Go(func() error {
			r, err := fetch(ctx, id)
			if err != nil {
				return fmt.Errorf("fetch %d: %w", id, err)
			}
			results[i] = r
			return nil
		})
	}

	return results, g.Wait()
}

func main() {
	ctx := context.Background()
	ids := []int{1, 2, 3, 4, 5}

	fmt.Println("--- unbounded concurrency ---")
	results, err := fetchAll(ctx, ids)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		for _, r := range results {
			fmt.Printf("  %d: %s\n", r.id, r.value)
		}
	}

	fmt.Println("--- limited to 2 concurrent ---")
	results, err = fetchWithLimit(ctx, ids, 2)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		for _, r := range results {
			fmt.Printf("  %d: %s\n", r.id, r.value)
		}
	}
}
