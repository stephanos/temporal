package runner

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

func TestChoiceTraceObservedExcludesWatchdogAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name   string
		result execution.Result
		want   bool
	}{
		{name: "exited", result: execution.Result{}, want: true},
		{name: "watchdog", result: execution.Result{WatchdogTimeout: true}, want: false},
		{name: "cancelled", result: execution.Result{Cancelled: true}, want: false},
	} {
		if got := choiceTraceObserved(test.result); got != test.want {
			t.Fatalf("%s: choiceTraceObserved() = %v, want %v", test.name, got, test.want)
		}
	}
}
