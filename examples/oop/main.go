// Go is not object-oriented, but structs with methods and embedding cover most OOP patterns.
// This example shows methods on structs and polymorphism via interfaces.
// See composition/ for struct embedding (Go's alternative to inheritance).
package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rect struct {
	Width, Height float64
}

func (r Rect) Area() float64      { return r.Width * r.Height }
func (r Rect) Perimeter() float64 { return 2 * (r.Width + r.Height) }

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

func printShape(s Shape) {
	fmt.Printf("%T — area: %.2f, perimeter: %.2f\n", s, s.Area(), s.Perimeter())
}

func main() {
	shapes := []Shape{
		Rect{Width: 10, Height: 5},
		Circle{Radius: 7},
	}
	for _, s := range shapes {
		printShape(s)
	}
}
