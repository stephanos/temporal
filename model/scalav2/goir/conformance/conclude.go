package conformance

import "go.temporal.io/server/common/testing/testpilot"

// This file is the one place a conclusion is decided. A live Run, a replay, an Observe and a Close
// all come here with what an exploration counted, so none can weigh it on its own terms.

// status is what one execution says of one claim. The order is the precedence along a path: a
// violation is never taken back, a step the claim could not be read on leaves it unknown whatever
// held before, and a claim that held was read.
type status uint8

const (
	unread status = iota
	held
	unknown
	violated
)

// tally is what the executions left say of one claim: how many say each thing, and whether a hole was
// in reach of one that had not violated it, which is an execution nobody knows.
type tally struct {
	count   [violated + 1]int
	tainted bool
}

// Why a conclusion is what it is. They are details of an Assessment, and no stable API.
const (
	whyNoEvidence   = "the Run recorded no evidence"
	whyIncomplete   = "the Run did not close complete"
	whyHole         = "a hole of the Model is in reach of an execution that explains the evidence"
	whyUnexplained  = "no modeled execution explains the evidence"
	whyDisagreement = "the executions that explain the evidence disagree"
	whyNeverRead    = "an execution that explains the evidence never reaches the claim's evaluation point"
	whyUnreadable   = "the claim cannot be read on an execution that explains the evidence"
	whyViolated     = "every modeled execution that explains the evidence violates it"
)

// claimConclusion is one claim on one instance of the machine. A violation needs every execution
// left to violate it and no hole beside one that does not; it stands on a Run that did not close
// complete. Satisfaction needs every execution to read the claim and none to violate it, and is
// said only of a Run that closed complete: positive.
func claimConclusion(t tally, positive bool) (testpilot.PropertyStatus, string) {
	total := 0
	for _, n := range t.count {
		total += n
	}
	switch {
	case total == 0:
		return testpilot.PropertyInconclusive, whyUnexplained
	case t.tainted:
		return testpilot.PropertyInconclusive, whyHole
	case t.count[violated] == total:
		return testpilot.PropertyViolated, whyViolated
	case t.count[violated] > 0:
		return testpilot.PropertyInconclusive, whyDisagreement
	case t.count[unknown] > 0:
		return testpilot.PropertyInconclusive, whyUnreadable
	case t.count[unread] > 0:
		return testpilot.PropertyInconclusive, whyNeverRead
	case !positive:
		return testpilot.PropertyInconclusive, whyIncomplete
	default:
		return testpilot.PropertySatisfied, ""
	}
}

// conformanceConclusion is one instance's conformance: candidates executions explain its evidence,
// holes says a hole was in reach of one that was tried. No execution left is a nonconformance unless
// a hole could account for it, and it stands on a Run that did not close complete; an execution left
// conforms only on one that did.
func conformanceConclusion(candidates int, holes, positive bool) (testpilot.ConformanceStatus, string) {
	switch {
	case candidates == 0 && holes:
		return testpilot.ConformanceInconclusive, whyHole
	case candidates == 0:
		return testpilot.ConformanceNonconformant, whyUnexplained
	case !positive:
		return testpilot.ConformanceInconclusive, whyIncomplete
	default:
		return testpilot.ConformanceConformant, ""
	}
}

// over is one conclusion for several instances of the machine: bad when any is, good when all are,
// and open otherwise, for the reason of the first that is neither. No instance at all is no evidence.
func over[S comparable](parts []S, reasons []string, bad, good, open S) (S, string) {
	if len(parts) == 0 {
		return open, whyNoEvidence
	}
	reason := ""
	for i, part := range parts {
		switch {
		case part == bad:
			return bad, reasons[i]
		case part != good && reason == "":
			reason = reasons[i]
		default:
		}
	}
	if reason != "" {
		return open, reason
	}
	return good, ""
}
