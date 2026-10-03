package choice

import (
	"slices"

	"go.temporal.io/server/tools/gomad3/choice"
)

// noOpSelectShape is the evidence a select-poll decision carries: how many
// cases its select polled, which is the alternatives of the select's last poll
// step, and the readiness the runtime recorded with the select's result. Poll
// order can change behavior only when at least two cases are ready, but the
// explorer trusts that for a shape only after the select-readiness fixture
// (internal/gomadtool/conformance) explored it to exhaustion with and without
// the reduction and reached the same outcomes and deadlocks; the comparison is
// retained under .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/
// select-reduction/. Eligibility is therefore a list of shapes that passed. A
// shape off the list, a readiness the result did not record, and a ready count
// of two or more keep their decisions expanded.
type noOpSelectShape struct {
	cases     uint32
	readiness choice.SelectReadiness
}

var noOpSelectShapes = []noOpSelectShape{
	{cases: 2, readiness: choice.SelectReadiness{Known: true}},                                // blocking-zero-ready
	{cases: 2, readiness: choice.SelectReadiness{Known: true, Ready: 1}},                      // blocking-one-ready
	{cases: 2, readiness: choice.SelectReadiness{Known: true, Default: true}},                 // nonblocking-default
	{cases: 2, readiness: choice.SelectReadiness{Known: true, TimerChannel: true}},            // timer-channel
	{cases: 2, readiness: choice.SelectReadiness{Known: true, Ready: 1, ClosedChannel: true}}, // closed-channel
	{cases: 2, readiness: choice.SelectReadiness{Known: true, Ready: 1, NilChannel: true}},    // nil-channel
	{cases: 2, readiness: choice.SelectReadiness{Known: true, Ready: 1, TimerChannel: true}},  // timer-channel-due
}

// noOpSelectPolls marks each decision of trace that is a select-poll decision
// of a listed shape. One select's poll steps are consecutive decisions at its
// site whose alternatives count 2, 3, ... up to the polled cases, and the
// projection gives them all the select's readiness; a run that breaks either
// pattern is not reduced.
func noOpSelectPolls(trace choice.ReplayPlan) []bool {
	marks := make([]bool, len(trace.Decisions))
	if len(trace.Readiness) != len(trace.Decisions) {
		return marks
	}
	for start := 0; start < len(trace.Decisions); {
		first := trace.Decisions[start]
		if first.Kind != choice.KindSelectPoll || first.Alternatives != 2 {
			start++
			continue
		}
		end := start + 1
		for end < len(trace.Decisions) && continuesSelect(trace.Decisions[end-1], trace.Decisions[end]) {
			end++
		}
		readiness := trace.Readiness[start]
		eligible := readiness.Known && readiness.Ready < 2 && slices.Contains(noOpSelectShapes, noOpSelectShape{cases: trace.Decisions[end-1].Alternatives, readiness: readiness})
		for index := start; index < end; index++ {
			eligible = eligible && trace.Readiness[index] == readiness
		}
		for index := start; index < end; index++ {
			marks[index] = eligible
		}
		start = end
	}
	return marks
}

func continuesSelect(previous, current choice.Decision) bool {
	return current.Kind == choice.KindSelectPoll && current.SiteOffset == previous.SiteOffset && current.SiteMissing == previous.SiteMissing && current.Alternatives == previous.Alternatives+1
}
