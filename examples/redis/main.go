// Redis task queue example: an HTTP endpoint enqueues a job and waits for the result,
// while background engine workers process the queue and publish results.
// Demonstrates go-redis v9 where every operation requires a context.Context.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	redisAddr              = "localhost:6379"
	queueTaskKey           = "queue:task:"
	queueProcessedKey      = "queue:processed:"
	workerTimeout          = 4 * time.Second
)

var redisClient *redis.Client

type Response struct {
	Engine []string `json:"engine"`
}

func main() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot connect to Redis at %s: %v", redisAddr, err)
	}

	const numEngines = 2
	for i := range numEngines {
		go runEngine(i)
	}

	r := gin.Default()
	r.GET("/quote", handleQuote(numEngines))
	log.Println("listening on :8080")
	r.Run(":8080")
}

func handleQuote(numEngines int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		results := make(chan string, numEngines)

		var wg sync.WaitGroup
		wg.Add(numEngines)
		for i := range numEngines {
			go func(engineID int) {
				defer wg.Done()
				result, err := enqueueAndWait(ctx, engineID)
				if err != nil {
					log.Printf("engine %d: %v", engineID, err)
					results <- fmt.Sprintf("engine %d: error", engineID)
					return
				}
				results <- result
			}(i)
		}

		wg.Wait()
		close(results)

		var response Response
		for r := range results {
			response.Engine = append(response.Engine, r)
		}
		c.JSON(http.StatusOK, response)
	}
}

func enqueueAndWait(ctx context.Context, engineID int) (string, error) {
	queueID, err := redisClient.Incr(ctx, fmt.Sprintf("%s%d:counter", queueTaskKey, engineID)).Result()
	if err != nil {
		return "", fmt.Errorf("incr: %w", err)
	}

	taskQueue := fmt.Sprintf("%s%d", queueTaskKey, engineID)
	if err := redisClient.RPush(ctx, taskQueue, queueID).Err(); err != nil {
		return "", fmt.Errorf("rpush: %w", err)
	}

	resultKey := fmt.Sprintf("%s%d:%d", queueProcessedKey, engineID, queueID)
	vals, err := redisClient.BLPop(ctx, workerTimeout, resultKey).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("timed out waiting for result")
	}
	if err != nil {
		return "", fmt.Errorf("blpop: %w", err)
	}
	return fmt.Sprintf("engine %d result: %s", engineID, vals[1]), nil
}

func runEngine(engineID int) {
	client := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		MaxRetries: 3,
	})
	taskQueue := fmt.Sprintf("%s%d", queueTaskKey, engineID)

	for {
		ctx := context.Background()
		vals, err := client.BLPop(ctx, workerTimeout, taskQueue).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			log.Printf("engine %d blpop error: %v", engineID, err)
			time.Sleep(time.Second)
			continue
		}

		jobID := vals[1]
		delay := time.Duration(rand.Intn(2000)) * time.Millisecond
		time.Sleep(delay)

		resultKey := fmt.Sprintf("%s%d:%s", queueProcessedKey, engineID, jobID)
		if err := client.RPush(ctx, resultKey, strconv.Itoa(int(delay.Milliseconds()))).Err(); err != nil {
			log.Printf("engine %d rpush error: %v", engineID, err)
		}
	}
}
