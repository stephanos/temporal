package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/publish"
)

// UMPIRE_RECEIPT_GOLDENS=write rewrites the receipt goldens from Assess; a changed golden is a
// changed receipt, reviewed as such.
const receiptGoldensVariable = "UMPIRE_RECEIPT_GOLDENS"

func receiptOf(t *testing.T, subject *Subject, profile Profile) []byte {
	t.Helper()
	return assessedReceiptOf(t, subject, profile, nil)
}

func assessedReceiptOf(t *testing.T, subject *Subject, profile Profile, assessed *testpilot.Assessment) []byte {
	t.Helper()
	rendered, err := Render(subject, profile, Assess(subject, profile, assessed))
	require.NoError(t, err)
	return rendered
}

// The receipt bytes of an accepted, a rejected and an incomplete Decision, each produced by Assess
// on the control's record, and of the control decided with a Model assessment, are pinned; each
// reads back to itself.
func TestReceiptGoldens(t *testing.T) {
	c := loadControl(t)
	profile := localEphemeral(t)
	for name, probe := range map[string]struct {
		editCase   func(*testpilotspb.Case)
		editRun    func(*testpilotspb.Run)
		assessment *testpilot.Assessment
		outcome    string
	}{
		"accepted": {nil, satisfy, nil, DecisionAccepted},
		"rejected": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED }, nil, DecisionRejected},
		"incomplete": {withGap(testpilotspb.KNOWN_GAP_KIND_CAPABILITY), func(run *testpilotspb.Run) {
			satisfy(run)
			run.Verdict.Rules[0].SupportingEventSequences = nil
		}, nil, DecisionIncomplete},
		"assessed": {nil, nil, assessment(func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive, Reason: "incomplete", Detail: "never recorded"}
			a.Properties = []testpilot.PropertyAssessment{{ID: "forgedSuccess", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{8, 27, 32}, Reason: "every_explanation_violates", Detail: "never recorded"}}
		}), DecisionRejected},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, probe.editCase, probe.editRun)
			rendered := assessedReceiptOf(t, subject, profile, probe.assessment)
			decoded, err := DecodeReceipt(rendered)
			require.NoError(t, err)
			require.Equal(t, probe.outcome, decoded.Decision)
			require.Equal(t, subject.CaseIdentity, decoded.Case.Identity)
			require.Equal(t, subject.RunIdentity, decoded.Run.Identity)
			require.Equal(t, localEphemeralIdentity, decoded.Profile.Identity)
			require.Equal(t, AdmissionCaps(), decoded.Caps)
			require.Len(t, ReceiptIdentity(rendered), 64)

			path := filepath.Join("testdata", "receipts", name+".json")
			if os.Getenv(receiptGoldensVariable) == "write" {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, rendered, 0o644))
			}
			golden, err := os.ReadFile(path)
			require.NoError(t, err, "write the goldens with %s=write", receiptGoldensVariable)
			require.Equal(t, string(golden), string(rendered))
			require.NotContains(t, string(rendered), "payload", "a receipt carries no event body")
			require.NotContains(t, string(rendered), "never recorded", "a receipt carries no assessment prose")
			require.Equal(t, probe.assessment != nil, decoded.Assessment != nil)
		})
	}
}

// The same subject and Profile render the same bytes; another Profile renders another receipt; a
// Decision made under one Profile is not rendered under another.
func TestReceiptsAreDeterministicAndPerProfile(t *testing.T) {
	c := loadControl(t)
	subject := c.admitted(t, nil, satisfy)
	local, strict := localEphemeral(t), localStrict(t)
	first := receiptOf(t, subject, local)
	require.Equal(t, first, receiptOf(t, subject, local))
	other := receiptOf(t, subject, strict)
	require.NotEqual(t, ReceiptIdentity(first), ReceiptIdentity(other))
	_, err := Render(subject, local, Assess(subject, strict, nil))
	require.ErrorContains(t, err, "not made under this Profile")
	violated := c.admitted(t, nil, nil)
	_, err = Render(subject, local, Assess(violated, local, nil))
	require.ErrorContains(t, err, "not made on this subject")

	// Published under their identities, the same receipt is already published the second time and
	// the other Profile's receipt stands beside it.
	root := t.TempDir()
	for _, rendered := range [][]byte{first, first, other} {
		_, err := publish.Publish(t.Context(), root, ReceiptIdentity(rendered)+".json", rendered)
		require.NoError(t, err)
	}
	listed, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, listed, 2)
}

// DecodeReceipt reads only the canonical rendering of this format version, at most the cap.
func TestDecodeReceiptIsStrict(t *testing.T) {
	c := loadControl(t)
	valid := string(receiptOf(t, c.admitted(t, nil, satisfy), localEphemeral(t)))
	assessedValid := string(assessedReceiptOf(t, c.admitted(t, nil, satisfy), localEphemeral(t), assessment(func(a *testpilot.Assessment) {
		a.Properties = nil
		a.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentCloseFailed, Detail: "never recorded", EventSequence: 31}
	})))
	decoded, err := DecodeReceipt([]byte(assessedValid))
	require.NoError(t, err)
	require.Equal(t, &ReceiptAssessmentFailure{Code: "close_failed", EventSequence: 31}, decoded.Assessment.Failure)
	require.NotContains(t, assessedValid, "never recorded")
	for name, probe := range map[string]struct {
		encoded string
		detail  string
	}{
		"an unknown key":                  {strings.Replace(valid, `{"version":2,`, `{"version":2,"extra":1,`, 1), "unknown field"},
		"a repeated key":                  {strings.Replace(valid, `"decision":`, `"decision":"rejected","decision":`, 1), "canonical form"},
		"a case-folded key":               {strings.Replace(valid, `"decision":`, `"Decision":`, 1), "canonical form"},
		"a trailing document":             {valid + valid, "canonical form"},
		"other spacing":                   {strings.Replace(valid, `{"version":2,`, `{"version": 2,`, 1), "canonical form"},
		"a null list":                     {strings.Replace(valid, `"knownGaps":[]`, `"knownGaps":null`, 1), "canonical form"},
		"a null assessment list":          {strings.Replace(assessedValid, `"properties":[]`, `"properties":null`, 1), "canonical form"},
		"a null assessment sequence list": {strings.Replace(assessedValid, `"supportingEventSequences":[8,27]`, `"supportingEventSequences":null`, 1), "canonical form"},
		"another format version":          {strings.Replace(valid, `{"version":2,`, `{"version":1,`, 1), "format version 1"},
		"not JSON":                        {"{", "decode receipt"},
		"over the cap":                    {valid + strings.Repeat(" ", MaxReceiptBytes+1-len(valid)), "receipt-oversized"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeReceipt([]byte(probe.encoded))
			require.ErrorContains(t, err, probe.detail)
		})
	}
}

// A receipt at its cap is a receipt and one byte over is the named tooling failure; a subject whose
// receipt would be over the cap renders nothing.
func TestTheReceiptCap(t *testing.T) {
	require.NoError(t, checkReceiptSize(MaxReceiptBytes))
	var oversized *ReceiptOversizedError
	require.ErrorAs(t, checkReceiptSize(MaxReceiptBytes+1), &oversized)
	require.Equal(t, ReceiptOversizedError{Size: MaxReceiptBytes + 1, Cap: MaxReceiptBytes}, *oversized)

	sequences := make([]int64, MaxRunEvents)
	for index := range sequences {
		sequences[index] = int64(index + 1)
	}
	var rules []*testpilotspb.RuleVerdict
	for _, id := range []string{"a", "b", "c"} {
		rules = append(rules, &testpilotspb.RuleVerdict{RuleId: id, Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED, TerminalStateId: "done", SupportingEventSequences: sequences})
	}
	subject := &Subject{
		Driver:      testpilot.DriverIdentity{},
		Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		Cleanup:     testpilotspb.CLEANUP_STATUS_SUCCEEDED,
		Verdict:     &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED, Rules: rules},
		Caps:        AdmissionCaps(),
	}
	profile := localEphemeral(t)
	rendered, err := Render(subject, profile, Assess(subject, profile, nil))
	require.Nil(t, rendered)
	require.ErrorAs(t, err, &oversized)
	require.Greater(t, oversized.Size, MaxReceiptBytes)
}
