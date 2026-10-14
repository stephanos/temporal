package replay

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
)

// Over a closed, cleaned-up Run, the violated form keeps its classes in order: a completed Run
// beside a violated Verdict is malformed, an incomplete Run incomplete, a Verdict that is not
// violated or whose rules ConcludeVerdict does not conclude violated is non-violated, and what
// remains is admitted exactly when it agrees.
func TestViolatedFormConcludesThroughConcludeVerdict(t *testing.T) {
	const (
		satisfied = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
		violated  = testpilotspb.RULE_VERDICT_STATUS_VIOLATED
		pending   = testpilotspb.RULE_VERDICT_STATUS_PENDING
	)
	mixes := [][]testpilotspb.RuleVerdictStatus{nil, {satisfied}, {pending}, {violated}, {satisfied, violated}, {pending, violated}}
	for _, mix := range mixes {
		for disposition := range testpilotspb.RunDisposition_name {
			for status := range testpilotspb.VerdictStatus_name {
				disposition, status := testpilotspb.RunDisposition(disposition), testpilotspb.VerdictStatus(status)
				t.Run(fmt.Sprintf("%v/%s/%s", mix, disposition, status), func(t *testing.T) {
					verdict := &testpilotspb.Verdict{Status: status}
					for i, ruleStatus := range mix {
						verdict.Rules = append(verdict.Rules, &testpilotspb.RuleVerdict{RuleId: fmt.Sprint(i), Status: ruleStatus})
					}
					run := &testpilotspb.Run{
						RunId:       "run",
						Disposition: disposition,
						Events:      []*testpilotspb.RunEvent{{Sequence: 1}},
						Cleanup:     &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
						Verdict:     verdict,
					}
					concluded, _ := testpilot.ConcludeVerdict(disposition, mix)
					agrees, _ := recordedrun.Agreement(run, verdict)
					want := ""
					switch {
					case disposition == testpilotspb.RUN_DISPOSITION_COMPLETED && status == testpilotspb.VERDICT_STATUS_VIOLATED:
						want = ReasonMalformed
					case disposition == testpilotspb.RUN_DISPOSITION_INCOMPLETE:
						want = ReasonIncomplete
					case status != testpilotspb.VERDICT_STATUS_VIOLATED, concluded != testpilotspb.VERDICT_STATUS_VIOLATED:
						want = ReasonNonViolated
					case !agrees:
						want = ReasonMalformed
					default:
					}
					ok, class, detail := ViolatedForm(run, verdict)
					require.Equal(t, want == "", ok, detail)
					require.Equal(t, want, class, detail)
				})
			}
		}
	}
}
