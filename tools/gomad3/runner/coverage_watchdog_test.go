package runner

import (
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
)

func TestChoiceTraceObservedExcludesWatchdogAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name   string
		result backend.ExecutionResult
		want   bool
	}{
		{name: "exited", result: backend.ExecutionResult{}, want: true},
		{name: "watchdog", result: backend.ExecutionResult{WatchdogTimeout: true}, want: false},
		{name: "cancelled", result: backend.ExecutionResult{Cancelled: true}, want: false},
	} {
		if got := choiceTraceObserved(test.result); got != test.want {
			t.Fatalf("%s: choiceTraceObserved() = %v, want %v", test.name, got, test.want)
		}
	}
}
