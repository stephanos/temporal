package clock_synctest

import (
	"testing"
	"testing/synctest"
	"time"
)

// TestBubbleClockAdvances checks that a synctest bubble's fake clock and the
// seeded virtual clock agree on how sleeps advance time.
func TestBubbleClockAdvances(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		fired := make(chan struct{})
		time.AfterFunc(time.Hour, func() { close(fired) })
		time.Sleep(30 * time.Minute)
		select {
		case <-fired:
			t.Fatal("timer fired after 30 minutes")
		default:
		}
		time.Sleep(30 * time.Minute)
		synctest.Wait()
		select {
		case <-fired:
		default:
			t.Fatal("timer did not fire after one hour")
		}
		if elapsed := time.Since(start); elapsed != time.Hour {
			t.Fatalf("elapsed %v, want exactly 1h", elapsed)
		}
	})
}
