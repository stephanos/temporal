package clock_synctest

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestBubbleClockTakesPrecedence(t *testing.T) {
	processStart := time.Now()
	wantStart := time.Unix(946684800, 0).UTC()
	if !processStart.Equal(wantStart) {
		t.Fatalf("process time = %v, want %v", processStart, wantStart)
	}

	synctest.Test(t, func(t *testing.T) {
		bubbleStart := time.Now()
		if !bubbleStart.Equal(wantStart) {
			t.Fatalf("bubble time = %v, want %v", bubbleStart, wantStart)
		}

		resetTimer := time.NewTimer(4 * time.Hour)
		if active := resetTimer.Reset(3 * time.Hour); !active {
			t.Fatal("active bubble timer reported inactive during reset")
		}
		if got, want := <-resetTimer.C, bubbleStart.Add(3*time.Hour); !got.Equal(want) {
			t.Fatalf("reset bubble timer fired at %v, want %v", got, want)
		}

		ticker := time.NewTicker(time.Hour)
		for tick := 1; tick <= 2; tick++ {
			if got, want := <-ticker.C, bubbleStart.Add(time.Duration(3+tick)*time.Hour); !got.Equal(want) {
				t.Fatalf("bubble ticker tick %d fired at %v, want %v", tick, got, want)
			}
		}
		ticker.Stop()
		if got := time.Since(bubbleStart); got != 5*time.Hour {
			t.Fatalf("bubble elapsed time = %v, want %v", got, 5*time.Hour)
		}
		synctest.Wait()
	})

	if got := time.Now(); !got.Equal(processStart) {
		t.Fatalf("process clock advanced with bubble: got %v, want %v", got, processStart)
	}
}
