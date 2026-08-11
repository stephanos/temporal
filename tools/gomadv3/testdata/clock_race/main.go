package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: clock_race <new|active-reset|stopped-reset|contexts|tickers>")
	}

	switch os.Args[1] {
	case "new":
		runNewTimers()
	case "active-reset":
		runResetCallbacks(false)
	case "stopped-reset":
		runResetCallbacks(true)
	case "contexts":
		runContexts()
	case "tickers":
		runTickers()
	default:
		panic("unknown clock race mode")
	}
}

func runNewTimers() {
	const timerCount = 24
	deadline := time.Now().Add(time.Hour)
	results := make(chan int, timerCount)
	var workers sync.WaitGroup
	workers.Add(timerCount)
	for id := range timerCount {
		timer := time.NewTimer(time.Until(deadline))
		go func() {
			defer workers.Done()
			<-timer.C
			results <- id
		}()
	}

	order := collectResults(results, timerCount)
	workers.Wait()
	fmt.Println(order)
}

func runResetCallbacks(stopped bool) {
	const timerCount = 24
	results := make(chan int, timerCount)
	timers := make([]*time.Timer, timerCount)
	var workers sync.WaitGroup
	workers.Add(timerCount)
	for id := range timerCount {
		timers[id] = time.AfterFunc(time.Duration(id+2)*time.Hour, func() {
			defer workers.Done()
			results <- id
		})
		if stopped && !timers[id].Stop() {
			panic("callback timer did not stop")
		}
	}
	deadline := time.Now().Add(time.Hour)
	for id, timer := range timers {
		wasActive := timer.Reset(time.Until(deadline))
		if wasActive == stopped {
			panic(fmt.Sprintf("timer %d active state = %v, want %v", id, wasActive, !stopped))
		}
	}
	order := collectResults(results, timerCount)
	workers.Wait()
	fmt.Println(order)
}

func runContexts() {
	const contextCount = 24
	deadline := time.Now().Add(time.Hour)
	results := make(chan int, contextCount)
	cancels := make([]context.CancelFunc, 0, contextCount)
	var workers sync.WaitGroup
	workers.Add(contextCount)
	for id := range contextCount {
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		cancels = append(cancels, cancel)
		go func() {
			defer workers.Done()
			<-ctx.Done()
			if ctx.Err() != context.DeadlineExceeded {
				panic(fmt.Sprintf("context %d error = %v", id, ctx.Err()))
			}
			results <- id
		}()
	}
	order := collectResults(results, contextCount)
	workers.Wait()
	for _, cancel := range cancels {
		cancel()
	}
	fmt.Println(order)
}

func runTickers() {
	const tickerCount = 24
	results := make(chan int, 2*tickerCount)
	var workers sync.WaitGroup
	workers.Add(tickerCount)
	for id := range tickerCount {
		ticker := time.NewTicker(time.Hour)
		go func() {
			defer workers.Done()
			<-ticker.C
			results <- id
			<-ticker.C
			results <- id + tickerCount
			ticker.Stop()
		}()
	}
	order := collectResults(results, 2*tickerCount)
	workers.Wait()
	fmt.Println(order)
}

func collectResults(results <-chan int, count int) []int {
	order := make([]int, 0, count)
	for range count {
		order = append(order, <-results)
	}
	return order
}
