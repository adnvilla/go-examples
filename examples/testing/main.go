package main

import "fmt"

// Sum returns the sum of x and y.
func Sum(x int, y int) int {
	return x + y
}

func main() {
	fmt.Println(Sum(5, 5))
}
