package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type Request struct {
	message string
}

type Response struct {
	message string
}

func main() {
	workers := 10

	responseChannel := make(chan Response, workers)

	var responses []Response

	var wg sync.WaitGroup

	wg.Add(workers)

	// Broadcast
	for i := 1; i <= workers; i++ {
		request := Request{message: strconv.Itoa(i)}

		go worker(i, 5, request, &wg, responseChannel)
		fmt.Println("Worker: " + strconv.Itoa(i))
	}

	fmt.Println()
	wg.Wait()
	close(responseChannel)

	// Aggregator
	for response := range responseChannel {
		responses = append(responses, response)
	}

	// Response
	fmt.Println()
	for i := 0; i < len(responses); i++ {
		fmt.Println(responses[i].message)
	}
}

func worker(idWorker int, timeOut int, request Request, wg *sync.WaitGroup, responseChannel chan<- Response) {
	defer wg.Done()
	respond := make(chan Response, 1)

	go workerTask(idWorker, request, respond)

	select {
	case workerResponse := <-respond:
		responseChannel <- workerResponse
	case <-time.After(time.Duration(timeOut) * time.Second):
		responseChannel <- Response{message: "TimeOut: " + strconv.Itoa(idWorker)}
	}
}

func workerTask(idWorker int, request Request, ch chan<- Response) {
	r := rand.Intn(10000)

	time.Sleep(time.Duration(r) * time.Millisecond)
	fmt.Printf("Worker %d Time: %d \n", idWorker, r)

	response := Response{message: "Response: " + strconv.Itoa(idWorker)}

	ch <- response
}
