// runq_user_choice holds the user goroutines a program starts to a known
// count, so its Runnable decisions can be checked against that count: the
// run-queue choice offers user goroutines only, and runtime-owned goroutines
// run first by a fixed rule. Every mode prints only what is the same under
// every schedule, except two-users, whose step order is the point.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

const steps = 4

var sink [][]byte

// churn allocates and collects so the collector's runtime-owned goroutines
// (sweeper, scavenger, mark workers, finalizer goroutine) become runnable
// beside whatever user goroutine runs it.
func churn() {
	for range 2 {
		for range 64 {
			sink = append(sink, make([]byte, 64<<10))
		}
		sink = nil
		runtime.GC()
	}
}

func main() {
	if len(os.Args) != 2 {
		panic("mode is required")
	}
	switch os.Args[1] {
	case "two-users":
		// Main parks in Wait while exactly two workers take turns yielding.
		var group sync.WaitGroup
		var mutex sync.Mutex
		var order []string
		for _, name := range []string{"a", "b"} {
			group.Go(func() {
				for range steps {
					mutex.Lock()
					order = append(order, name)
					mutex.Unlock()
					runtime.Gosched()
				}
			})
		}
		group.Wait()
		fmt.Println("two-users", strings.Join(order, ""))
	case "main-only":
		// No goroutine is started; main is the only user goroutine.
		churn()
		fmt.Println("main-only done")
	case "one-user":
		// Main parks on the receive while one worker churns, so at most one
		// user goroutine is runnable at a time.
		done := make(chan struct{})
		go func() {
			churn()
			close(done)
		}()
		<-done
		fmt.Println("one-user done")
	case "busy-runtime":
		busyRuntime()
	default:
		panic("unknown mode " + os.Args[1])
	}
}

type object struct{ payload [256]byte }

// busyRuntime keeps the finalizer goroutine and the sweeper busy while two
// user goroutines yield to each other through a channel. Each worker notes
// whether finalizers had run by the time it finished, so a starved
// runtime-owned goroutine shows as false and a starved worker as a short count.
func busyRuntime() {
	const rounds = 32
	var finalized atomic.Int64
	var group sync.WaitGroup
	var mutex sync.Mutex
	steps := map[string]int{}
	sawFinalizers := map[string]bool{}
	turn := make(chan struct{}, 1)
	turn <- struct{}{}
	for _, name := range []string{"a", "b"} {
		group.Go(func() {
			for round := range rounds {
				<-turn
				for range 16 {
					value := new(object)
					runtime.SetFinalizer(value, func(*object) { finalized.Add(1) })
				}
				if round%8 == 7 {
					runtime.GC()
				}
				mutex.Lock()
				steps[name]++
				mutex.Unlock()
				turn <- struct{}{}
				runtime.Gosched()
			}
			mutex.Lock()
			sawFinalizers[name] = finalized.Load() > 0
			mutex.Unlock()
		})
	}
	group.Wait()
	fmt.Printf("busy-runtime a=%d b=%d a-saw-finalizers=%t b-saw-finalizers=%t\n", steps["a"], steps["b"], sawFinalizers["a"], sawFinalizers["b"])
}
