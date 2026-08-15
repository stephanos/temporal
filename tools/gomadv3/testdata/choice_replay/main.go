package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: choice_replay reorder ab|ba | select ITERATIONS | overflow WAVES")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "reorder":
		if os.Args[2] != "ab" && os.Args[2] != "ba" {
			fmt.Fprintln(os.Stderr, "choice_replay reorder requires ab or ba")
			os.Exit(2)
		}
		runReorderedQueue(os.Args[2])
	case "select":
		if os.Args[2] != "8" {
			fmt.Fprintln(os.Stderr, "choice_replay select requires 8 iterations")
			os.Exit(2)
		}
		runSelectSequence()
	case "overflow":
		if os.Args[2] != "2" {
			fmt.Fprintln(os.Stderr, "choice_replay overflow requires 2 waves")
			os.Exit(2)
		}
		runQueueOverflow()
	default:
		fmt.Fprintln(os.Stderr, "choice_replay mode must be reorder, select, or overflow")
		os.Exit(2)
	}
}

func runReorderedQueue(order string) {
	ready := make(chan struct{}, 3)
	firstStart := make(chan struct{})
	secondStart := make(chan struct{})
	sentinelStart := make(chan struct{})
	results := make(chan string, 2)
	go func() {
		ready <- struct{}{}
		<-firstStart
		results <- "a"
	}()
	go func() {
		ready <- struct{}{}
		<-secondStart
		results <- "b"
	}()
	go func() {
		ready <- struct{}{}
		<-sentinelStart
	}()
	for range 3 {
		<-ready
	}
	if order == "ba" {
		close(secondStart)
		close(firstStart)
	} else {
		close(firstStart)
		close(secondStart)
	}
	close(sentinelStart)
	fmt.Println(<-results + <-results)
}

func runSelectSequence() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	var output strings.Builder
	for range 8 {
		select {
		case <-left:
			output.WriteByte('a')
			left <- struct{}{}
		case <-right:
			output.WriteByte('b')
			right <- struct{}{}
		}
	}
	fmt.Println(output.String())
}

func runQueueOverflow() {
	for range 2 {
		var start sync.WaitGroup
		var finished sync.WaitGroup
		ready := make(chan struct{})
		start.Add(1024)
		finished.Add(1024)
		go func() {
			start.Wait()
			close(ready)
		}()
		for range 1024 {
			go func() {
				defer finished.Done()
				start.Wait()
				<-ready
			}()
			start.Done()
		}
		finished.Wait()

		start = sync.WaitGroup{}
		finished = sync.WaitGroup{}
		var available atomic.Bool
		start.Add(1024)
		finished.Add(1024)
		go func() {
			start.Wait()
			available.Store(true)
		}()
		for range 1024 {
			go func() {
				defer finished.Done()
				start.Wait()
				for !available.Load() {
					runtime.Gosched()
				}
			}()
			start.Done()
		}
		finished.Wait()
	}
	fmt.Println("overflow")
}
