package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		panic("select shape is required")
	}
	first, second := make(chan int, 1), make(chan int, 1)
	outcome := ""
	switch os.Args[1] {
	case "blocking-zero-ready":
		time.AfterFunc(time.Second, func() { first <- 1 })
		select {
		case <-first:
			outcome = "first"
		case <-second:
			outcome = "second"
		}
	case "blocking-one-ready":
		first <- 1
		select {
		case <-first:
			outcome = "first"
		case <-second:
			outcome = "second"
		}
	case "blocking-two-ready":
		first <- 1
		second <- 2
		select {
		case <-first:
			outcome = "first"
		case <-second:
			outcome = "second"
		}
	case "nonblocking-default":
		select {
		case <-first:
			outcome = "first"
		case <-second:
			outcome = "second"
		default:
			outcome = "default"
		}
	case "timer-channel":
		timer := time.NewTimer(time.Second)
		select {
		case <-timer.C:
			outcome = "timer"
		case <-second:
			outcome = "second"
		}
	case "closed-channel":
		close(first)
		select {
		case _, ok := <-first:
			if ok {
				panic("closed receive succeeded")
			}
			outcome = "closed"
		case <-second:
			outcome = "second"
		}
	case "nil-channel":
		var disabled chan int
		first <- 1
		select {
		case <-disabled:
			outcome = "nil"
		case <-first:
			outcome = "first"
		case <-second:
			outcome = "second"
		}
	default:
		panic("unknown select shape")
	}
	fmt.Println(os.Args[1], outcome)
}
