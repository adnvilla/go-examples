package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var result []D

func TestSlice(t *testing.T) {
	slice := CreateSlice(2)
	expected := []D{D{0}, D{1}}

	assert.Equal(t, expected, slice)
}

func TestReflectionSlice(t *testing.T) {
	slice := CreateSliceReflect(2)
	expected := []D{D{0}, D{1}}

	assert.Equal(t, expected, slice)
}

func BenchmarkSlices(b *testing.B) {
	slices := []struct {
		name string
		fun  func(n int) []D
	}{
		{"ReflectionSlice", CreateSliceReflect},
		{"Slice", CreateSlice},
	}

	for _, slice := range slices {
		for x := 0.; x <= 10; x = x + 1 {
			n := int(math.Pow(2, x))
			b.Run(fmt.Sprintf("%s/%d", slice.name, n), func(pb *testing.B) {
				r := slice.fun(pb.N)

				result = r
			})
		}
	}
}

func BenchmarkSlice(b *testing.B) {
	r := CreateSlice(b.N)
	result = r
}

func BenchmarkSliceReflect(b *testing.B) {
	r := CreateSliceReflect(b.N)
	result = r
}

func BenchmarkSlices10000000(b *testing.B) {
	r := CreateSlice(10000000)
	result = r
}

func BenchmarkSliceReflect1000000(b *testing.B) {
	r := CreateSliceReflect(1000000)
	result = r
}
