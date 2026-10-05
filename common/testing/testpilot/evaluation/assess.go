package evaluation

import (
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// KnownGapRef is one of the Case's Known Gaps as a receipt names it: its kind and its code.
type KnownGapRef struct {
	Kind string
	Code string
}

// Reason is why a Decision is not accepted, by a fixed id. Each forces one decision, except
// ReasonRuleUnsupported, which forces the one the Profile names.
type Reason string

// The reasons, in the order a Decision lists those that hold: the Verdict's, the Run's, the Case's,
// then the Model assessment's.
const (
	// ReasonVerdictViolated: the Contract's Verdict, as testpilot.ConcludeVerdict concludes it, is
	// violated. Rejected.
	ReasonVerdictViolated Reason = "verdict-violated"
	// ReasonVerdictInconclusive: the Verdict is inconclusive, which a Run that did not close complete
	// always is. Incomplete.
	ReasonVerdictInconclusive Reason = "verdict-inconclusive"
	// ReasonCleanupUnclosed: the Run's cleanup did not succeed. Incomplete.
	ReasonCleanupUnclosed Reason = "cleanup-unclosed"
	// ReasonKnownGapBlocking: the Case declares a Known Gap of a kind the Profile blocks on.
	// Incomplete.
	ReasonKnownGapBlocking Reason = "known-gap-blocking"
	// ReasonRuleUnsupported: a rule concluded at a terminal state with no supporting event. What the
	// Profile's UnsupportedRule names.
	ReasonRuleUnsupported Reason = "rule-unsupported"
	// ReasonNonconformant: the Model explains no execution of the Run. Rejected.
	ReasonNonconformant Reason = "nonconformant"
	// ReasonPropertyViolated: the Model assessment concludes a property violated. Rejected.
	ReasonPropertyViolated Reason = "property-violated"
	// ReasonAssessmentInconclusive: the Model assessment leaves the conformance or a property
	// inconclusive. Incomplete.
	ReasonAssessmentInconclusive Reason = "assessment-inconclusive"
	// ReasonAssessmentFailed: the Model assessment failed before concluding. Incomplete.
	ReasonAssessmentFailed Reason = "assessment-failed"
)

// Decision is what one Profile says of one subject: the decision and every reason that holds, in
// the fixed order, beside the recorded facts they were read from, each kept as its own field so
// that no status is folded into another.
type Decision struct {
	// Outcome is accepted, rejected or incomplete.
	Outcome string
	// Reasons are the reasons that hold, in the fixed order.
	Reasons []Reason

	ProfileName     string
	ProfileIdentity string
	Claim           string
	Trust           string

	Verdict     testpilotspb.VerdictStatus
	Disposition testpilotspb.RunDisposition
	Cleanup     testpilotspb.CleanupStatus
	KnownGaps   []KnownGapRef
	// UnsupportedRules are the rules the Verdict names at a terminal state with no supporting
	// event, in the Verdict's order.
	UnsupportedRules []string
	// Assessment is the Model assessment the subject was decided with, or nil when none was supplied.
	Assessment *testpilot.Assessment
}

// knownGapKindName is the kind as a Profile spells it.
func knownGapKindName(kind testpilotspb.KnownGapKind) string {
	return strings.ToLower(strings.TrimPrefix(kind.String(), "KNOWN_GAP_KIND_"))
}

// Assess decides one admitted subject under one Profile, and under the Model assessment of the same
// Run when one is supplied. It reads only recorded values: it prepares, runs and evaluates nothing,
// so the same subject, Profile and assessment always give the same Decision and none is changed.
//
// The precedence is fixed. A violated Verdict, a nonconformant Run, a violated property, or an
// unsupported rule under a Profile that rejects one, rejects. Otherwise an inconclusive Verdict, a
// cleanup that did not succeed, a blocking Known Gap, an unsupported rule, or a Model assessment
// that failed or left something inconclusive, leaves the subject incomplete. Otherwise it is
// accepted.
func Assess(subject *Subject, profile Profile, assessment *testpilot.Assessment) Decision {
	decision := Decision{
		ProfileName:     profile.Name,
		ProfileIdentity: profile.Identity,
		Claim:           profile.Claim,
		Trust:           profile.Trust,
		Verdict:         subject.Verdict.GetStatus(),
		Disposition:     subject.Disposition,
		Cleanup:         subject.Cleanup,
		Assessment:      assessment,
	}
	blocking := false
	for _, gap := range subject.KnownGaps {
		kind := knownGapKindName(gap.GetKind())
		decision.KnownGaps = append(decision.KnownGaps, KnownGapRef{Kind: kind, Code: gap.GetCode()})
		blocking = blocking || slices.Contains(profile.BlockingKnownGaps, kind)
	}
	for _, rule := range subject.Verdict.GetRules() {
		if rule.GetTerminalStateId() != "" && len(rule.GetSupportingEventSequences()) == 0 {
			decision.UnsupportedRules = append(decision.UnsupportedRules, rule.GetRuleId())
		}
	}

	decision.Outcome = DecisionAccepted
	hold := func(reason Reason, forces string) {
		decision.Reasons = append(decision.Reasons, reason)
		if forces == DecisionRejected || decision.Outcome == DecisionAccepted {
			decision.Outcome = forces
		}
	}
	switch concluded(subject) {
	case testpilotspb.VERDICT_STATUS_VIOLATED:
		hold(ReasonVerdictViolated, DecisionRejected)
	case testpilotspb.VERDICT_STATUS_SATISFIED:
	default:
		hold(ReasonVerdictInconclusive, DecisionIncomplete)
	}
	if subject.Cleanup != testpilotspb.CLEANUP_STATUS_SUCCEEDED {
		hold(ReasonCleanupUnclosed, DecisionIncomplete)
	}
	if blocking {
		hold(ReasonKnownGapBlocking, DecisionIncomplete)
	}
	if len(decision.UnsupportedRules) > 0 {
		hold(ReasonRuleUnsupported, profile.UnsupportedRule)
	}
	if assessment != nil {
		assessed(assessment, hold)
	}
	return decision
}

// concluded is the subject's Verdict status as testpilot.ConcludeVerdict concludes it from the Run's
// disposition and its rules, which is what a violated Run stopped by its Monitor and a Run that did
// not close complete come down to. Admission holds the recorded status to the same conclusion
// (recordedrun.Agreement); a subject built by hand that disagrees is decided by the worse of the two,
// so a disagreement never lets it through, and a Run its Monitor stopped is violated, as admission
// admits a stopped Run only beside a violation.
func concluded(subject *Subject) testpilotspb.VerdictStatus {
	statuses := make([]testpilotspb.RuleVerdictStatus, 0, len(subject.Verdict.GetRules()))
	for _, rule := range subject.Verdict.GetRules() {
		statuses = append(statuses, rule.GetStatus())
	}
	status, _ := testpilot.ConcludeVerdict(subject.Disposition, statuses)
	recorded := subject.Verdict.GetStatus()
	switch {
	case status == testpilotspb.VERDICT_STATUS_VIOLATED || recorded == testpilotspb.VERDICT_STATUS_VIOLATED ||
		subject.Disposition == testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR:
		return testpilotspb.VERDICT_STATUS_VIOLATED
	case status == testpilotspb.VERDICT_STATUS_SATISFIED && recorded == testpilotspb.VERDICT_STATUS_SATISFIED:
		return testpilotspb.VERDICT_STATUS_SATISFIED
	default:
		return testpilotspb.VERDICT_STATUS_INCONCLUSIVE
	}
}

// assessed holds the Model assessment's reasons. A violation it established stands even when the
// assessment then failed; a conclusion it did not reach is inconclusive.
func assessed(assessment *testpilot.Assessment, hold func(Reason, string)) {
	conformance := assessment.Conformance.Status
	violated, open := false, conformance != testpilot.ConformanceConformant && conformance != testpilot.ConformanceNonconformant
	for _, property := range assessment.Properties {
		switch property.Status {
		case testpilot.PropertyViolated:
			violated = true
		case testpilot.PropertySatisfied:
		default:
			open = true
		}
	}
	if conformance == testpilot.ConformanceNonconformant {
		hold(ReasonNonconformant, DecisionRejected)
	}
	if violated {
		hold(ReasonPropertyViolated, DecisionRejected)
	}
	if open {
		hold(ReasonAssessmentInconclusive, DecisionIncomplete)
	}
	if assessment.Failure != nil {
		hold(ReasonAssessmentFailed, DecisionIncomplete)
	}
}
