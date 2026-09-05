package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"google.golang.org/protobuf/proto"
)

func TestEvaluatorFailureViolationOrderingAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name               string
		failure, violation int
		want               umpirespb.VerdictKind
		stop               int64
	}{
		{"failure before violation", 1, 2, umpirespb.VERDICT_KIND_INCONCLUSIVE, 0},
		{"failure on violating event", 1, 1, umpirespb.VERDICT_KIND_INCONCLUSIVE, 0},
		{"violation before failure", 2, 1, umpirespb.VERDICT_KIND_VIOLATED, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			rule := c.Rules[0]
			addCapture(rule)
			rule.Transitions[0].TargetState = "bad"
			rule.Transitions[0].Predicate = present(observation("id"))
			assign(rule.Transitions[0])
			prepared, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)
			events := []*umpirespb.RunEvent{event(1, 0, umpirespb.RUN_EVENT_KIND_RUN_OPENED), event(2, 100, umpirespb.RUN_EVENT_KIND_DIAGNOSTIC), event(3, 200, umpirespb.RUN_EVENT_KIND_DIAGNOSTIC), event(4, 300, umpirespb.RUN_EVENT_KIND_RUN_CLOSED)}
			events[tc.violation] = observed(int64(tc.violation+1), int64(tc.violation)*100, 7)
			events[tc.failure].ExecutionIncomplete = true
			run := &umpirespb.Run{RunId: "run", ProgramId: "program", Events: events, Disposition: umpirespb.RUN_DISPOSITION_INCOMPLETE}
			if tc.stop > 0 {
				run.Disposition = umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR
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
			require.Equal(t, tc.want, verdict.Kind)
			offline, replayed, err := prepared.evaluate(context.Background(), run)
			require.NoError(t, err)
			require.True(t, proto.Equal(verdict, replayed))
			require.Equal(t, live.trace, offline.trace)
			if tc.stop == 0 {
				require.Empty(t, verdict.SupportingEventSequences)
				require.Empty(t, live.trace)
				require.Equal(t, umpirespb.RULE_VERDICT_KIND_INCONCLUSIVE, verdict.Rules[0].Kind)
			} else {
				require.Equal(t, []int64{2}, verdict.SupportingEventSequences)
				require.Equal(t, umpirespb.RULE_VERDICT_KIND_VIOLATED, verdict.Rules[0].Kind)
			}
		})
	}
}
