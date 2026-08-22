package clock_gotest

import (
	"flag"
	"testing"
	"time"
)

var packageInitTime = time.Now()
var logicalTimeout = flag.Bool("gomad-logical-timeout", false, "run the logical timeout fixture")

func TestVirtualClockAndDeadline(t *testing.T) {
	const timeout = 48 * time.Hour
	wantStart := time.Unix(946684800, 0).UTC()
	if !packageInitTime.Equal(wantStart) {
		t.Fatalf("package init time = %v, want %v", packageInitTime, wantStart)
	}
	deadline, ok := t.Deadline()
	if !ok {
		t.Fatal("T.Deadline is absent")
	}
	if want := wantStart.Add(timeout); !deadline.Equal(want) {
		t.Fatalf("T.Deadline = %v, want %v", deadline, want)
	}

	start := time.Now()
	time.Sleep(24 * time.Hour)
	if got := time.Since(start); got != 24*time.Hour {
		t.Fatalf("elapsed time = %v, want %v", got, 24*time.Hour)
	}
}

func TestSecondVirtualClockTest(t *testing.T) {
	start := time.Now()
	time.Sleep(6 * time.Hour)
	if got := time.Since(start); got != 6*time.Hour {
		t.Fatalf("elapsed time = %v, want %v", got, 6*time.Hour)
	}
}

func TestLogicalTimeout(t *testing.T) {
	if !*logicalTimeout {
		t.Skip("logical timeout subprocess only")
	}
	time.Sleep(2 * time.Hour)
}
