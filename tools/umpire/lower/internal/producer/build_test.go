package producer

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestWaitWithinRejectsInvalidBoundsWithoutPanicking(t *testing.T) {
	for _, test := range []struct {
		name, problem string
		interval      int64
		hints         []*testpilotspb.WaitHint
	}{
		{name: "fractional hint", interval: 1, hints: []*testpilotspb.WaitHint{{HintId: "visibility", AtMost: &durationpb.Duration{Nanos: 1}}}, problem: "hint visibility: wait hint at_most: Duration must contain exact whole milliseconds"},
		{name: "negative hint", interval: 1, hints: []*testpilotspb.WaitHint{{AtMost: &durationpb.Duration{Seconds: -1}}}, problem: "must be nonnegative"},
		{name: "combined protobuf range", interval: 1, hints: []*testpilotspb.WaitHint{{AtMost: &durationpb.Duration{Seconds: 315576000000}}, {AtMost: duration.FromMilliseconds(1000)}}, problem: "wait hint duration sum: invalid Duration"},
		{name: "zero interval", problem: "wait interval must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			apply, err := WaitWithin(test.interval, test.hints...)
			require.Nil(t, apply)
			require.ErrorContains(t, err, test.problem)
		})
	}
}

func TestWaitWithinPreservesExactFullRangeBounds(t *testing.T) {
	const milliseconds = int64(315576000000000)
	hints := []*testpilotspb.WaitHint{{HintId: "visibility", AtMost: duration.FromMilliseconds(milliseconds)}}
	apply, err := WaitWithin(1501, hints...)
	require.NoError(t, err)
	node := Node("read", ReadEvidence("evidence", "role", nil, nil, 0), apply)
	require.Equal(t, duration.FromMilliseconds(milliseconds), node.GetLimits().GetTimeout())
	require.Equal(t, duration.FromMilliseconds(1501), node.GetInstruction().GetReadEvidence().GetInterval())
	require.Equal(t, hints, node.GetWaitHints())
}

func TestEnumPreservesDescriptorAndWireNumber(t *testing.T) {
	value := Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED).GetEnumValue()
	require.Equal(t, "temporal.server.api.testpilot.v1.InstructionOutcomeStatus", value.GetType())
	require.Equal(t, int32(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED), value.GetValue())
}
