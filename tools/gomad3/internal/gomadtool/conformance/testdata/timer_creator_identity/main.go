// Each mode starts the same two children after main either created no timer,
// created and stopped a timer, or created a timer that never fires before the
// program exits. The children's identities must not depend on the mode.
package main

import (
	"fmt"
	"os"
	"time"
)

func childA(done chan string, left, right chan struct{}) {
	select {
	case <-left:
	case <-right:
	}
	done <- "A"
}

func childB(done chan string, left, right chan struct{}) {
	select {
	case <-left:
	case <-right:
	}
	done <- "B"
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: timer_creator_identity none|stopped|pending")
		os.Exit(2)
	}
	// The runtime goroutines started before main are still runnable; waiting
	// for the virtual clock parks them, so the first decision after main
	// blocks below is between the two children alone.
	time.Sleep(time.Millisecond)
	switch os.Args[1] {
	case "none":
	case "stopped":
		time.AfterFunc(time.Second, func() { panic("stopped timer fired") }).Stop()
	case "pending":
		time.AfterFunc(time.Hour, func() { panic("pending timer fired") })
	default:
		fmt.Fprintln(os.Stderr, "usage: timer_creator_identity none|stopped|pending")
		os.Exit(2)
	}
	done := make(chan string, 2)
	leftA, rightA := make(chan struct{}, 1), make(chan struct{}, 1)
	leftB, rightB := make(chan struct{}, 1), make(chan struct{}, 1)
	leftA <- struct{}{}
	rightA <- struct{}{}
	leftB <- struct{}{}
	rightB <- struct{}{}
	go childA(done, leftA, rightA)
	go childB(done, leftB, rightB)
	// Both children are runnable before main blocks, so the scheduler chooses
	// between them and the chosen child's select follows that choice.
	fmt.Println(<-done, <-done)
}
