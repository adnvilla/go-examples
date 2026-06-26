package main

import "fmt"

func main() {
	r := rect{width: 10, height: 5}
	fmt.Println("area: ", r.area())
}

type rect struct {
	width  int
	height int
}

func (r *rect) area() int {
	return r.width * r.height
}

//http://spf13.com/post/is-go-object-oriented/
