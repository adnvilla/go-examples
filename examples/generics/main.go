// Generics (Go 1.18+): type parameters eliminate repetitive code for collections.
// Constraints define the set of types a type parameter accepts.
package main

import (
	"cmp"
	"fmt"
	"slices"
)

// --- generic collection functions ---

// Map transforms each element of a slice using f.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Filter returns elements for which keep returns true.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce folds a slice into a single value using f, starting from initial.
func Reduce[T, U any](s []T, initial U, f func(U, T) U) U {
	acc := initial
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// --- constrained generics ---

// Min returns the smallest element in a non-empty slice.
// cmp.Ordered covers all built-in ordered types (integers, floats, strings).
func Min[T cmp.Ordered](s []T) T {
	if len(s) == 0 {
		panic("Min called on empty slice")
	}
	return slices.Min(s)
}

// Keys returns the keys of a map in unspecified order.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- generic optional type ---

// Option[T] avoids nil pointer pitfalls by making absence explicit.
type Option[T any] struct {
	value *T
}

func Some[T any](v T) Option[T] { return Option[T]{value: &v} }
func None[T any]() Option[T]    { return Option[T]{} }

func (o Option[T]) IsPresent() bool    { return o.value != nil }
func (o Option[T]) Unwrap() T          { return *o.value }
func (o Option[T]) OrElse(def T) T {
	if o.value == nil {
		return def
	}
	return *o.value
}

func main() {
	nums := []int{1, 2, 3, 4, 5, 6}

	doubled := Map(nums, func(n int) int { return n * 2 })
	fmt.Println("Map (double):", doubled)

	evens := Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("Filter (even):", evens)

	sum := Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Println("Reduce (sum):", sum)

	words := []string{"banana", "apple", "cherry"}
	fmt.Println("Min string:", Min(words))

	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(m)
	slices.Sort(keys)
	fmt.Println("Keys:", keys)

	fmt.Println("Some(42).OrElse(0):", Some(42).OrElse(0))
	fmt.Println("None[int]().OrElse(0):", None[int]().OrElse(0))
}
