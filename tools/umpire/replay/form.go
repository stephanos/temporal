package replay

import (
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// Class is what one Run says about the subject's violation: the admissible violated form with the
// subject's key, a decisive answer against it, or no answer.
type Class string

const (
	// ClassReproduced: the Run is in the admissible violated form and its key is the subject's.
	ClassReproduced Class = "reproduced"
	// ClassNotReproduced: a completed satisfied Verdict, or a violated Verdict with another key.
	ClassNotReproduced Class = "not-reproduced"
	// ClassIndeterminate: an incomplete Run, an unclosed cleanup or an inconclusive Verdict.
	ClassIndeterminate Class = "indeterminate"
)

// ViolatedForm decides the admissible violated form, once, for admission and for reruns alike: a
// Run the Monitor stopped (the evaluator stops at the first violation, and records one found at
// closure the same way; it refuses a completed disposition beside a violation, so that pair is
// malformed rather than violated), whose cleanup succeeded, with a violated Verdict naming at
// least one violated rule. The reason says which part failed.
func ViolatedForm(run *testpilotspb.Run, verdict *testpilotspb.Verdict) (bool, string) {
	if run == nil || verdict == nil {
		return false, "no Run"
	}
	if run.GetDisposition() == testpilotspb.RUN_DISPOSITION_COMPLETED && verdict.GetStatus() == testpilotspb.VERDICT_STATUS_VIOLATED {
		return false, "a completed disposition beside a violated Verdict, which the Monitor never produces"
	}
	if run.GetDisposition() != testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR {
		return false, fmt.Sprintf("disposition %s, not stopped by the Monitor", run.GetDisposition())
	}
	if run.GetCleanup().GetStatus() != testpilotspb.CLEANUP_STATUS_SUCCEEDED {
		return false, fmt.Sprintf("cleanup %s, not closed", run.GetCleanup().GetStatus())
	}
	if verdict.GetStatus() != testpilotspb.VERDICT_STATUS_VIOLATED {
		return false, fmt.Sprintf("verdict %s, not violated", verdict.GetStatus())
	}
	for _, rule := range verdict.GetRules() {
		if rule.GetStatus() == testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
			return true, ""
		}
	}
	return false, "a violated Verdict naming no violated rule"
}

// Classify reads one Run against the subject's key: the admissible violated form with that key is
// reproduced; a completed satisfied Verdict or a violated Verdict with another key is not
// reproduced; anything incomplete, unclosed or inconclusive is indeterminate.
func Classify(subject ViolationKey, run *testpilotspb.Run, verdict *testpilotspb.Verdict, key ViolationKey) (Class, string) {
	if ok, reason := ViolatedForm(run, verdict); !ok {
		if run.GetDisposition() == testpilotspb.RUN_DISPOSITION_COMPLETED && verdict.GetStatus() == testpilotspb.VERDICT_STATUS_SATISFIED &&
			run.GetCleanup().GetStatus() == testpilotspb.CLEANUP_STATUS_SUCCEEDED {
			return ClassNotReproduced, "the Run satisfied the Contract"
		}
		return ClassIndeterminate, reason
	}
	if !subject.Equal(key) {
		return ClassNotReproduced, "the Run violated the Contract otherwise: " + key.String()
	}
	return ClassReproduced, ""
}

// ClassifyPair is the pair rule: any not-reproduced makes the pair not reproduced, otherwise any
// indeterminate makes it indeterminate, otherwise it is reproduced.
func ClassifyPair(classes ...Class) Class {
	pair := ClassReproduced
	for _, class := range classes {
		switch class {
		case ClassNotReproduced:
			return ClassNotReproduced
		case ClassIndeterminate:
			pair = ClassIndeterminate
		default:
		}
	}
	return pair
}
