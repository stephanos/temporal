package conformance

import (
	"fmt"

	"go.temporal.io/server/tools/gomad3/choice"
)

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

// selectReadinessShapes are the select shapes of the E3 fixture with the
// readiness the runtime must record for each. The first seven are the task 2
// reference shapes; timer-channel-due has its timer run inside the poll loop,
// before the lock, so the timer's send counts as a ready case; repeated-channel
// names one channel in two cases, so both cases count as ready although one
// receive empties the channel. The blocking shapes park between their polls
// and their result, so only the others show the recording adding nothing.
var selectReadinessShapes = []selectShape{
	{name: "blocking-zero-ready", outcomes: []string{"blocking-zero-ready first"}, readiness: choice.SelectReadiness{Known: true}},
	{name: "blocking-one-ready", outcomes: []string{"blocking-one-ready first"}, readiness: choice.SelectReadiness{Known: true, Ready: 1}, completesLocked: true},
	{name: "blocking-two-ready", outcomes: []string{"blocking-two-ready first", "blocking-two-ready second"}, readiness: choice.SelectReadiness{Known: true, Ready: 2}, completesLocked: true},
	{name: "nonblocking-default", outcomes: []string{"nonblocking-default default"}, readiness: choice.SelectReadiness{Known: true, Default: true}, completesLocked: true},
	{name: "timer-channel", outcomes: []string{"timer-channel timer"}, readiness: choice.SelectReadiness{Known: true, TimerChannel: true}},
	{name: "closed-channel", outcomes: []string{"closed-channel closed"}, readiness: choice.SelectReadiness{Known: true, Ready: 1, ClosedChannel: true}, completesLocked: true},
	{name: "nil-channel", outcomes: []string{"nil-channel first"}, readiness: choice.SelectReadiness{Known: true, Ready: 1, NilChannel: true}, completesLocked: true},
	{name: "timer-channel-due", outcomes: []string{"timer-channel-due timer"}, readiness: choice.SelectReadiness{Known: true, Ready: 1, TimerChannel: true}, completesLocked: true},
	{name: "repeated-channel", outcomes: []string{"repeated-channel again", "repeated-channel first"}, readiness: choice.SelectReadiness{Known: true, Ready: 2, RepeatedChannel: true}, completesLocked: true},
}
