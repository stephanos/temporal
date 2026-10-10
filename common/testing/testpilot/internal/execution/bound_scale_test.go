package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// A scale below 100% only makes a wait fail sooner, so preparation refuses it, naming the scale.
func TestPrepareRefusesABoundScaleBelowOneHundred(t *testing.T) {
	for _, scale := range []contract.BoundScale{-100, -1, 1, 99} {
		c, catalog, p := fixture(t)
		p.BoundScale = scale
		_, err := Prepare(c, catalog, p)
		var diagnostic *ir.Error
		require.ErrorAs(t, err, &diagnostic)
		require.Equal(t, ir.Malformed, diagnostic.Category)
		require.Equal(t, "policy.bound_scale", diagnostic.Path)
		require.Equal(t, fmt.Sprintf("bound scale %d%% is below 100%%", scale), diagnostic.Detail)
	}
}

// A scaled Profile's duration ceilings must fit the Program ceiling once scaled, and the refusal
// names the scale.
func TestPrepareRefusesScaledCeilingsAboveTheProgramCeiling(t *testing.T) {
	c, catalog, p := fixture(t)
	p.Limits.MaxDuration = pbduration.FromMilliseconds(ProgramCeiling().MaxDuration.AsDuration().Milliseconds() / 2)
	p.BoundScale = 200
	_, err := Prepare(c, catalog, p)
	require.NoError(t, err)

	p.BoundScale = 201
	_, err = Prepare(c, catalog, p)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.Error{Category: ir.LimitExceeded, Path: "max_duration", Detail: "limit scaled by 201% is outside the positive Driver ceiling"}, *diagnostic)
}

// The prepared ceilings are the scaled ones, and the Run records the scale only when it scales:
// an unscaled Run, at zero or 100%, keeps the bytes it had before scales existed.
func TestScaledPreparationScalesCeilingsAndRecordsTheScale(t *testing.T) {
	for _, tc := range []struct {
		scale         contract.BoundScale
		total, clean  int64
		recordedScale int64
	}{
		{scale: 0, total: 30000, clean: 5000},
		{scale: 100, total: 30000, clean: 5000},
		{scale: 250, total: 75000, clean: 12500, recordedScale: 250},
	} {
		c, catalog, p := fixture(t)
		p.BoundScale = tc.scale
		prepared, err := Prepare(c, catalog, p)
		require.NoError(t, err)
		want := proto.CloneOf(p.Limits)
		want.MaxDuration, want.CleanupDuration = pbduration.FromMilliseconds(tc.total), pbduration.FromMilliseconds(tc.clean)
		require.True(t, proto.Equal(want, prepared.Limits()), "scale %d", tc.scale)
		require.Equal(t, int64(30000), p.Limits.MaxDuration.AsDuration().Milliseconds(), "the Profile's limits are not scaled in place")

		r, err := newRecorder(prepared.View(), "run", "case", &recorderMonitor{}, func() time.Time { return time.Unix(100, 0) }, nil, nil)
		require.NoError(t, err)
		_, err = r.publish(context.Background(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "open"}}, nil)
		require.NoError(t, err)
		run := closeRecorder(t, r)
		require.Equal(t, tc.recordedScale, run.GetBoundScalePercent(), "scale %d", tc.scale)
	}
}
