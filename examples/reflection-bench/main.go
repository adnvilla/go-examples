package main

import (
	"flag"
	"log"
	_ "net/http/pprof"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var t []D

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// ... rest of the program ...
	r := CreateSliceReflect(10000000)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
	t = r
}

func CreateSliceReflect(n int) []D {
	elemType := reflect.TypeOf(D{})
	elemSlice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)

	for i := 0; i < n; i = i + 1 {
		item := sccanner(i)
		elemSlice = reflect.Append(elemSlice, reflect.ValueOf(item))
	}

	return elemSlice.Interface().([]D)
}

func sccanner(n int) interface{} {
	d := D{n}
	return d
}

func CreateSlice(n int) []D {
	slice := []D{}

	for i := 0; i < n; i = i + 1 {
		slice = append(slice, D{i})
	}

	return slice
}

type D struct {
	ID int
}
