package verification

import (
	"context"
	"strconv"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestRunElapsedNativeDurationReference(t *testing.T) {
	for _, milliseconds := range []int64{0, 2, 315576000000000} {
		t.Run(strconv.FormatInt(milliseconds, 10), func(t *testing.T) {
			c, catalog, view, limits := fixture(t)
			elapsed := ir.MillisecondsDuration(milliseconds)
			literal, err := anypb.New(elapsed)
			require.NoError(t, err)
			reference := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: testpilotspb.RUN_EVENT_FIELD_ELAPSED}}}})
			c.Rules[0].Transitions[0].Predicate = equal(reference, cel.Literal(&celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: literal}}))
			prepared, err := Prepare(c, catalog, view, limits, nil)
			require.NoError(t, err)
			evaluator, err := prepared.newEvaluator(t.Context(), view)
			require.NoError(t, err)
			_, err = evaluator.Observe(t.Context(), event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			_, err = evaluator.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: 2, Elapsed: elapsed, Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED})
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RULE_VERDICT_STATUS_SATISFIED, evaluator.result.Rules[0].Status)
			c.Rules[0].Transitions[0].Predicate = equal(reference, cel.Literal(&celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: milliseconds}}))
			_, err = Prepare(c, catalog, view, limits, nil)
			require.Error(t, err)
		})
	}
}

func TestContractSupportPresence(t *testing.T) {
	for _, support := range []*bool{nil, proto.Bool(false), proto.Bool(true)} {
		c, catalog, view, limits := fixture(t)
		c.Rules[0].Transitions[0].SupportsEvent = support
		_, err := Prepare(c, catalog, view, limits, nil)
		if support == nil {
			require.ErrorContains(t, err, "explicit presence")
		} else {
			require.NoError(t, err)
		}
	}
}

func TestContractElapsedDurationAdmission(t *testing.T) {
	for name, elapsed := range map[string]*durationpb.Duration{"absent": nil, "zero": {}, "negative": {Nanos: -1_000_000}, "submillisecond": {Nanos: 1}, "invalid-range": {Seconds: 315576000001}} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, limits := fixture(t)
			c.Rules[0].Deadline = &testpilotspb.Deadline{ViolationStateId: "bad", Bound: &testpilotspb.Deadline_Elapsed{Elapsed: elapsed}}
			_, err := Prepare(c, catalog, view, limits, nil)
			require.ErrorContains(t, err, ".deadline.elapsed")
		})
	}
}

func TestRunElapsedDurationAdmission(t *testing.T) {
	for name, elapsed := range map[string]*durationpb.Duration{"absent": nil, "zero": {}, "negative": {Nanos: -1_000_000}, "submillisecond": {Nanos: 1}, "invalid-range": {Seconds: 315576000001}} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, limits := fixture(t)
			prepared, err := Prepare(c, catalog, view, limits, nil)
			require.NoError(t, err)
			evaluator, err := prepared.newEvaluator(context.Background(), view)
			require.NoError(t, err)
			_, err = evaluator.Observe(context.Background(), &testpilotspb.RunEvent{Sequence: 1, Elapsed: elapsed, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED})
			if name == "absent" || name == "zero" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, "run_event.elapsed")
		})
	}
	t.Run("decreasing", func(t *testing.T) {
		c, catalog, view, limits := fixture(t)
		c.Rules[0].Transitions[0].Predicate = boolean(false)
		prepared, err := Prepare(c, catalog, view, limits, nil)
		require.NoError(t, err)
		evaluator, err := prepared.newEvaluator(context.Background(), view)
		require.NoError(t, err)
		for _, event := range []*testpilotspb.RunEvent{{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}, {Sequence: 2, Elapsed: ir.MillisecondsDuration(2), Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED}} {
			_, err := evaluator.Observe(context.Background(), event)
			require.NoError(t, err)
		}
		_, err = evaluator.Observe(context.Background(), &testpilotspb.RunEvent{Sequence: 3, Elapsed: ir.MillisecondsDuration(1), Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED})
		require.ErrorContains(t, err, "elapsed coordinate")
	})
}
