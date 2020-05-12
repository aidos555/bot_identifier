package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"sync/atomic"
	"time"
)

var currentBots int32

const (
	SECONDS = 60
	FLAG    = 100
)

func main() {
	r := gin.Default()

	counterChan := make(chan string)
	finishChan := make(chan struct{})

	r.GET("", func(c *gin.Context) {
		url := c.Request.URL.Query()
		username := url["username"][0]

		counterChan <- username
		c.JSON(200, "ok")
	})

	r.GET("/count", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"count": atomic.LoadInt32(&currentBots),
		})
	})

	go counter(counterChan, finishChan)

	if err := r.Run(":7708"); err != nil {
		log.Fatalf("fatal err: %s", err.Error())
	}

	finishChan <- struct{}{}

}

func counter(usernames <-chan string, finishChan <-chan struct{}) {
	tick := time.NewTicker(1 * time.Second)
	defer func() { tick.Stop() }()

	history := make([]map[string]int, SECONDS)
	state := make(map[string]int)

	for i := range history {
		h := make(map[string]int)
		history[i] = h
	}

	index := 0
	for {
		select {
		case username := <-usernames:
			history[index][username] = history[index][username] + 1
		case <-tick.C:
			diff := int32(0)
			nextIndex := (index + 1) % SECONDS

			for k, v := range history[nextIndex] {
				before := state[k]
				state[k] -= v
				after := state[k]

				if before >= FLAG && after < FLAG {
					diff -= 1
				}
			}

			for k, v := range history[index] {
				before := state[k]
				state[k] += v
				after := state[k]

				if before < FLAG && after >= FLAG {
					diff += 1
				}
			}

			history[nextIndex] = make(map[string]int)
			index = nextIndex

			_ = atomic.AddInt32(&currentBots, diff)
		case <-finishChan:
			break
		}
	}
}
