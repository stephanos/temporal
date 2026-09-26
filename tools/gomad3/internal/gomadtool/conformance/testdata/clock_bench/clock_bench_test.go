package clock_bench

import (
	"testing"
	"time"
)

var sink time.Time

// BenchmarkDisabledClockNow measures the unseeded clock read; the driver
// compares the patched toolchain against stock Go.
func BenchmarkDisabledClockNow(b *testing.B) {
	for b.Loop() {
		sink = time.Now()
	}
}
