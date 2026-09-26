// clock exercises the time APIs in one mode per invocation and prints
// "clock <mode> ok" when the mode's contract held.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: clock <mode>")
	}
	mode := os.Args[1]
	var err error
	switch mode {
	case "disabled":
		err = disabled()
	case "initial":
		err = initial()
	case "sleep":
		err = sleep()
	case "runnable":
		err = runnable()
	case "timers":
		err = timers()
	case "contexts":
		err = contexts()
	case "edges":
		err = edges()
	default:
		fail("unknown clock mode: " + mode)
	}
	if err != nil {
		fail(fmt.Sprintf("clock %s: %v", mode, err))
	}
	fmt.Printf("clock %s ok\n", mode)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

// disabled runs without a seed, so it only checks the stock clock contract.
func disabled() error {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		return fmt.Errorf("sleep returned after %v", elapsed)
	}
	return nil
}

func initial() error {
	now := time.Now()
	if now.IsZero() || now.Year() < 2000 {
		return fmt.Errorf("initial time %v is implausible", now)
	}
	if later := time.Now(); later.Before(now) {
		return fmt.Errorf("clock moved backwards from %v to %v", now, later)
	}
	return nil
}

func sleep() error {
	start := time.Now()
	time.Sleep(time.Hour)
	if elapsed := time.Since(start); elapsed < time.Hour {
		return fmt.Errorf("one hour sleep advanced the clock by %v", elapsed)
	}
	return nil
}

// runnable checks that time does not advance while goroutines can still run.
func runnable() error {
	const workers = 4
	const yields = 1000
	var completed atomic.Int64
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			for range yields {
				runtime.Gosched()
			}
			completed.Add(1)
		})
	}
	time.Sleep(time.Second)
	if completed.Load() != workers {
		return fmt.Errorf("%d of %d runnable workers finished before the sleep expired", completed.Load(), workers)
	}
	group.Wait()
	return nil
}

func timers() error {
	order := make(chan string, 3)
	time.AfterFunc(2*time.Hour, func() { order <- "two-hours" })
	time.AfterFunc(30*time.Minute, func() { order <- "thirty-minutes" })
	time.AfterFunc(time.Hour, func() { order <- "one-hour" })
	start := time.Now()
	for _, want := range []string{"thirty-minutes", "one-hour", "two-hours"} {
		if got := <-order; got != want {
			return fmt.Errorf("timer %s fired before %s", got, want)
		}
	}
	if elapsed := time.Since(start); elapsed < 2*time.Hour {
		return fmt.Errorf("timers completed after %v of virtual time", elapsed)
	}
	return nil
}

func contexts() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	start := time.Now()
	<-ctx.Done()
	if ctx.Err() != context.DeadlineExceeded {
		return fmt.Errorf("context finished with %v", ctx.Err())
	}
	if elapsed := time.Since(start); elapsed < time.Hour {
		return fmt.Errorf("deadline expired after %v", elapsed)
	}
	cancelled, cancelNow := context.WithTimeout(context.Background(), time.Hour)
	cancelNow()
	<-cancelled.Done()
	if cancelled.Err() != context.Canceled {
		return fmt.Errorf("cancelled context finished with %v", cancelled.Err())
	}
	return nil
}

func edges() error {
	select {
	case <-time.After(-time.Second):
	case <-time.After(time.Hour):
		return fmt.Errorf("negative duration did not fire first")
	}
	zero := time.NewTimer(0)
	<-zero.C
	if zero.Stop() {
		return fmt.Errorf("stopping a fired timer reported it active")
	}
	zero.Reset(time.Minute)
	if !zero.Stop() {
		return fmt.Errorf("stopping a reset timer reported it fired")
	}
	time.Sleep(0)
	if time.Since(time.Now()) < 0 {
		return fmt.Errorf("elapsed time went negative")
	}
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Minute))
	defer cancel()
	select {
	case <-expired.Done():
	default:
		return fmt.Errorf("past deadline was not already done")
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	<-ticker.C
	<-ticker.C
	ticker.Stop()
	return nil
}
