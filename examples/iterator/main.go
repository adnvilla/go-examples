// Range-over-func iterators (Go 1.23): functions with signature func(yield func(V) bool)
// can be used directly in a for-range loop.
// This lets you build lazy iterators without materialising a full slice.
package main

import (
	"fmt"
	"iter"
	"slices"
)

// Filter returns an iterator that yields only elements satisfying pred.
func Filter[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if pred(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Map transforms each element of seq using f.
func Map[V, W any](seq iter.Seq[V], f func(V) W) iter.Seq[W] {
	return func(yield func(W) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Take yields at most n elements from seq.
func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		count := 0
		for v := range seq {
			if count >= n {
				return
			}
			if !yield(v) {
				return
			}
			count++
		}
	}
}

// Naturals is an infinite iterator of integers starting at start.
func Naturals(start int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := start; ; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func main() {
	// Collect the first 5 even numbers from the infinite natural sequence.
	evens := Filter(Naturals(1), func(n int) bool { return n%2 == 0 })
	first5 := slices.Collect(Take(evens, 5))
	fmt.Println("first 5 evens:", first5)

	// Chain: squares of odd numbers, take 4.
	odds := Filter(Naturals(1), func(n int) bool { return n%2 != 0 })
	squares := Map(odds, func(n int) int { return n * n })
	result := slices.Collect(Take(squares, 4))
	fmt.Println("squares of first 4 odds:", result)

	// for-range directly over an iterator — no intermediate slice.
	fmt.Print("inline range: ")
	for n := range Take(Naturals(10), 5) {
		fmt.Print(n, " ")
	}
	fmt.Println()
}
