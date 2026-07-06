// Demonstrates the pipeline concurrency pattern from the Go blog: stages
// connected by channels, upstream close propagating downstream via range,
// fan-in with a WaitGroup, and context cancellation so a consumer that stops
// early doesn't leak producer goroutines. Every stage goroutine is tracked in
// a WaitGroup so the no-leak claim is verified, not assumed.
package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// gen emits the given numbers on its output channel, then closes it. The
// close is what lets downstream stages use plain `range`. The select on
// ctx.Done() is what lets an abandoned stage exit instead of blocking on a
// send forever — the Go blog's `done` channel, in its modern context form.
func gen(ctx context.Context, wg *sync.WaitGroup, nums ...int) <-chan int {
	out := make(chan int)
	wg.Go(func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	})
	return out
}

// square reads from in until it's closed (or the context is canceled) and
// emits each value squared. Same shape as gen: close on the way out,
// select on every send.
func square(ctx context.Context, wg *sync.WaitGroup, in <-chan int) <-chan int {
	out := make(chan int)
	wg.Go(func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	})
	return out
}

// merge fans multiple input channels into one: a forwarding goroutine per
// input, and one more that closes the output when all forwarders finish —
// without that closer, downstream range loops would never terminate.
func merge(ctx context.Context, wg *sync.WaitGroup, ins ...<-chan int) <-chan int {
	out := make(chan int)

	var forwarders sync.WaitGroup
	for _, in := range ins {
		forwarders.Add(1)
		wg.Go(func() {
			defer forwarders.Done()
			for n := range in {
				select {
				case out <- n:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	wg.Go(func() {
		forwarders.Wait()
		close(out)
	})
	return out
}

func main() {
	linearPipeline()
	fanInPipeline()
	earlyCancellation()
}

// linearPipeline chains two stages; the consumer is just a range loop, and
// termination flows from gen's close through square's close.
func linearPipeline() {
	fmt.Println("--- linear pipeline: gen -> square ---")
	ctx := context.Background()
	var wg sync.WaitGroup

	var results []string
	for v := range square(ctx, &wg, gen(ctx, &wg, 1, 2, 3, 4, 5)) {
		results = append(results, fmt.Sprint(v))
	}
	wg.Wait()
	fmt.Println(strings.Join(results, " "))
}

// fanInPipeline runs two square stages reading the same gen channel — the
// work is split between them — and merges their outputs. Fan-in trades away
// ordering for parallelism, so the results are sorted before printing.
func fanInPipeline() {
	fmt.Println("\n--- fan-in: two square stages share the work, merged ---")
	ctx := context.Background()
	var wg sync.WaitGroup

	nums := gen(ctx, &wg, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	merged := merge(ctx, &wg, square(ctx, &wg, nums), square(ctx, &wg, nums))

	var results []int
	for v := range merged {
		results = append(results, v)
	}
	wg.Wait()

	slices.Sort(results)
	out := make([]string, len(results))
	for i, v := range results {
		out[i] = fmt.Sprint(v)
	}
	fmt.Println(strings.Join(out, " ") + " (sorted; arrival order varies)")
}

// earlyCancellation is the case the ctx plumbing exists for: the consumer
// takes 3 of 1000 values and walks away. cancel() unblocks the stages stuck
// on sends, and wg.Wait() proves every goroutine exited — no leaks.
func earlyCancellation() {
	fmt.Println("\n--- early cancellation: take 3 of 1000, then cancel ---")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	big := make([]int, 1000)
	for i := range big {
		big[i] = i + 1
	}
	squares := square(ctx, &wg, gen(ctx, &wg, big...))

	var taken []string
	for v := range squares {
		taken = append(taken, fmt.Sprint(v))
		if len(taken) == 3 {
			break // consumer bails out mid-stream
		}
	}
	cancel() // tell upstream stages to stop; without this, they'd block forever
	wg.Wait()

	fmt.Println("took: " + strings.Join(taken, " "))
	fmt.Println("all stage goroutines exited — no leaks")
}
