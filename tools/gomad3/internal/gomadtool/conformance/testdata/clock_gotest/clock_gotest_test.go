package clock_gotest

import (
	"context"
	"flag"
	"testing"
	"time"
)

var logicalTimeout = flag.Bool("gomad-logical-timeout", false, "sleep past the go test timeout to prove it is logical")

// TestVirtualClockAndDeadline must report a 24 hour logical duration.
func TestVirtualClockAndDeadline(t *testing.T) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Hour)
	defer cancel()
	<-ctx.Done()
	time.Sleep(4 * time.Hour)
	if elapsed := time.Since(start); elapsed < 24*time.Hour {
		t.Fatalf("elapsed %v, want at least 24h", elapsed)
	}
}

// TestSecondVirtualClockTest must report a 6 hour logical duration.
func TestSecondVirtualClockTest(t *testing.T) {
	start := time.Now()
	time.Sleep(6 * time.Hour)
	if elapsed := time.Since(start); elapsed < 6*time.Hour {
		t.Fatalf("elapsed %v, want at least 6h", elapsed)
	}
}

// TestLogicalTimeout sleeps past -timeout so the test binary's own timeout,
// not the wall watchdog, ends the run.
func TestLogicalTimeout(t *testing.T) {
	if !*logicalTimeout {
		t.Skip("run with -gomad-logical-timeout")
	}
	time.Sleep(2 * time.Hour)
}
