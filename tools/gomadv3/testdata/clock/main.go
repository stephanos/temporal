package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

var initTime = time.Now()

func main() {
	if len(os.Args) != 2 {
		panic("usage: clock <disabled|initial|sleep|runnable|timers|contexts|edges>")
	}

	start := time.Now()
	switch os.Args[1] {
	case "disabled":
		if initTime.UnixNano() == 946684800000000000 {
			panic("disabled clock used the Gomad initial time")
		}
		time.Sleep(time.Millisecond)
		if elapsed := time.Since(start); elapsed < time.Millisecond {
			panic(fmt.Sprintf("disabled clock elapsed time = %v, want at least %v", elapsed, time.Millisecond))
		}
	case "initial":
		if got, want := initTime.UnixNano(), int64(946684800000000000); got != want {
			panic(fmt.Sprintf("init time = %d, want %d", got, want))
		}
		if !start.Equal(initTime) {
			panic(fmt.Sprintf("main time = %v, want %v", start, initTime))
		}
		if time.Local.String() != "UTC" {
			panic(fmt.Sprintf("time.Local = %v, want UTC", time.Local))
		}
	case "sleep":
		if got, want := time.Until(start.Add(24*time.Hour)), 24*time.Hour; got != want {
			panic(fmt.Sprintf("time until deadline = %v, want %v", got, want))
		}
		time.Sleep(24 * time.Hour)
		end := time.Now()
		if got, want := end.Sub(start), 24*time.Hour; got != want {
			panic(fmt.Sprintf("monotonic subtraction = %v, want %v", got, want))
		}
		if got, want := time.Since(start), 24*time.Hour; got != want {
			panic(fmt.Sprintf("elapsed time = %v, want %v", got, want))
		}
	case "runnable":
		future := time.NewTimer(time.Hour)
		for range 1_000_000 {
		}
		select {
		case got := <-future.C:
			panic(fmt.Sprintf("future timer skipped runnable work and fired at %v", got))
		default:
		}
		if got := time.Now(); !got.Equal(start) {
			panic(fmt.Sprintf("time advanced during runnable work: got %v, want %v", got, start))
		}
		if got := <-future.C; !got.Equal(start.Add(time.Hour)) {
			panic(fmt.Sprintf("future timer fired at %v, want %v", got, start.Add(time.Hour)))
		}
	case "timers":
		checkTimers()
	case "contexts":
		checkContexts(start)
	case "edges":
		checkEdges()
	default:
		panic(fmt.Sprintf("unknown clock case %q", os.Args[1]))
	}

	fmt.Printf("clock %s ok\n", os.Args[1])
}

func checkTimers() {
	noStaleStart := time.Now()
	noStale := time.NewTimer(time.Hour)
	if !noStale.Stop() {
		panic("timer for stale-send check did not stop")
	}
	fallback := time.NewTimer(2 * time.Hour)
	select {
	case got := <-noStale.C:
		panic(fmt.Sprintf("stopped timer delivered stale value %v", got))
	case got := <-fallback.C:
		if want := noStaleStart.Add(2 * time.Hour); !got.Equal(want) {
			panic(fmt.Sprintf("fallback timer fired at %v, want %v", got, want))
		}
	}

	stoppedStart := time.Now()
	stopped := time.NewTimer(time.Hour)
	if !stopped.Stop() {
		panic("active timer Stop returned false")
	}
	if stopped.Reset(3 * time.Hour) {
		panic("stopped timer Reset returned true")
	}
	if got := <-stopped.C; !got.Equal(stoppedStart.Add(3 * time.Hour)) {
		panic(fmt.Sprintf("reset stopped timer fired at %v, want %v", got, stoppedStart.Add(3*time.Hour)))
	}
	if stopped.Stop() {
		panic("expired timer Stop returned true")
	}

	activeStart := time.Now()
	active := time.NewTimer(5 * time.Hour)
	if !active.Reset(2 * time.Hour) {
		panic("active timer Reset returned false")
	}
	if got := <-active.C; !got.Equal(activeStart.Add(2 * time.Hour)) {
		panic(fmt.Sprintf("reset active timer fired at %v, want %v", got, activeStart.Add(2*time.Hour)))
	}
	if active.Reset(time.Hour) {
		panic("expired timer Reset returned true")
	}
	if got := <-active.C; !got.Equal(activeStart.Add(3 * time.Hour)) {
		panic(fmt.Sprintf("reset expired timer fired at %v, want %v", got, activeStart.Add(3*time.Hour)))
	}

	callbackStart := time.Now()
	callback := make(chan time.Time, 1)
	time.AfterFunc(time.Hour, func() {
		callback <- time.Now()
	})
	if got := <-callback; !got.Equal(callbackStart.Add(time.Hour)) {
		panic(fmt.Sprintf("AfterFunc ran at %v, want %v", got, callbackStart.Add(time.Hour)))
	}

	tickerStart := time.Now()
	ticker := time.NewTicker(time.Hour)
	if got := <-ticker.C; !got.Equal(tickerStart.Add(time.Hour)) {
		panic(fmt.Sprintf("ticker fired at %v, want %v", got, tickerStart.Add(time.Hour)))
	}
	ticker.Reset(30 * time.Minute)
	if got := <-ticker.C; !got.Equal(tickerStart.Add(90 * time.Minute)) {
		panic(fmt.Sprintf("reset ticker fired at %v, want %v", got, tickerStart.Add(90*time.Minute)))
	}
	ticker.Stop()
	stoppedTickerStart := time.Now()
	select {
	case got := <-ticker.C:
		panic(fmt.Sprintf("stopped ticker delivered stale value %v", got))
	case got := <-time.After(15 * time.Minute):
		if want := stoppedTickerStart.Add(15 * time.Minute); !got.Equal(want) {
			panic(fmt.Sprintf("After fired at %v, want %v", got, want))
		}
	}

	coalescedStart := time.Now()
	coalesced := time.NewTicker(time.Hour)
	gate := time.NewTimer(5 * time.Hour)
	<-gate.C
	if got := <-coalesced.C; !got.Equal(coalescedStart.Add(time.Hour)) {
		panic(fmt.Sprintf("coalesced ticker fired at %v, want %v", got, coalescedStart.Add(time.Hour)))
	}
	if got := <-coalesced.C; !got.Equal(coalescedStart.Add(6 * time.Hour)) {
		panic(fmt.Sprintf("ticker after coalescing fired at %v, want %v", got, coalescedStart.Add(6*time.Hour)))
	}
	coalesced.Stop()

	tickStart := time.Now()
	if got := <-time.Tick(time.Hour); !got.Equal(tickStart.Add(time.Hour)) {
		panic(fmt.Sprintf("Tick fired at %v, want %v", got, tickStart.Add(time.Hour)))
	}
}

func checkContexts(start time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()
	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		panic(fmt.Sprintf("timeout context error = %v, want %v", ctx.Err(), context.DeadlineExceeded))
	}
	if got := time.Now(); !got.Equal(start.Add(4 * time.Hour)) {
		panic(fmt.Sprintf("timeout context completed at %v, want %v", got, start.Add(4*time.Hour)))
	}

	cancelStart := time.Now()
	parent, cancelParent := context.WithCancel(context.Background())
	child, cancelChild := context.WithTimeout(parent, 5*time.Hour)
	cancelParent()
	<-child.Done()
	if !errors.Is(child.Err(), context.Canceled) {
		panic(fmt.Sprintf("canceled context error = %v, want %v", child.Err(), context.Canceled))
	}
	if got := time.Now(); !got.Equal(cancelStart) {
		panic(fmt.Sprintf("cancellation advanced time: got %v, want %v", got, cancelStart))
	}
	cancelChild()

	deadlineStart := time.Now()
	deadlineCtx, cancelDeadline := context.WithDeadline(context.Background(), deadlineStart.Add(3*time.Hour))
	defer cancelDeadline()
	<-deadlineCtx.Done()
	if !errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
		panic(fmt.Sprintf("deadline context error = %v, want %v", deadlineCtx.Err(), context.DeadlineExceeded))
	}
	if got := time.Now(); !got.Equal(deadlineStart.Add(3 * time.Hour)) {
		panic(fmt.Sprintf("deadline context completed at %v, want %v", got, deadlineStart.Add(3*time.Hour)))
	}

	childCancelStart := time.Now()
	canceledParent, cancelCanceledParent := context.WithCancel(context.Background())
	canceledChild, cancelCanceledChild := context.WithTimeout(canceledParent, 5*time.Hour)
	childDeliveries := make(chan struct{}, 2)
	context.AfterFunc(canceledChild, func() {
		childDeliveries <- struct{}{}
	})
	cancelCanceledChild()
	<-canceledChild.Done()
	<-childDeliveries
	if !errors.Is(canceledChild.Err(), context.Canceled) {
		panic(fmt.Sprintf("child cancellation error = %v, want %v", canceledChild.Err(), context.Canceled))
	}
	if err := canceledParent.Err(); err != nil {
		panic(fmt.Sprintf("parent error after child cancellation = %v, want nil", err))
	}
	if got := time.Now(); !got.Equal(childCancelStart) {
		panic(fmt.Sprintf("child cancellation advanced time: got %v, want %v", got, childCancelStart))
	}
	if got := <-time.After(6 * time.Hour); !got.Equal(childCancelStart.Add(6 * time.Hour)) {
		panic(fmt.Sprintf("child cancellation cleanup timer fired at %v, want %v", got, childCancelStart.Add(6*time.Hour)))
	}
	select {
	case <-childDeliveries:
		panic("canceled child delivered a second cancellation callback")
	default:
	}
	cancelCanceledParent()
}

func checkEdges() {
	for _, duration := range []time.Duration{0, -1, -time.Hour} {
		start := time.Now()
		if got := <-time.After(duration); !got.Equal(start) {
			panic(fmt.Sprintf("timer duration %v fired at %v, want %v", duration, got, start))
		}
	}

	const nearMax = int64(1<<63 - 2)
	nearDuration := time.Duration(nearMax - time.Now().UnixNano())
	nearTimer := time.NewTimer(nearDuration)
	if got := <-nearTimer.C; got.UnixNano() != nearMax {
		panic(fmt.Sprintf("near-maximum timer fired at %d, want %d", got.UnixNano(), nearMax))
	}
	if got := time.Now().UnixNano(); got != nearMax {
		panic(fmt.Sprintf("near-maximum clock = %d, want %d", got, nearMax))
	}

	maxTimer := time.NewTimer(time.Duration(1<<63 - 1))
	got := <-maxTimer.C
	if now := time.Now(); !got.Equal(now) {
		panic(fmt.Sprintf("maximum-duration timer fired at %v, want %v", got, now))
	}
	if got, want := time.Now().UnixNano(), int64(1<<63-1); got != want {
		panic(fmt.Sprintf("maximum-duration clock = %d, want %d", got, want))
	}
}
