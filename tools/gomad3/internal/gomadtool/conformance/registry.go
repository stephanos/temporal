package conformance

import "fmt"

type Mode struct {
	Tiers   []string
	Success string
}

var modes = map[string]Mode{
	"test": {
		Tiers:   []string{"test-builder", "test-live-capability", "test-runtime", "test-upstream"},
		Success: "gomad3 all black-box tiers passed",
	},
	"test-builder": {
		Tiers:   []string{"test-builder"},
		Success: "gomad3 builder tier passed",
	},
	"test-runtime": {
		Tiers:   []string{"test-runtime"},
		Success: "gomad3 runtime tier passed",
	},
	"test-upstream": {
		Tiers:   []string{"test-upstream"},
		Success: "gomad3 upstream-compatibility tier passed",
	},
	"test-interception": {
		Tiers:   []string{"test-interception"},
		Success: "gomad3 interception tier passed",
	},
	"test-live-capability": {
		Tiers:   []string{"test-live-capability"},
		Success: "gomad3 live-capability tier passed",
	},
}

func Resolve(name string) (Mode, error) {
	mode, found := modes[name]
	if !found {
		return Mode{}, fmt.Errorf("unknown gomad3 test mode: %s", name)
	}
	mode.Tiers = append([]string(nil), mode.Tiers...)
	return mode, nil
}

var schedulingSearchFixtures = []struct{ name, packageName string }{
	{name: "timer-callback-identity", packageName: "./timer_callback_identity"},
	{name: "timer-creator-identity", packageName: "./timer_creator_identity"},
	{name: "timer-reset-identity", packageName: "./timer_reset_identity"},
	{name: "select-readiness", packageName: "./select_readiness"},
}

var selectReadinessShapes = []selectShape{
	{name: "blocking-zero-ready", readyAtPoll: 0, outcomes: []string{"blocking-zero-ready first"}},
	{name: "blocking-one-ready", readyAtPoll: 1, outcomes: []string{"blocking-one-ready first"}},
	{name: "blocking-two-ready", readyAtPoll: 2, outcomes: []string{"blocking-two-ready first", "blocking-two-ready second"}},
	{name: "nonblocking-default", readyAtPoll: 0, outcomes: []string{"nonblocking-default default"}},
	{name: "timer-channel", readyAtPoll: 0, outcomes: []string{"timer-channel timer"}},
	{name: "closed-channel", readyAtPoll: 1, outcomes: []string{"closed-channel closed"}},
	{name: "nil-channel", readyAtPoll: 1, outcomes: []string{"nil-channel first"}},
}
