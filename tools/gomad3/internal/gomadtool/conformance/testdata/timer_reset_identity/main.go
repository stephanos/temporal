// One AfterFunc timer fires twice: its first callback resets it, and both
// callback goroutines park until main releases them together, so the
// scheduler has to tell the two callbacks of one timer apart.
package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func main() {
	parked := make(chan struct{}, 2)
	gate := make(chan struct{})
	done := make(chan string, 2)
	left, right := make(chan struct{}, 1), make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	var firings atomic.Int32
	var timer *time.Timer
	timer = time.AfterFunc(time.Second, func() {
		firing := firings.Add(1)
		if firing == 1 {
			timer.Reset(time.Second)
		}
		parked <- struct{}{}
		<-gate
		select {
		case <-left:
		case <-right:
		}
		done <- fmt.Sprint(firing)
	})
	<-parked
	<-parked
	// Both callback goroutines are parked before either is released, so the
	// scheduler chooses between them and the chosen callback's select follows.
	close(gate)
	fmt.Println(<-done, <-done)
}
