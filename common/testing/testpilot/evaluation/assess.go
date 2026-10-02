package evaluation

import (
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// KnownGapRef is one of the Case's Known Gaps as a receipt names it: its kind and its code.
type KnownGapRef struct {
	Kind string
	Code string
}

// Decision is what one Profile says of one subject: the decision and every reason that holds, in
// the table's order, beside the recorded facts they were read from, each kept as its own field so
// that no status is folded into another.
type Decision struct {
	// Outcome is accepted, rejected or incomplete.
	Outcome string
	// Reasons are the rows of the Profile's table that hold, in the table's order.
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
}

// knownGapKindName is the kind as Umpire.Evaluation spells it.
func knownGapKindName(kind testpilotspb.KnownGapKind) string {
	return strings.ToLower(strings.TrimPrefix(kind.String(), "KNOWN_GAP_KIND_"))
}

// Assess decides one admitted subject under one Profile. It reads only the subject's recorded
// values: it prepares, runs and evaluates nothing, so the same subject and Profile always give the
// same Decision and the subject is never changed. A reason forcing rejected wins over one forcing
// incomplete; with no reason holding, the subject is accepted.
func Assess(subject *Subject, profile Profile) Decision {
	decision := Decision{
		ProfileName:     profile.Name,
		ProfileIdentity: profile.Identity,
		Claim:           profile.Claim,
		Trust:           profile.Trust,
		Verdict:         subject.Verdict.GetStatus(),
		Disposition:     subject.Disposition,
		Cleanup:         subject.Cleanup,
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
	holds := conditionsHolding(decision, blocking)
	decision.Outcome = DecisionAccepted
	for _, reason := range profile.Reasons {
		// A loaded Profile names only known conditions; one that is not known holds, so that a
		// condition this reader cannot evaluate can never let a subject through.
		if held, known := holds[reason.Condition]; known && !held {
			continue
		}
		decision.Reasons = append(decision.Reasons, reason)
		if reason.Decision == DecisionRejected {
			decision.Outcome = DecisionRejected
		} else if decision.Outcome != DecisionRejected {
			decision.Outcome = DecisionIncomplete
		}
	}
	return decision
}

// conditionsHolding evaluates every condition of the closed set against the recorded facts.
func conditionsHolding(decision Decision, blocking bool) map[string]bool {
	return map[string]bool{
		"verdict-violated":       decision.Verdict == testpilotspb.VERDICT_STATUS_VIOLATED,
		"verdict-inconclusive":   decision.Verdict == testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		"disposition-stopped":    decision.Disposition == testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
		"disposition-incomplete": decision.Disposition == testpilotspb.RUN_DISPOSITION_INCOMPLETE,
		"cleanup-unclosed":       decision.Cleanup != testpilotspb.CLEANUP_STATUS_SUCCEEDED,
		"known-gap-blocking":     blocking,
		"unsupported-rule":       len(decision.UnsupportedRules) > 0,
	}
}
