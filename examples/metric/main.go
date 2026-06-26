package main

import (
	"log"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

func main() {
	statsd, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}

	tags := []string{"service:go-examples"}
	for {
		err := statsd.Count("example_metric.histogram", int64(1), tags, 1)
		if err != nil {
			log.Println(err)
		}
		log.Println("Done...")
		time.Sleep(1 * time.Second)
	}
}
