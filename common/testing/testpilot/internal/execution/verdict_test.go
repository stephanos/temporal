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

// The recorder concludes the Monitor's answer against the Run's disposition through Conclude:
// incompleteness turns anything short of a violation inconclusive, and a violation stops the Run.
func TestRecorderClosesThroughConclude(t *testing.T) {
	answers := map[testpilotspb.VerdictStatus]testpilotspb.RuleVerdictStatus{
		testpilotspb.VERDICT_STATUS_SATISFIED:    testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
		testpilotspb.VERDICT_STATUS_VIOLATED:     testpilotspb.RULE_VERDICT_STATUS_VIOLATED,
		testpilotspb.VERDICT_STATUS_INCONCLUSIVE: testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE,
	}
	for answer, rule := range answers {
		for _, disposition := range terminalDispositions[1:] {
			for _, stopped := range []bool{false, true} {
				for _, incomplete := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/stopped=%t/incomplete=%t", answer, disposition, stopped, incomplete), func(t *testing.T) {
						input := disposition
						if stopped {
							input = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
						}
						if incomplete || disposition == testpilotspb.RUN_DISPOSITION_INCOMPLETE {
							input = testpilotspb.RUN_DISPOSITION_INCOMPLETE
						}
						if input == testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR && answer == testpilotspb.VERDICT_STATUS_SATISFIED {
							t.Skip("a Monitor that stops a Run answers violated")
						}
						r, _ := recorderFixture(t, &recorderMonitor{close: func(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, error) {
							return &testpilotspb.Verdict{Status: answer}, nil
						}})
						r.stopped = stopped
						r.incomplete = incomplete
						run, verdict, err := r.close(context.Background(), disposition, &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED})
						require.NoError(t, err)
						wantStatus, wantDisposition := Conclude(input, []testpilotspb.RuleVerdictStatus{rule})
						require.Equal(t, wantStatus, verdict.GetStatus())
						require.Equal(t, wantDisposition, run.GetDisposition())
					})
				}
			}
		}
	}
}
