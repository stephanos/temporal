package verification

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
)

// The evaluator's Verdict is Conclude over its rule statuses, with an incomplete evaluation read as
// an incomplete Run whatever disposition closed it.
func TestEvaluatorVerdictConcludes(t *testing.T) {
	const (
		satisfied    = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
		violated     = testpilotspb.RULE_VERDICT_STATUS_VIOLATED
		inconclusive = testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
	)
	mixes := [][]testpilotspb.RuleVerdictStatus{
		nil,
		{satisfied},
		{satisfied, satisfied},
		{satisfied, inconclusive},
		{violated},
		{satisfied, violated},
		{inconclusive, violated},
	}
	dispositions := []testpilotspb.RunDisposition{
		testpilotspb.RUN_DISPOSITION_COMPLETED,
		testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
		testpilotspb.RUN_DISPOSITION_INCOMPLETE,
	}
	for _, mix := range mixes {
		for _, disposition := range dispositions {
			for _, incomplete := range []bool{false, true} {
				t.Run(fmt.Sprintf("%v/%s/incomplete=%t", mix, disposition, incomplete), func(t *testing.T) {
					e := &Evaluator{result: &testpilotspb.Verdict{}, incomplete: incomplete}
					for i, status := range mix {
						e.result.Rules = append(e.result.Rules, &testpilotspb.RuleVerdict{RuleId: fmt.Sprint(i), Status: status})
					}
					input := disposition
					if incomplete {
						input = testpilotspb.RUN_DISPOSITION_INCOMPLETE
					}
					want, _ := execution.Conclude(input, mix)
					require.Equal(t, want, e.verdict(disposition).GetStatus())
				})
			}
		}
	}
}
