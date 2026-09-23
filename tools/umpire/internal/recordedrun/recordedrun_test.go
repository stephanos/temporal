package recordedrun

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

func rule(id string, status testpilotspb.RuleVerdictStatus, sequences ...int64) *testpilotspb.RuleVerdict {
	return &testpilotspb.RuleVerdict{RuleId: id, Status: status, SupportingEventSequences: sequences}
}

func closed(disposition testpilotspb.RunDisposition, status testpilotspb.VerdictStatus, rules ...*testpilotspb.RuleVerdict) (*testpilotspb.Run, *testpilotspb.Verdict) {
	verdict := &testpilotspb.Verdict{Status: status, Rules: rules}
	return &testpilotspb.Run{Disposition: disposition, Verdict: verdict}, verdict
}

// The agreement holds both ways: violated exactly when a rule is violated and exactly on a stopped
// Run, satisfied exactly when every rule is satisfied on a completed Run, inconclusive otherwise.
func TestAgreementHoldsBothWays(t *testing.T) {
	const (
		completed  = testpilotspb.RUN_DISPOSITION_COMPLETED
		stopped    = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
		incomplete = testpilotspb.RUN_DISPOSITION_INCOMPLETE
		violated   = testpilotspb.VERDICT_STATUS_VIOLATED
		satisfied  = testpilotspb.VERDICT_STATUS_SATISFIED
		unsettled  = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
	)
	ruleViolated := rule("a", testpilotspb.RULE_VERDICT_STATUS_VIOLATED)
	ruleSatisfied := rule("b", testpilotspb.RULE_VERDICT_STATUS_SATISFIED)
	ruleInconclusive := rule("c", testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE)
	for name, probe := range map[string]struct {
		disposition testpilotspb.RunDisposition
		status      testpilotspb.VerdictStatus
		rules       []*testpilotspb.RuleVerdict
		agrees      bool
		detail      string
	}{
		"violated and stopped":                {stopped, violated, []*testpilotspb.RuleVerdict{ruleViolated, ruleSatisfied}, true, ""},
		"satisfied and completed":             {completed, satisfied, []*testpilotspb.RuleVerdict{ruleSatisfied}, true, ""},
		"inconclusive on a completed Run":     {completed, unsettled, []*testpilotspb.RuleVerdict{ruleSatisfied, ruleInconclusive}, true, ""},
		"inconclusive on an incomplete Run":   {incomplete, unsettled, []*testpilotspb.RuleVerdict{ruleSatisfied}, true, ""},
		"violated but completed":              {completed, violated, []*testpilotspb.RuleVerdict{ruleViolated}, false, "disposition"},
		"stopped but not violated":            {stopped, unsettled, []*testpilotspb.RuleVerdict{ruleInconclusive}, false, "disposition"},
		"violated status, no violated rule":   {stopped, violated, []*testpilotspb.RuleVerdict{ruleSatisfied}, false, "violated: false"},
		"a violated rule under another state": {completed, satisfied, []*testpilotspb.RuleVerdict{ruleViolated}, false, "violated: true"},
		"satisfied on an incomplete Run":      {incomplete, satisfied, []*testpilotspb.RuleVerdict{ruleSatisfied}, false, "with a rule not satisfied"},
		"inconclusive where all is satisfied": {completed, unsettled, []*testpilotspb.RuleVerdict{ruleSatisfied}, false, "every rule is satisfied"},
		"unspecified status":                  {completed, testpilotspb.VERDICT_STATUS_UNSPECIFIED, nil, false, "unspecified"},
		"a pending rule":                      {completed, unsettled, []*testpilotspb.RuleVerdict{rule("p", testpilotspb.RULE_VERDICT_STATUS_PENDING)}, false, "Pending"},
		"an unspecified rule":                 {completed, unsettled, []*testpilotspb.RuleVerdict{rule("u", testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED)}, false, "Unspecified"},
	} {
		t.Run(name, func(t *testing.T) {
			run, verdict := closed(probe.disposition, probe.status, probe.rules...)
			agrees, detail := Agreement(run, verdict)
			require.Equal(t, probe.agrees, agrees, detail)
			require.Contains(t, detail, probe.detail)
		})
	}
}

func TestCheckSupportNamesTheOwnerAndTheProblem(t *testing.T) {
	run := &testpilotspb.Run{Events: []*testpilotspb.RunEvent{{Sequence: 1}, {Sequence: 2}}}
	require.Nil(t, CheckSupport(run, &testpilotspb.Verdict{SupportingEventSequences: []int64{1, 2}, Rules: []*testpilotspb.RuleVerdict{rule("a", 0, 2)}}))
	problem := CheckSupport(run, &testpilotspb.Verdict{SupportingEventSequences: []int64{3}})
	require.Equal(t, &SupportError{Problem: SupportUnknown, Owner: "the Verdict", Sequence: 3}, problem)
	problem = CheckSupport(run, &testpilotspb.Verdict{Rules: []*testpilotspb.RuleVerdict{rule("a", 0, 1, 1)}})
	require.Equal(t, &SupportError{Problem: SupportRepeated, Owner: "rule a", Sequence: 1}, problem)
	require.EqualError(t, problem, "rule a names supporting event 1 twice")
	problem = CheckSupport(&testpilotspb.Run{Events: []*testpilotspb.RunEvent{{Sequence: 2}}}, &testpilotspb.Verdict{SupportingEventSequences: []int64{1}})
	require.Equal(t, SupportUnknown, problem.Problem, "a sequence must name the event at its position")
	require.Equal(t, SupportUnknown, CheckSupport(run, &testpilotspb.Verdict{SupportingEventSequences: []int64{0}}).Problem)
}

func TestCrossedComparesTheCaseAndProgramIDs(t *testing.T) {
	source := &testpilotspb.Case{CaseId: "c", Program: &testpilotspb.Program{ProgramId: "p"}}
	require.Empty(t, Crossed(source, &testpilotspb.Run{CaseId: "c", ProgramId: "p"}))
	require.Contains(t, Crossed(source, &testpilotspb.Run{CaseId: "d", ProgramId: "p"}), `names Case "d"`)
	require.Contains(t, Crossed(source, &testpilotspb.Run{CaseId: "c", ProgramId: "q"}), `names Program "q"`)
}

// The Case identity is the canonical bytes' digest from either stored form, and nothing else has
// one; a record carries it, and encoding refuses a value that is not a digest.
func TestCaseIdentityAndTheRecord(t *testing.T) {
	compact := []byte(`{"caseId":"c","program":{"programId":"p"}}`)
	persisted := []byte("{\n  \"caseId\": \"c\",\n  \"program\": {\n    \"programId\": \"p\"\n  }\n}\n")
	fromCompact, err := CaseIdentity(compact)
	require.NoError(t, err)
	fromPersisted, err := CaseIdentity(persisted)
	require.NoError(t, err)
	require.Equal(t, fromCompact, fromPersisted)
	require.Equal(t, Digest(compact), fromCompact)
	_, err = CaseIdentity([]byte(" " + string(compact)))
	require.Error(t, err)

	run := &testpilotspb.Run{RunId: "r", CaseId: "c", ProgramId: "p"}
	identity := testpilot.DriverIdentity{Profile: "p", Catalog: "k", Bindings: "b"}
	_, err = Encode("not-a-digest", identity, run)
	require.ErrorContains(t, err, "not a hex SHA-256")
	document, err := Encode(fromCompact, identity, run)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(document), `{"case":"`+fromCompact+`","identity":{"profile":"p","catalog":"k","bindings":"b"},"run":{`))
	decoded, err := Decode(document)
	require.NoError(t, err)
	require.Equal(t, fromCompact, decoded.Case)
	require.Equal(t, identity, decoded.Driver)
	again, err := Encode(decoded.Case, decoded.Driver, decoded.Run)
	require.NoError(t, err)
	require.Equal(t, string(document), string(again), "decoding and encoding is the identity on a record")

	for name, mutated := range map[string]string{
		"not an object":               `[]`,
		"a repeated identity":         strings.Replace(string(document), `"identity":{`, `"identity":{},"identity":{`, 1),
		"a case-folded run":           strings.Replace(string(document), `"run":`, `"Run":`, 1),
		"an unknown field":            strings.Replace(string(document), `{"case":`, `{"extra":1,"case":`, 1),
		"no identity":                 `{"case":"` + fromCompact + `","run":{}}`,
		"no Run":                      `{"case":"` + fromCompact + `","identity":{}}`,
		"an unknown Run field":        strings.Replace(string(document), `"run":{`, `"run":{"extra":1,`, 1),
		"a case that is not a string": strings.Replace(string(document), `"case":"`+fromCompact+`"`, `"case":1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Decode([]byte(mutated))
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrNoCase)
		})
	}
	_, err = Decode([]byte(`{"identity":{},"run":{}}`))
	require.ErrorIs(t, err, ErrNoCase)
}
