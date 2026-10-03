package choice

import (
	"slices"

	"go.temporal.io/server/tools/gomad3/choice"
)

// noOpSelectShapes are the shapes the select-readiness fixture proved; a shape
// off the list, a readiness the result did not record, and a ready count of
// two or more keep their decisions expanded.
var noOpSelectShapes = choice.NoOpSelectShapes()

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
		eligible := readiness.Known && readiness.Ready < 2 && slices.Contains(noOpSelectShapes, choice.NoOpSelectShape{PolledCases: trace.Decisions[end-1].Alternatives, Readiness: readiness})
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
