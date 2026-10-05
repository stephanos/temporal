package conformance

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

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

// Why a conclusion is what it is: the IR's reason, whose id (umpiremodel.ExpectationID) an Assessment
// reports and an expected Run names, and its prose, a detail of an Assessment and no stable API. This
// table is the one place a reason is worded.
type reason = umpirespb.RunExpectation_Reason

const (
	none            = umpirespb.RunExpectation_REASON_UNSPECIFIED
	whyNoEvidence   = umpirespb.RunExpectation_REASON_NO_EVIDENCE
	whyIncomplete   = umpirespb.RunExpectation_REASON_INCOMPLETE
	whyHole         = umpirespb.RunExpectation_REASON_HOLE
	whyUnexplained  = umpirespb.RunExpectation_REASON_UNEXPLAINED
	whyDisagreement = umpirespb.RunExpectation_REASON_EXPLANATIONS_DISAGREE
	whyNeverRead    = umpirespb.RunExpectation_REASON_NEVER_EVALUATED
	whyUnreadable   = umpirespb.RunExpectation_REASON_UNREADABLE
	whyViolated     = umpirespb.RunExpectation_REASON_EVERY_EXPLANATION_VIOLATES
)

var wording = map[reason]string{
	whyNoEvidence:   "the Run recorded no evidence",
	whyIncomplete:   "the Run did not close complete",
	whyHole:         "a hole of the Model is in reach of an execution that explains the evidence",
	whyUnexplained:  "no modeled execution explains the evidence",
	whyDisagreement: "the executions that explain the evidence disagree",
	whyNeverRead:    "an execution that explains the evidence never reaches the claim's evaluation point",
	whyUnreadable:   "the claim cannot be read on an execution that explains the evidence",
	whyViolated:     "every modeled execution that explains the evidence violates it",
}

// because is why one conclusion is what it is: its reason, and the detail that says it of an
// instance.
type because struct {
	reason reason
	detail string
}

// id is the reason's stable id, empty for none.
func (b because) id() string { return umpiremodel.ExpectationID(b.reason) }

// claimConclusion is one claim on one instance of the machine. A violation needs every execution
// left to violate it and no hole beside one that does not; it stands on a Run that did not close
// complete. Satisfaction needs every execution to read the claim and none to violate it, and is
// said only of a Run that closed complete: positive.
func claimConclusion(t tally, positive bool) (testpilot.PropertyStatus, reason) {
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
		return testpilot.PropertySatisfied, none
	}
}

// conformanceConclusion is one instance's conformance: candidates executions explain its evidence,
// holes says a hole was in reach of one that was tried. No execution left is a nonconformance unless
// a hole could account for it, and it stands on a Run that did not close complete; an execution left
// conforms only on one that did.
func conformanceConclusion(candidates int, holes, positive bool) (testpilot.ConformanceStatus, reason) {
	switch {
	case candidates == 0 && holes:
		return testpilot.ConformanceInconclusive, whyHole
	case candidates == 0:
		return testpilot.ConformanceNonconformant, whyUnexplained
	case !positive:
		return testpilot.ConformanceInconclusive, whyIncomplete
	default:
		return testpilot.ConformanceConformant, none
	}
}

// over is one conclusion for several instances of the machine: bad when any is, good when all are,
// and open otherwise, for the reason of the first that is neither. No instance at all is no evidence.
func over[S comparable](parts []S, reasons []because, bad, good, open S) (S, because) {
	if len(parts) == 0 {
		return open, because{reason: whyNoEvidence, detail: wording[whyNoEvidence]}
	}
	first := -1
	for i, part := range parts {
		switch {
		case part == bad:
			return bad, reasons[i]
		case part != good && first < 0:
			first = i
		default:
		}
	}
	if first >= 0 {
		return open, reasons[first]
	}
	return good, because{}
}
