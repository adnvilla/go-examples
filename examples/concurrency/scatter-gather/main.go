// Scatter-gather pattern: fan out work to N goroutines, collect all results.
// Unlike worker-pool, each unit of work gets its own goroutine.
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func fetch(id int, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()
	delay := time.Duration(rand.Intn(500)) * time.Millisecond //nolint:gosec
	time.Sleep(delay)
	results <- fmt.Sprintf("result from worker %d (took %s)", id, delay)
}

func main() {
	const n = 10
	results := make(chan string, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go fetch(i, &wg, results)
	}

	wg.Wait()
	close(results)

	for r := range results {
		fmt.Println(r)
	}
}
