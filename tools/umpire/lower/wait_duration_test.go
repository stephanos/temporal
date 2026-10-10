package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDerivedWaitRejectsCombinedBoundAtOwningSource(t *testing.T) {
	w := &waiting{waits: map[string]derivedWait{}}
	w.recordWait(&reading{at: &umpirespb.Position{File: "Realization.scala", Line: 37}, name: "controller/poll", interval: 1,
		hints: []*testpilotspb.WaitHint{{AtMost: &durationpb.Duration{Seconds: 315576000000}}, {AtMost: &durationpb.Duration{Seconds: 1}}}})
	require.Empty(t, w.waits)
	require.Len(t, w.problems, 1)
	var located *interp.Error
	require.ErrorAs(t, w.problems[0], &located)
	require.Equal(t, "Realization.scala:37", located.Position)
	require.Contains(t, located.Message, "command controller/poll: wait hint duration sum: invalid Duration")
}

func TestAdmittedDurationMillisecondsPreservesFullRange(t *testing.T) {
	require.Zero(t, durationMilliseconds(nil))
	require.Equal(t, int64(1501), durationMilliseconds(&durationpb.Duration{Seconds: 1, Nanos: 501000000}))
	require.Equal(t, int64(315576000000000), durationMilliseconds(&durationpb.Duration{Seconds: 315576000000}))
}
