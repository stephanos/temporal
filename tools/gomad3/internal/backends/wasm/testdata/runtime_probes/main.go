package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"time"
)

type collected struct{ storage [4096]byte }

var retained [4][]byte

//go:noinline
func registerCollectorCallbacks(finalized, cleaned chan int) {
	finalizer := new(collected)
	runtime.SetFinalizer(finalizer, func(*collected) { finalized <- 1 })
	cleanup := new(collected)
	runtime.AddCleanup(cleanup, func(value int) { cleaned <- value }, 1)
	runtime.KeepAlive(finalizer)
	runtime.KeepAlive(cleanup)
}

func main() {
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		panic(err)
	}
	fmt.Printf("entropy %x\n", entropy[:8])
	if os.Getenv("EXTRA_ENTROPY") == "yes" {
		for range 8 {
			if _, err := rand.Read(entropy[:]); err != nil {
				panic(err)
			}
		}
	}
	values := make(map[int]int, 32)
	for index := range 32 {
		values[index] = index
	}
	fmt.Print("map")
	for value := range values {
		fmt.Print(" ", value)
	}
	fmt.Println()

	runnable := make(chan int, 4)
	for index := range 4 {
		go func() { runnable <- index }()
	}
	runtime.Gosched()
	fmt.Println("runq", <-runnable, <-runnable, <-runnable, <-runnable)

	for _, duration := range []time.Duration{0, -1} {
		timer := time.NewTimer(duration)
		<-timer.C
		timer.Reset(time.Microsecond)
		<-timer.C
		if timer.Stop() {
			panic("expired timer stopped")
		}
	}
	farFuture := time.AfterFunc(time.Duration(math.MaxInt64), func() { panic("stopped timer fired") })
	if !farFuture.Stop() {
		panic("future timer already fired")
	}
	timerDone := make(chan int, 2)
	deadline := time.Now().Add(time.Millisecond)
	time.AfterFunc(time.Until(deadline), func() { go func() { timerDone <- 1 }() })
	time.AfterFunc(time.Until(deadline), func() { go func() { timerDone <- 2 }() })
	fmt.Println("timer callbacks", <-timerDone, <-timerDone)
	time.Sleep(time.Millisecond)
	fmt.Println("timer parking")
	var profile bytes.Buffer
	if err := pprof.StartCPUProfile(&profile); err != nil {
		panic(err)
	}
	runtime.Gosched()
	pprof.StopCPUProfile()
	if profile.Len() == 0 {
		panic("parked profile reader did not finish")
	}
	fmt.Println("note negative park wake")

	debug.SetGCPercent(20)
	finalized, cleaned := make(chan int, 1), make(chan int, 1)
	registerCollectorCallbacks(finalized, cleaned)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for index := range 128 {
		retained[index%len(retained)] = make([]byte, 128<<10)
		retained[index%len(retained)][0] = byte(index)
	}
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		panic("automatic collector did not run")
	}
	runtime.GC()
	fmt.Println("collector automatic finalizer cleanup", <-finalized, <-cleaned)

	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	selected := 0
	select {
	case selected = <-left:
	case selected = <-right:
	}
	fmt.Println("canary", selected)
	if selected == 2 {
		os.Exit(17)
	}
}
