package main

import (
	"errors"
	"testing"
)

// Table-driven tests: a single test function covers many input/output pairs.
// t.Run creates named subtests visible in go test -v output.
func TestAdd(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		a, b int
		want int
	}{
		{"positive", 2, 3, 5},
		{"negative", -2, -3, -5},
		{"mixed", -2, 3, 1},
		{"zeros", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // subtests can also run in parallel
			got := Add(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestDivide(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		a, b    float64
		want    float64
		wantErr error
	}{
		{"normal", 10, 2, 5, nil},
		{"fraction", 1, 3, 1.0 / 3.0, nil},
		{"by zero", 5, 0, 0, ErrDivByZero},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Divide(tc.a, tc.b)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Divide(%v, %v) error = %v, want %v", tc.a, tc.b, err, tc.wantErr)
			}
			if tc.wantErr == nil && got != tc.want {
				t.Errorf("Divide(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestFizzBuzz(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n    int
		want string
	}{
		{1, "1"},
		{3, "Fizz"},
		{5, "Buzz"},
		{15, "FizzBuzz"},
		{30, "FizzBuzz"},
		{7, "7"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := FizzBuzz(tc.n); got != tc.want {
				t.Errorf("FizzBuzz(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

// Fuzz test (Go 1.18+): the fuzzer generates arbitrary inputs and checks invariants.
// Run with: go test -fuzz=FuzzAdd
func FuzzAdd(f *testing.F) {
	// Seed corpus: known interesting inputs.
	f.Add(0, 0)
	f.Add(1, -1)
	f.Add(100, 200)

	f.Fuzz(func(t *testing.T, a, b int) {
		// Invariant: Add must be commutative.
		if Add(a, b) != Add(b, a) {
			t.Errorf("Add(%d, %d) != Add(%d, %d)", a, b, b, a)
		}
	})
}
