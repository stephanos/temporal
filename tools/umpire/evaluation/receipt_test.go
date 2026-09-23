package evaluation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// UMPIRE_RECEIPT_GOLDENS=write rewrites the receipt goldens from Assess; a changed golden is a
// changed receipt, reviewed as such.
const receiptGoldensVariable = "UMPIRE_RECEIPT_GOLDENS"

func receiptOf(t *testing.T, subject *Subject, profile Profile) []byte {
	t.Helper()
	rendered, err := Render(subject, profile, Assess(subject, profile))
	require.NoError(t, err)
	return rendered
}

// The receipt bytes of an accepted, a rejected and an incomplete Decision, each produced by Assess
// on the control's record, are pinned; each reads back to itself.
func TestReceiptGoldens(t *testing.T) {
	c := loadControl(t)
	profile := localEphemeral(t)
	for name, probe := range map[string]struct {
		editCase func(*testpilotspb.Case)
		editRun  func(*testpilotspb.Run)
		outcome  string
	}{
		"accepted": {nil, satisfy, DecisionAccepted},
		"rejected": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED }, DecisionRejected},
		"incomplete": {withGap(testpilotspb.KNOWN_GAP_KIND_CAPABILITY), func(run *testpilotspb.Run) {
			satisfy(run)
			run.Verdict.Rules[0].SupportingEventSequences = nil
		}, DecisionIncomplete},
	} {
		t.Run(name, func(t *testing.T) {
			subject := c.admitted(t, probe.editCase, probe.editRun)
			rendered := receiptOf(t, subject, profile)
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
	_, err := Render(subject, local, Assess(subject, strict))
	require.ErrorContains(t, err, "not made under this Profile")

	// Published under their identities, the same receipt is already published the second time and
	// the other Profile's receipt stands beside it.
	root := t.TempDir()
	for _, rendered := range [][]byte{first, first, other} {
		_, err := cli.Publish(t.Context(), root, ReceiptIdentity(rendered)+".json", rendered)
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
	for name, probe := range map[string]struct {
		encoded string
		detail  string
	}{
		"an unknown key":         {strings.Replace(valid, `{"version":1,`, `{"version":1,"extra":1,`, 1), "unknown field"},
		"a repeated key":         {strings.Replace(valid, `"decision":`, `"decision":"rejected","decision":`, 1), "canonical form"},
		"a case-folded key":      {strings.Replace(valid, `"decision":`, `"Decision":`, 1), "canonical form"},
		"a trailing document":    {valid + valid, "canonical form"},
		"other spacing":          {strings.Replace(valid, `{"version":1,`, `{"version": 1,`, 1), "canonical form"},
		"a null list":            {strings.Replace(valid, `"knownGaps":[]`, `"knownGaps":null`, 1), "canonical form"},
		"another format version": {strings.Replace(valid, `{"version":1,`, `{"version":2,`, 1), "format version 2"},
		"not JSON":               {"{", "decode receipt"},
		"over the cap":           {valid + strings.Repeat(" ", MaxReceiptBytes+1-len(valid)), "receipt-oversized"},
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
	rendered, err := Render(subject, profile, Assess(subject, profile))
	require.Nil(t, rendered)
	require.ErrorAs(t, err, &oversized)
	require.Greater(t, oversized.Size, MaxReceiptBytes)
}
