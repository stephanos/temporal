package contract

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestEntrypointKindOfClassifiesEachActivation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		entrypoint *testpilotspb.Entrypoint
		want       EntrypointKind
	}{
		{name: "controller", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}}, want: ControllerEntrypoint},
		{name: "workflow", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{}}}, want: WorkflowEntrypoint},
		{name: "activity", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{}}}, want: ActivityEntrypoint},
		{name: "nexus handler", entrypoint: &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{}}}, want: NexusHandlerEntrypoint},
		{name: "no activation", entrypoint: &testpilotspb.Entrypoint{}},
		{name: "nil entrypoint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, EntrypointKindOf(tc.entrypoint))
		})
	}
	require.Equal(t, NexusHandlerEntrypoint, MaxEntrypointKind)
}

// A bound scale of zero is 100%; a scaled bound rounds up, so it is never shorter than declared,
// and saturates rather than wrapping.
func TestBoundScaleAppliesRoundingUpAndSaturating(t *testing.T) {
	t.Parallel()
	require.Equal(t, int64(100), BoundScale(0).Percent())
	require.False(t, BoundScale(0).Scaled())
	require.False(t, BoundScale(100).Scaled())
	require.Equal(t, int64(250), BoundScale(250).Percent())
	require.True(t, BoundScale(250).Scaled())

	require.Equal(t, int64(1234), BoundScale(0).Apply(1234))
	require.Equal(t, int64(1234), BoundScale(100).Apply(1234))
	require.Equal(t, int64(3085), BoundScale(250).Apply(1234))
	require.Equal(t, int64(2), BoundScale(101).Apply(1), "101% of 1 ms rounds up")
	require.Equal(t, int64(0), BoundScale(250).Apply(0))
	require.Equal(t, int64(math.MaxInt64), BoundScale(200).Apply(math.MaxInt64/2+1))
}

// Ceilings scales only the total and cleanup duration ceilings, in a copy, and hands back the
// limits themselves when the scale changes nothing.
func TestBoundScaleCeilingsScaleOnlyTheDurations(t *testing.T) {
	t.Parallel()
	limits := &testpilotspb.ProgramLimits{MaxAttempts: 32, MaxRunEvents: 256, MaxDuration: durationpb.New(time.Duration(30000) * time.Millisecond), CleanupDuration: durationpb.New(time.Duration(5000) * time.Millisecond)}
	require.Same(t, limits, BoundScale(0).Ceilings(limits))
	require.Same(t, limits, BoundScale(100).Ceilings(limits))
	require.Nil(t, BoundScale(200).Ceilings(nil))

	scaled := BoundScale(150).Ceilings(limits)
	require.NotSame(t, limits, scaled)
	require.True(t, proto.Equal(&testpilotspb.ProgramLimits{MaxAttempts: 32, MaxRunEvents: 256, MaxDuration: durationpb.New(time.Duration(45000) * time.Millisecond), CleanupDuration: durationpb.New(time.Duration(7500) * time.Millisecond)}, scaled))
	require.Equal(t, int64(30000), limits.MaxDuration.AsDuration().Milliseconds())
	require.Equal(t, int64(5000), limits.CleanupDuration.AsDuration().Milliseconds())
}
