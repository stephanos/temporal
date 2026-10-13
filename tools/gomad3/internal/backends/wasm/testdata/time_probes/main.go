// Command time_probes checks the strict virtual-time contract: the clock moves
// only when every goroutine is blocked, and then jumps to the next deadline.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"
)

var start = time.Now()

func elapsed() time.Duration { return time.Since(start) }

func check(ok bool, format string, args ...any) {
	if !ok {
		fmt.Printf("FAIL "+format+"\n", args...)
		os.Exit(3)
	}
}

func main() {
	// Runnable work and zero-time polls never advance the clock.
	before := time.Now()
	ch := make(chan int)
	for range 1000 {
		select {
		case <-ch:
		default:
		}
		runtime.Gosched()
	}
	sum := 0
	for i := range 1000000 {
		sum += i
	}
	check(time.Now().Equal(before), "busy work advanced time by %v (sum %d)", time.Since(before), sum)
	fmt.Println("runnable work keeps time frozen")

	// A long sleep completes immediately in wall time at its exact deadline.
	time.Sleep(2 * time.Hour)
	check(elapsed() == 2*time.Hour, "sleep woke at %v", elapsed())
	fmt.Println("sleep 2h woke at", elapsed())

	// Equal deadlines fire together; neither callback observes a later time.
	var tieTimes [2]time.Duration
	done := make(chan int, 2)
	deadline := 30 * time.Minute
	for i := range 2 {
		time.AfterFunc(deadline, func() { tieTimes[i] = elapsed(); done <- i })
	}
	<-done
	<-done
	check(tieTimes[0] == tieTimes[1] && tieTimes[0] == 2*time.Hour+deadline, "timer tie fired at %v", tieTimes)
	fmt.Println("timer tie fired at", tieTimes[0])

	// A stopped timer never fires and does not drag the clock forward.
	stopped := time.AfterFunc(time.Hour, func() { panic("stopped timer fired") })
	check(stopped.Stop(), "timer already fired")
	time.Sleep(time.Minute)
	check(elapsed() == 2*time.Hour+deadline+time.Minute, "stopped timer moved time to %v", elapsed())
	fmt.Println("stopped timer ignored at", elapsed())

	// Context deadlines cancel at their exact virtual time.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	began := elapsed()
	<-ctx.Done()
	check(ctx.Err() == context.DeadlineExceeded && elapsed()-began == 45*time.Minute, "context deadline at %v: %v", elapsed()-began, ctx.Err())
	fmt.Println("context deadline after", elapsed()-began)

	// A ticker delivers every period across hours of virtual time.
	ticker := time.NewTicker(10 * time.Minute)
	began = elapsed()
	for range 36 {
		<-ticker.C
	}
	ticker.Stop()
	check(elapsed()-began == 6*time.Hour, "36 ticks took %v", elapsed()-began)
	fmt.Println("36 ticks took", elapsed()-began)

	// Cancellation wins over a later timer; the earlier deadline decides.
	ctx2, cancel2 := context.WithCancel(context.Background())
	time.AfterFunc(5*time.Second, cancel2)
	began = elapsed()
	select {
	case <-ctx2.Done():
	case <-time.After(time.Hour):
		check(false, "later timer won over cancellation")
	}
	check(elapsed()-began == 5*time.Second, "cancellation after %v", elapsed()-began)
	fmt.Println("cancellation after", elapsed()-began)
	fmt.Println("total virtual", elapsed())
}
