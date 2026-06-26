// Synchronisation primitives: sync.Once, sync.Map, and atomic operations.
// Each solves a specific coordination problem; choosing the right one avoids
// unnecessary mutex contention.
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// --- sync.Once: initialise exactly once, even under concurrent access ---

type expensiveService struct{ name string }

var (
	serviceInstance *expensiveService
	once            sync.Once
)

func getService() *expensiveService {
	once.Do(func() {
		// This runs exactly once regardless of how many goroutines call getService.
		serviceInstance = &expensiveService{name: "singleton"}
		fmt.Println("service initialised")
	})
	return serviceInstance
}

func onceExample() {
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := getService()
			_ = s
		}()
	}
	wg.Wait()
	fmt.Println("once: all goroutines got", getService().name)
}

// --- sync.Map: concurrent map without a manual mutex ---
// Optimised for two access patterns:
//   - write-once / read-many (e.g., caches)
//   - many goroutines each writing to disjoint keys

func syncMapExample() {
	var m sync.Map

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			m.Store(key, i*i)
		}(i)
	}
	wg.Wait()

	m.Range(func(k, v any) bool {
		fmt.Printf("syncMap: %s = %v\n", k, v)
		return true
	})
}

// --- atomic: lock-free counter and compare-and-swap ---

func atomicExample() {
	var counter atomic.Int64

	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	fmt.Println("atomic counter:", counter.Load()) // always 1000

	// Compare-and-swap: update only if the current value matches expected.
	var state atomic.Int32
	swapped := state.CompareAndSwap(0, 1) // 0 → 1
	fmt.Println("CAS (0→1):", swapped)
	swapped = state.CompareAndSwap(0, 2) // expected 0 but got 1 — no swap
	fmt.Println("CAS (0→2, should fail):", swapped)
}

func main() {
	fmt.Println("=== sync.Once ===")
	onceExample()

	fmt.Println("\n=== sync.Map ===")
	syncMapExample()

	fmt.Println("\n=== atomic ===")
	atomicExample()
}
