package main

import (
	"errors"
	"fmt"
)

var ErrDivByZero = errors.New("division by zero")

func Add(a, b int) int      { return a + b }
func Multiply(a, b int) int { return a * b }

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, ErrDivByZero
	}
	return a / b, nil
}

// FizzBuzz returns "Fizz", "Buzz", "FizzBuzz", or the number as a string.
func FizzBuzz(n int) string {
	switch {
	case n%15 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	default:
		return fmt.Sprintf("%d", n)
	}
}
