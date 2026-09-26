// clock_race arms many timers for the same instant and prints the order in
// which their waiters complete, which the seed must decide.
package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

const timerCount = 24

func main() {
	if len(os.Args) != 2 {
		fail("usage: clock_race <mode>")
	}
	var order []int
	switch os.Args[1] {
	case "new":
		order = race(timerCount, func(int) <-chan time.Time { return time.NewTimer(10 * time.Minute).C })
	case "active-reset":
		order = race(timerCount, func(int) <-chan time.Time {
			timer := time.NewTimer(time.Hour)
			timer.Reset(10 * time.Minute)
			return timer.C
		})
	case "stopped-reset":
		order = race(timerCount, func(int) <-chan time.Time {
			timer := time.NewTimer(time.Hour)
			timer.Stop()
			timer.Reset(10 * time.Minute)
			return timer.C
		})
	case "contexts":
		order = race(timerCount, func(int) <-chan time.Time {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			done := make(chan time.Time, 1)
			go func() {
				<-ctx.Done()
				cancel()
				done <- time.Now()
			}()
			return done
		})
	case "tickers":
		order = race(2*timerCount, func(int) <-chan time.Time { return time.NewTicker(10 * time.Minute).C })
	default:
		fail("unknown clock_race mode: " + os.Args[1])
	}
	fmt.Println(order)
}

func race(count int, arm func(int) <-chan time.Time) []int {
	completed := make(chan int, count)
	for id := range count {
		wait := arm(id)
		go func() {
			<-wait
			completed <- id
		}()
	}
	order := make([]int, 0, count)
	for range count {
		order = append(order, <-completed)
	}
	return order
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}
