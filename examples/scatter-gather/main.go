package main

import (
	"bytes"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"fmt"
)

func main() {

	ch := make(chan string, 10)
	var buffer bytes.Buffer
	buffer.WriteString("Responses: ")

	var wg sync.WaitGroup

	wg.Add(10)

	for i := 1; i <= 10; i++ {
		go scattr(strconv.Itoa(i), &wg, ch)
	}

	wg.Wait()
	close(ch)

	for chItem := range ch {
		buffer.WriteString(chItem)
		buffer.WriteString(",")
	}

	fmt.Println(buffer.String())
}

func scattr(message string, wg *sync.WaitGroup, ch chan<- string) {
	defer wg.Done()

	var str string

	r := rand.Intn(5000)

	time.Sleep(time.Duration(r) * time.Millisecond)
	fmt.Printf("Time: %s, %d \n", message, r)

	str += message

	ch <- str
}
