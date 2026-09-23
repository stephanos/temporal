package replay

import (
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// EvidenceCore is the subset of a recorded Run's events that the violated rules' supporting
// sequences name, exactly as the Verdict records them, in sequence order without repeats. It
// references sequences and rewrites nothing: the Run and the Verdict are read, never changed.
func EvidenceCore(verdict *testpilotspb.Verdict) []int64 {
	var core []int64
	for _, rule := range verdict.GetRules() {
		if rule.GetStatus() != testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
			continue
		}
		core = append(core, rule.GetSupportingEventSequences()...)
	}
	slices.Sort(core)
	return slices.Compact(core)
}

// OutsideCore is every instruction event of the Run that the core does not name, in any order:
// the realization's own scaffolding, which supports no violated rule. Each is named by its
// sequence and its instruction id, so a proof can say which labeled event the core omits.
func OutsideCore(run *testpilotspb.Run, core []int64) []OutsideEvent {
	named := make(map[int64]bool, len(core))
	for _, sequence := range core {
		named[sequence] = true
	}
	var outside []OutsideEvent
	for _, event := range run.GetEvents() {
		switch event.GetKind() {
		case testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT:
		default:
			continue
		}
		if named[event.GetSequence()] {
			continue
		}
		outside = append(outside, OutsideEvent{Sequence: event.GetSequence(), InstructionID: event.GetCoordinates().GetInstructionId(), Kind: event.GetKind()})
	}
	return outside
}

// OutsideEvent is one instruction event the evidence core does not name.
type OutsideEvent struct {
	Sequence      int64
	InstructionID string
	Kind          testpilotspb.RunEventKind
}
