package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"google.golang.org/protobuf/proto"
)

func TestEvaluatorFailureViolationOrderingAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name               string
		failure, violation int
		want               testpilotpb.VerdictStatus
		stop               int64
	}{
		{"failure before violation", 1, 2, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, 0},
		{"failure on violating event", 1, 1, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, 0},
		{"violation before failure", 2, 1, testpilotpb.VERDICT_STATUS_VIOLATED, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			rule := c.Rules[0]
			addCapture(rule)
			rule.Transitions[0].TargetStateId = "bad"
			rule.Transitions[0].Predicate = present(observation("id"))
			assign(rule.Transitions[0])
			prepared, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)
			events := []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), event(2, 100, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC), event(3, 200, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC), event(4, 300, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED)}
			events[tc.violation] = observed(int64(tc.violation+1), int64(tc.violation)*100, 7)
			events[tc.failure].ExecutionIncomplete = true
			run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Events: events, Status: testpilotpb.RUN_STATUS_INCOMPLETE}
			if tc.stop > 0 {
				run.Status = testpilotpb.RUN_STATUS_STOPPED_BY_MONITOR
			}
			live, err := prepared.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			var stop int64
			for _, fact := range events {
				decision, err := live.Observe(context.Background(), fact)
				require.NoError(t, err)
				if decision == execution.Stop && stop == 0 {
					stop = fact.Sequence
				}
			}
			require.Equal(t, tc.stop, stop)
			verdict, err := live.Close(context.Background(), run)
			require.NoError(t, err)
			require.Equal(t, tc.want, verdict.Status)
			offline, replayed, err := prepared.evaluate(context.Background(), run)
			require.NoError(t, err)
			require.True(t, proto.Equal(verdict, replayed))
			require.Equal(t, live.trace, offline.trace)
			if tc.stop == 0 {
				require.Empty(t, verdict.SupportingEventSequences)
				require.Empty(t, live.trace)
				require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_INCONCLUSIVE, verdict.Rules[0].Status)
			} else {
				require.Equal(t, []int64{2}, verdict.SupportingEventSequences)
				require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_VIOLATED, verdict.Rules[0].Status)
			}
		})
	}
}
