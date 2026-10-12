package main

import (
	"fmt"
	"time"
)

const timerCount = 24

func main() {
	completed := make(chan int, timerCount)
	deadline := time.Now().Add(time.Second)
	for id := range timerCount {
		time.AfterFunc(time.Until(deadline), func() {
			if !time.Now().Equal(deadline) {
				panic("timer callback missed the shared virtual deadline")
			}
			completed <- id
		})
	}
	order := make([]int, 0, timerCount)
	seen := make([]bool, timerCount)
	for range timerCount {
		id := <-completed
		if seen[id] {
			panic("timer callback fired twice")
		}
		seen[id] = true
		order = append(order, id)
	}
	fmt.Println(order)
}
