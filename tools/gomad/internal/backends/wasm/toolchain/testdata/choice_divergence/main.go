//go:build wasip1 && wasm

package main

import (
	"os"
	"runtime"
)

//go:wasmimport gomad_fixture_v1 choice_mode
func choiceMode() uint32

//go:noinline
func recordedSelect() {
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	select {
	case <-left:
	case <-right:
	}
}

//go:noinline
func replayedSelect() {
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	select {
	case <-left:
	case <-right:
	}
}

func runnable() {
	done := make(chan int, 2)
	go func() { done <- 1 }()
	go func() { done <- 2 }()
	runtime.Gosched()
	<-done
	<-done
}

func main() {
	mode := choiceMode()
	if mode == 1 {
		if os.Args[1] == "expected-missing" {
			runnable()
		} else {
			recordedSelect()
		}
	} else if os.Args[1] == "observed-missing" {
		runnable()
	} else {
		replayedSelect()
	}
}
