package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var redisClient *redis.Client

const redisAdrr string = "192.168.99.101:6379"
const queueTaskID string = "myqueue.task"
const queueProcessedTaskID string = "myqueue.processedtask"

// Response Response
type Response struct {
	Engine []string `json:"Engine"`
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}

func setupRouter() *gin.Engine {

	redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAdrr,
		Network:      "tcp",
		ReadTimeout:  0,
		WriteTimeout: 0,
		Password:     "", // no password set
		DB:           0,  // use default DB
	})

	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	engines := 2

	for i := 0; i < engines; i++ {
		go workerEngine(i)
	}

	r.GET("/quote", func(c *gin.Context) {
		var wg sync.WaitGroup

		responseChan := make(chan Response, engines)

		response := Response{
			Engine: []string{},
		}

		wg.Add(engines)
		for i := 0; i < engines; i++ {
			go workerEngineTask(i, responseChan, &wg)
		}

		go func() {
			defer close(responseChan)
			wg.Wait()
		}()

		for resp := range responseChan {
			response.Engine = append(response.Engine, resp.Engine...)
			//log.Println("Append")
		}

		c.JSON(http.StatusOK, response)
	})

	return r
}

func workerEngineTask(prefixEngine int, rChan chan Response, wg *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err := r.(error)
			log.Println("ERROR")
			log.Println(err)

			rChan <- Response{}
		}
		wg.Done()
	}()

	queueID, err := redisClient.IncrBy(strconv.Itoa(prefixEngine)+"queueId", 1).Result()
	//log.Println("IncrBy")
	if err != nil {
		log.Println(err)
		rChan <- Response{}
		return
	}

	taskID := strconv.Itoa(prefixEngine) + queueTaskID
	//log.Println(queueID)

	_, err = redisClient.RPush(taskID, strconv.FormatInt(queueID, 10)).Result()
	//log.Println("RPush")
	if err != nil {
		log.Println(err)
		rChan <- Response{}
		return
	}
	//log.Println(i)

	queueProcessedTask := strconv.Itoa(prefixEngine) + queueProcessedTaskID + strconv.FormatInt(queueID, 10)

	durationString := "4000ms"
	duration, _ := time.ParseDuration(durationString)
	processedResult, err := redisClient.BLPop(duration, queueProcessedTask).Result()
	if redis.Nil == err {
		log.Println(err)
		rChan <- Response{}
		return
	}

	if err != nil {
		log.Println(err)
		rChan <- Response{}
		return
	}

	var jobID string
	if processedResult != nil && len(processedResult) == 2 && processedResult[0] == queueProcessedTask {
		jobID = processedResult[1]
	} else {
		log.Println(fmt.Sprintf("Se obtiene incorrectamente items de la lista: %#v", queueProcessedTask))
		rChan <- Response{}
		return
	}

	//log.Println(string(jobID))

	rChan <- Response{
		Engine: []string{strconv.Itoa(prefixEngine) + " " + string(jobID)},
	}

	//log.Println("BLPop")
}

func workerEngine(prefixEngine int) {

	redisEngineClient := redis.NewClient(&redis.Options{
		Addr:         redisAdrr,
		Network:      "tcp",
		ReadTimeout:  0,
		WriteTimeout: 0,
		MaxRetries:   3,
		Password:     "", // no password set
		DB:           0,  // use default DB
	})

	taskID := strconv.Itoa(prefixEngine) + queueTaskID

	for {
		durationString := "4000ms"
		duration, _ := time.ParseDuration(durationString)
		job, err := redisEngineClient.BLPop(duration, taskID).Result()

		if redis.Nil == err {
			log.Println(err)
			continue
		}

		if err != nil {
			log.Println(err)
			return
		}

		var jobID string
		if job != nil && len(job) == 2 && job[0] == taskID {
			jobID = job[1]
		} else {
			log.Println(fmt.Sprintf("Se obtiene incorrectamente items de la lista: %#v", taskID))
			continue
		}
		//log.Println("BLPop")

		r := rand.Intn(2000)
		time.Sleep(time.Duration(r) * time.Millisecond)

		_, err = redisClient.RPush(strconv.Itoa(prefixEngine)+queueProcessedTaskID+string(jobID), strconv.Itoa(r)).Result()
		//log.Println("RPush")
		if err != nil {
			log.Println(err)
			return
		}
		//log.Println("=> " + strconv.FormatInt(i, 10))

	}

}
