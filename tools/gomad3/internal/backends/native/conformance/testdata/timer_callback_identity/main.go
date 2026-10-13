package main

import (
	"fmt"
	"time"
)

func callbackA(done chan string, left, right chan struct{}) {
	select {
	case <-left:
	case <-right:
	}
	done <- "A"
}

func callbackB(done chan string, left, right chan struct{}) {
	select {
	case <-left:
	case <-right:
	}
	done <- "B"
}

func main() {
	armed := make(chan struct{}, 2)
	done := make(chan string, 2)
	leftA, rightA := make(chan struct{}, 1), make(chan struct{}, 1)
	leftB, rightB := make(chan struct{}, 1), make(chan struct{}, 1)
	leftA <- struct{}{}
	rightA <- struct{}{}
	leftB <- struct{}{}
	rightB <- struct{}{}
	go func() { time.AfterFunc(time.Second, func() { callbackA(done, leftA, rightA) }); armed <- struct{}{} }()
	go func() { time.AfterFunc(time.Second, func() { callbackB(done, leftB, rightB) }); armed <- struct{}{} }()
	<-armed
	<-armed
	// Both timers are armed before the virtual clock advances.
	fmt.Println(<-done, <-done)
}
