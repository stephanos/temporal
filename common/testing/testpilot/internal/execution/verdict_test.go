package execution

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

var (
	terminalDispositions = []testpilotspb.RunDisposition{
		testpilotspb.RUN_DISPOSITION_UNSPECIFIED,
		testpilotspb.RUN_DISPOSITION_COMPLETED,
		testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
		testpilotspb.RUN_DISPOSITION_INCOMPLETE,
	}
	verdictRuleMixes = []struct {
		name  string
		rules []testpilotspb.RuleVerdictStatus
		// onCompleted is the Verdict a completed Run concludes; violated says the mix violates.
		onCompleted testpilotspb.VerdictStatus
		violated    bool
	}{
		{"no rules", nil, testpilotspb.VERDICT_STATUS_SATISFIED, false},
		{"all satisfied", rules(testpilotspb.RULE_VERDICT_STATUS_SATISFIED, testpilotspb.RULE_VERDICT_STATUS_SATISFIED), testpilotspb.VERDICT_STATUS_SATISFIED, false},
		{"one inconclusive", rules(testpilotspb.RULE_VERDICT_STATUS_SATISFIED, testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE), testpilotspb.VERDICT_STATUS_INCONCLUSIVE, false},
		{"one pending", rules(testpilotspb.RULE_VERDICT_STATUS_PENDING, testpilotspb.RULE_VERDICT_STATUS_SATISFIED), testpilotspb.VERDICT_STATUS_INCONCLUSIVE, false},
		{"one unspecified", rules(testpilotspb.RULE_VERDICT_STATUS_SATISFIED, testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED), testpilotspb.VERDICT_STATUS_INCONCLUSIVE, false},
		{"violated", rules(testpilotspb.RULE_VERDICT_STATUS_VIOLATED), testpilotspb.VERDICT_STATUS_VIOLATED, true},
		{"violated beside satisfied", rules(testpilotspb.RULE_VERDICT_STATUS_SATISFIED, testpilotspb.RULE_VERDICT_STATUS_VIOLATED), testpilotspb.VERDICT_STATUS_VIOLATED, true},
		{"violated beside pending", rules(testpilotspb.RULE_VERDICT_STATUS_PENDING, testpilotspb.RULE_VERDICT_STATUS_VIOLATED), testpilotspb.VERDICT_STATUS_VIOLATED, true},
		{"violated beside unspecified", rules(testpilotspb.RULE_VERDICT_STATUS_VIOLATED, testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED), testpilotspb.VERDICT_STATUS_VIOLATED, true},
		{"violated beside inconclusive", rules(testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE, testpilotspb.RULE_VERDICT_STATUS_VIOLATED), testpilotspb.VERDICT_STATUS_VIOLATED, true},
	}
)

func rules(statuses ...testpilotspb.RuleVerdictStatus) []testpilotspb.RuleVerdictStatus {
	return statuses
}

// Any violated rule makes the Verdict violated and the Run stopped by its Monitor, whatever the
// disposition; a completed Run whose rules are all satisfied, none at all included, is satisfied;
// everything else, a stopped Run without a violation among them, is inconclusive with its
// disposition kept.
func TestConclude(t *testing.T) {
	for _, mix := range verdictRuleMixes {
		for _, disposition := range terminalDispositions {
			t.Run(fmt.Sprintf("%s/%s", mix.name, disposition), func(t *testing.T) {
				wantStatus, wantDisposition := testpilotspb.VERDICT_STATUS_INCONCLUSIVE, disposition
				switch {
				case mix.violated:
					wantStatus, wantDisposition = testpilotspb.VERDICT_STATUS_VIOLATED, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
				case disposition == testpilotspb.RUN_DISPOSITION_COMPLETED:
					wantStatus = mix.onCompleted
				default:
				}
				status, concluded := Conclude(disposition, mix.rules)
				require.Equal(t, wantStatus, status)
				require.Equal(t, wantDisposition, concluded)
			})
		}
	}
}

// A Run's disposition is decided at close, strongest first: a violated Verdict stops the Run by its
// Monitor, even one found at closure or beside a failure; otherwise incompleteness makes it
// incomplete, even when the Monitor stopped it; otherwise the Monitor's stop or the completion
// stands, and a stopped Run without a violation is inconclusive. The cleanup outcome is no input.
func TestRunDispositionPrecedence(t *testing.T) {
	const (
		completed  = testpilotspb.RUN_DISPOSITION_COMPLETED
		stopped    = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
		incomplete = testpilotspb.RUN_DISPOSITION_INCOMPLETE
		satisfied  = testpilotspb.VERDICT_STATUS_SATISFIED
		violated   = testpilotspb.VERDICT_STATUS_VIOLATED
		unsettled  = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
		succeeded  = testpilotspb.CLEANUP_STATUS_SUCCEEDED
		failed     = testpilotspb.CLEANUP_STATUS_FAILED
	)
	for _, tc := range []struct {
		name                string
		disposition         testpilotspb.RunDisposition
		stopped, incomplete bool
		answer              testpilotspb.VerdictStatus
		cleanup             testpilotspb.CleanupStatus
		wantDisposition     testpilotspb.RunDisposition
		wantStatus          testpilotspb.VerdictStatus
	}{
		{"completed", completed, false, false, satisfied, succeeded, completed, satisfied},
		{"a failed cleanup leaves completion", completed, false, false, satisfied, failed, completed, satisfied},
		{"incompleteness over completion", completed, false, true, satisfied, succeeded, incomplete, unsettled},
		{"an incomplete disposition", incomplete, false, false, satisfied, succeeded, incomplete, unsettled},
		{"a violation found at closure", completed, false, false, violated, succeeded, stopped, violated},
		{"a violation over incompleteness and a failed cleanup", incomplete, true, true, violated, failed, stopped, violated},
		{"incompleteness over a stop without a violation", stopped, true, true, unsettled, succeeded, incomplete, unsettled},
		{"a stop without a violation", stopped, true, false, unsettled, succeeded, stopped, unsettled},
		{"a stopped Run is never satisfied", stopped, true, false, satisfied, succeeded, stopped, unsettled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := recorderFixture(t, &recorderMonitor{close: func(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, error) {
				return &testpilotspb.Verdict{Status: tc.answer}, nil
			}})
			r.stopped = tc.stopped
			r.incomplete = tc.incomplete
			run, verdict, err := r.close(context.Background(), tc.disposition, &testpilotspb.CleanupOutcome{Status: tc.cleanup})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, verdict.GetStatus())
			require.Equal(t, tc.wantDisposition, run.GetDisposition())
			require.Equal(t, tc.cleanup, run.GetCleanup().GetStatus())
		})
	}
}
