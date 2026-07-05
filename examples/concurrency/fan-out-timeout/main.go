// Fan-out with per-worker timeout: each worker races its result against a deadline.
// Workers that exceed the timeout contribute a sentinel response instead of blocking.
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const workerTimeout = 300 * time.Millisecond

type result struct {
	id       int
	message  string
	timedOut bool
}

func worker(id int, wg *sync.WaitGroup, out chan<- result) {
	defer wg.Done()

	done := make(chan result, 1)
	go func() {
		delay := time.Duration(rand.Intn(500)) * time.Millisecond //nolint:gosec
		time.Sleep(delay)
		done <- result{id: id, message: fmt.Sprintf("worker %d done in %s", id, delay)}
	}()

	select {
	case r := <-done:
		out <- r
	case <-time.After(workerTimeout):
		out <- result{id: id, timedOut: true, message: fmt.Sprintf("worker %d timed out", id)}
	}
}

func main() {
	const n = 10
	out := make(chan result, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go worker(i, &wg, out)
	}

	wg.Wait()
	close(out)

	timedOut := 0
	for r := range out {
		fmt.Println(r.message)
		if r.timedOut {
			timedOut++
		}
	}
	fmt.Printf("\n%d/%d workers timed out\n", timedOut, n)
}
