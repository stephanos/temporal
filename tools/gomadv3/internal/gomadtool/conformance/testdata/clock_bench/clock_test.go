package clock_bench

import (
	"testing"
	"time"
)

var benchmarkTime time.Time

func BenchmarkDisabledClockNow(b *testing.B) {
	for b.Loop() {
		benchmarkTime = time.Now()
	}
}
