package evaluation

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
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
	profile, err := ParseProfile(readTestProfile(t, "local-strict"))
	require.NoError(t, err)
	return *profile
}

// admitted isolates assessment conditions from the control's authored capability gap. These
// synthetic pairs are re-encoded after each edit; the pinned live pair remains unchanged.
func (c control) admitted(t *testing.T, editCase func(*testpilotspb.Case), editRun func(*testpilotspb.Run)) *Subject {
	t.Helper()
	caseBytes, recorded := c.pair(t, func(source *testpilotspb.Case) {
		source.Provenance.KnownGaps = nil
		if editCase != nil {
			editCase(source)
		}
	}, editRun)
	subject, err := Admit(caseBytes, recorded, c.catalog())
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
		names = append(names, string(reason))
	}
	return names
}

func TestAssessRetainsTheRecordedControlsAuthoredGap(t *testing.T) {
	c := loadControl(t)
	subject, err := Admit(c.caseBytes, c.recorded, c.catalog())
	require.NoError(t, err)
	decision := Assess(subject, localEphemeral(t), nil)
	require.Equal(t, DecisionRejected, decision.Outcome)
	require.Equal(t, []string{"verdict-violated", "known-gap-blocking"}, reasonNames(decision))
	require.Equal(t, []KnownGapRef{{Kind: "capability", Code: "temporal.nexus.control.action.forgedCompletion.inspect.unobserved"}}, decision.KnownGaps)
}

// Every reason decides as the fixed precedence says, every reason that holds is listed in the fixed
// order, and nothing but a clean satisfied Run is accepted. A Run stopped by its Monitor is its
// violated Verdict, and a Run that did not close complete its inconclusive one: neither is a reason
// of its own.
func TestAssessUnderTheLocalProfile(t *testing.T) {
	c := loadControl(t)
	profile := localEphemeral(t)
	unsupported := func(run *testpilotspb.Run) {
		satisfy(run)
		run.Verdict.Rules[0].SupportingEventSequences = nil
	}
	for name, probe := range map[string]struct {
		editCase func(*testpilotspb.Case)
		editRun  func(*testpilotspb.Run)
		outcome  string
		reasons  []string
	}{
		"a satisfied Run":                     {nil, satisfy, DecisionAccepted, nil},
		"the violated control":                {nil, nil, DecisionRejected, []string{"verdict-violated"}},
		"a violated Run whose cleanup failed": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED }, DecisionRejected, []string{"verdict-violated", "cleanup-unclosed"}},
		"an inconclusive incomplete Run": {nil, func(run *testpilotspb.Run) {
			satisfy(run)
			run.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE
			run.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
		}, DecisionIncomplete, []string{"verdict-inconclusive"}},
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
			run.Verdict.Rules[0].SupportingEventSequences = nil
		}, DecisionRejected, []string{"verdict-violated", "cleanup-unclosed", "known-gap-blocking", "rule-unsupported"}},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, probe.editCase, probe.editRun)
			decision := Assess(subject, profile, nil)
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
	decision := Assess(subject, localEphemeral(t), nil)
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
		run.Verdict.Rules[0].SupportingEventSequences = nil
	})
	before := *subject
	verdict := proto.CloneOf(subject.Verdict)
	var gaps []*testpilotspb.KnownGap
	for _, gap := range subject.KnownGaps {
		gaps = append(gaps, proto.CloneOf(gap))
	}

	local := Assess(subject, localEphemeral(t), nil)
	require.Equal(t, local, Assess(subject, localEphemeral(t), nil))
	strict := Assess(subject, localStrict(t), nil)
	require.Equal(t, DecisionIncomplete, local.Outcome)
	require.Equal(t, DecisionRejected, strict.Outcome, "the strict Profile rejects an unsupported rule")
	require.NotEqual(t, local.ProfileIdentity, strict.ProfileIdentity)
	require.Equal(t, local, Assess(subject, localEphemeral(t), nil), "assessing under another Profile changed nothing")

	require.True(t, proto.Equal(verdict, subject.Verdict))
	require.Len(t, subject.KnownGaps, len(gaps))
	for index, gap := range gaps {
		require.True(t, proto.Equal(gap, subject.KnownGaps[index]))
	}
	require.Equal(t, before, *subject)
}

// A subject built by hand whose recorded Verdict disagrees with what testpilot.ConcludeVerdict
// concludes from its rules, which admission refuses, is decided by the worse of the two.
func TestAssessDecidesADisagreeingVerdictByTheWorse(t *testing.T) {
	c := loadControl(t)
	for name, probe := range map[string]struct {
		edit    func(*Subject)
		outcome string
		reasons []string
	}{
		"recorded violated, rules satisfied": {func(s *Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED }, DecisionRejected, []string{"verdict-violated"}},
		"recorded satisfied, a rule violated": {func(s *Subject) {
			s.Verdict.Rules[0].Status = testpilotspb.RULE_VERDICT_STATUS_VIOLATED
		}, DecisionRejected, []string{"verdict-violated"}},
		"recorded inconclusive, rules satisfied": {func(s *Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE }, DecisionIncomplete, []string{"verdict-inconclusive"}},
		"recorded satisfied, the Run incomplete": {func(s *Subject) { s.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE }, DecisionIncomplete, []string{"verdict-inconclusive"}},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, nil, satisfy)
			probe.edit(subject)
			decision := Assess(subject, localEphemeral(t), nil)
			require.Equal(t, probe.outcome, decision.Outcome)
			require.Equal(t, probe.reasons, reasonNames(decision))
		})
	}
}

// assessment is a Model assessment of the control's Run: conformant, with every property satisfied,
// until edit says otherwise.
func assessment(edit func(*testpilot.Assessment)) *testpilot.Assessment {
	assessed := &testpilot.Assessment{
		Model:       "goir.model/v1:sha256:model",
		Query:       "temporal.nexus.control/forgedCompletion/forgedCompletion#query",
		Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: []int64{8, 27}},
		Properties: []testpilot.PropertyAssessment{
			{ID: "forgedSuccess", Status: testpilot.PropertySatisfied, SupportingEventSequences: []int64{27}},
			{ID: "terminalFinality", Status: testpilot.PropertySatisfied},
		},
	}
	if edit != nil {
		edit(assessed)
	}
	return assessed
}

// A Model assessment, when one is supplied, decides beside the Verdict by the same precedence: a
// nonconformant Run or a violated property rejects, even under a satisfied Verdict; an assessment
// that failed or left anything inconclusive leaves the subject incomplete; a violation it
// established stands although it then failed. Without one, nothing of it is read.
func TestAssessDecidesTheModelAssessmentBesideTheVerdict(t *testing.T) {
	c := loadControl(t)
	profile := localEphemeral(t)
	for name, probe := range map[string]struct {
		editRun    func(*testpilotspb.Run)
		assessment *testpilot.Assessment
		outcome    string
		reasons    []string
	}{
		"no assessment":                      {satisfy, nil, DecisionAccepted, nil},
		"a conformant, satisfied assessment": {satisfy, assessment(nil), DecisionAccepted, nil},
		"a violated property, Verdict satisfied": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Properties[0].Status, a.Properties[0].Reason = testpilot.PropertyViolated, "every_explanation_violates"
		}), DecisionRejected, []string{"property-violated"}},
		"a nonconformant Run": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, Reason: "unexplained"}
			for i := range a.Properties {
				a.Properties[i].Status, a.Properties[i].Reason = testpilot.PropertyInconclusive, "unexplained"
			}
		}), DecisionRejected, []string{"nonconformant", "assessment-inconclusive"}},
		"an inconclusive conformance": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive, Reason: "hole"}
		}), DecisionIncomplete, []string{"assessment-inconclusive"}},
		"an inconclusive property": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Properties[1].Status, a.Properties[1].Reason = testpilot.PropertyInconclusive, "never_evaluated"
		}), DecisionIncomplete, []string{"assessment-inconclusive"}},
		"a failed assessment": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive}
			a.Properties = nil
			a.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentLimitExceeded, EventSequence: 12}
		}), DecisionIncomplete, []string{"assessment-inconclusive", "assessment-failed"}},
		"a violation established before the assessment failed": {satisfy, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive}
			a.Properties = []testpilot.PropertyAssessment{{ID: "forgedSuccess", Status: testpilot.PropertyViolated}}
			a.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, EventSequence: 30}
		}), DecisionRejected, []string{"property-violated", "assessment-inconclusive", "assessment-failed"}},
		"the violated control, its property violated": {nil, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive, Reason: "incomplete"}
			a.Properties = []testpilot.PropertyAssessment{{ID: "forgedSuccess", Status: testpilot.PropertyViolated, Reason: "every_explanation_violates"}}
		}), DecisionRejected, []string{"verdict-violated", "property-violated", "assessment-inconclusive"}},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, nil, probe.editRun)
			decision := Assess(subject, profile, probe.assessment)
			require.Equal(t, probe.outcome, decision.Outcome)
			require.Equal(t, probe.reasons, reasonNames(decision))
			require.Same(t, probe.assessment, decision.Assessment)
		})
	}
}
