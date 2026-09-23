package evaluation

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

func localEphemeral(t *testing.T) Profile {
	t.Helper()
	profile, err := LoadProfile("local-ephemeral")
	require.NoError(t, err)
	return *profile
}

func localStrict(t *testing.T) Profile {
	t.Helper()
	profile, err := parseProfile(readTestProfile(t, "local-strict"))
	require.NoError(t, err)
	return *profile
}

// admitted admits the control pair after the edits.
func (c control) admitted(t *testing.T, editCase func(*testpilotspb.Case), editRun func(*testpilotspb.Run)) *Subject {
	t.Helper()
	caseBytes, recorded := c.pair(t, editCase, editRun)
	subject, err := Admit(caseBytes, recorded, controlCatalog)
	require.NoError(t, err)
	return subject
}

func withGap(kind testpilotspb.KnownGapKind) func(*testpilotspb.Case) {
	return func(source *testpilotspb.Case) {
		source.Provenance.KnownGaps = []*testpilotspb.KnownGap{{Kind: kind, Code: "umpire.gap.example"}}
	}
}

func reasonNames(decision Decision) []string {
	var names []string
	for _, reason := range decision.Reasons {
		names = append(names, reason.Name)
	}
	return names
}

// Every condition of the local table decides as the plan fixes it, every reason that holds is
// listed in the table's order, and nothing but a clean satisfied Run is accepted.
func TestAssessUnderTheLocalProfile(t *testing.T) {
	c := loadControl(t)
	profile := localEphemeral(t)
	unsupported := func(run *testpilotspb.Run) {
		satisfy(run)
		run.Verdict.Rules[1].SupportingEventSequences = nil
	}
	for name, probe := range map[string]struct {
		editCase func(*testpilotspb.Case)
		editRun  func(*testpilotspb.Run)
		outcome  string
		reasons  []string
	}{
		"a satisfied Run":                     {nil, satisfy, DecisionAccepted, nil},
		"the violated control":                {nil, nil, DecisionRejected, []string{"verdict-violated", "monitor-stopped"}},
		"a violated Run whose cleanup failed": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED }, DecisionRejected, []string{"verdict-violated", "monitor-stopped", "cleanup-unclosed"}},
		"an inconclusive incomplete Run": {nil, func(run *testpilotspb.Run) {
			satisfy(run)
			run.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE
			run.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
		}, DecisionIncomplete, []string{"verdict-inconclusive", "run-incomplete"}},
		"an inconclusive completed Run": {nil, func(run *testpilotspb.Run) {
			satisfy(run)
			run.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
			run.Verdict.Rules[0].Status = testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
		}, DecisionIncomplete, []string{"verdict-inconclusive"}},
		"a satisfied Run whose cleanup timed out": {nil, func(run *testpilotspb.Run) {
			satisfy(run)
			run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_TIMED_OUT
		}, DecisionIncomplete, []string{"cleanup-unclosed"}},
		"a capability gap":      {withGap(testpilotspb.KNOWN_GAP_KIND_CAPABILITY), satisfy, DecisionIncomplete, []string{"known-gap-blocking"}},
		"an interpretation gap": {withGap(testpilotspb.KNOWN_GAP_KIND_INTERPRETATION), satisfy, DecisionIncomplete, []string{"known-gap-blocking"}},
		"an input gap":          {withGap(testpilotspb.KNOWN_GAP_KIND_INPUT), satisfy, DecisionAccepted, nil},
		"an unsupported rule":   {nil, unsupported, DecisionIncomplete, []string{"rule-unsupported"}},
		"everything at once": {withGap(testpilotspb.KNOWN_GAP_KIND_CAPABILITY), func(run *testpilotspb.Run) {
			run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED
			run.Verdict.Rules[1].SupportingEventSequences = nil
		}, DecisionRejected, []string{"verdict-violated", "monitor-stopped", "cleanup-unclosed", "known-gap-blocking", "rule-unsupported"}},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, probe.editCase, probe.editRun)
			decision := Assess(subject, profile)
			require.Equal(t, probe.outcome, decision.Outcome)
			require.Equal(t, probe.reasons, reasonNames(decision))
		})
	}
}

// The recorded facts stay their own fields beside the decision.
func TestAssessKeepsTheRecordedFactsApart(t *testing.T) {
	c := loadControl(t)
	subject := c.admitted(t, withGap(testpilotspb.KNOWN_GAP_KIND_INPUT), func(run *testpilotspb.Run) {
		run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED
		run.Verdict.Rules[0].SupportingEventSequences = nil
	})
	decision := Assess(subject, localEphemeral(t))
	require.Equal(t, DecisionRejected, decision.Outcome)
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, decision.Verdict)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, decision.Disposition)
	require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, decision.Cleanup)
	require.Equal(t, []KnownGapRef{{Kind: "input", Code: "umpire.gap.example"}}, decision.KnownGaps)
	require.Equal(t, []string{subject.Verdict.GetRules()[0].GetRuleId()}, decision.UnsupportedRules)
	require.Equal(t, "local-ephemeral", decision.ProfileName)
	require.Equal(t, localEphemeralIdentity, decision.ProfileIdentity)
	require.Equal(t, "local-ephemeral-cluster", decision.Trust)
	require.NotEmpty(t, decision.Claim)
}

// Assessing is a pure reading: repeated and different-Profile assessments leave the subject as it
// was and agree with themselves, and another Profile is another, independent Decision.
func TestAssessIsPureAndProfilesAreIndependent(t *testing.T) {
	c := loadControl(t)
	subject := c.admitted(t, nil, func(run *testpilotspb.Run) {
		satisfy(run)
		run.Verdict.Rules[1].SupportingEventSequences = nil
	})
	before := *subject
	verdict := proto.CloneOf(subject.Verdict)
	var gaps []*testpilotspb.KnownGap
	for _, gap := range subject.KnownGaps {
		gaps = append(gaps, proto.CloneOf(gap))
	}

	local := Assess(subject, localEphemeral(t))
	require.Equal(t, local, Assess(subject, localEphemeral(t)))
	strict := Assess(subject, localStrict(t))
	require.Equal(t, DecisionIncomplete, local.Outcome)
	require.Equal(t, DecisionRejected, strict.Outcome, "the strict Profile rejects an unsupported rule")
	require.NotEqual(t, local.ProfileIdentity, strict.ProfileIdentity)
	require.Equal(t, local, Assess(subject, localEphemeral(t)), "assessing under another Profile changed nothing")

	require.True(t, proto.Equal(verdict, subject.Verdict))
	require.Len(t, subject.KnownGaps, len(gaps))
	for index, gap := range gaps {
		require.True(t, proto.Equal(gap, subject.KnownGaps[index]))
	}
	require.Equal(t, before, *subject)
}

// The reader evaluates every condition a Profile can name, and a condition it could not evaluate
// would hold rather than let a subject through.
func TestAssessEvaluatesEveryConditionAndFailsClosed(t *testing.T) {
	holds := conditionsHolding(Decision{}, false)
	var known []string
	for condition := range holds {
		known = append(known, condition)
	}
	require.ElementsMatch(t, conditions, known)

	c := loadControl(t)
	subject := c.admitted(t, nil, satisfy)
	profile := localEphemeral(t)
	profile.Reasons = append(profile.Reasons, Reason{Name: "future", Condition: "a-condition-from-the-future", Decision: DecisionIncomplete})
	decision := Assess(subject, profile)
	require.Equal(t, DecisionIncomplete, decision.Outcome)
	require.Equal(t, []string{"future"}, reasonNames(decision))
}
