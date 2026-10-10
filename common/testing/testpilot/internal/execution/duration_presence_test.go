package execution

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDurationInstructionPresenceAndCeilings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		timeout  *durationpb.Duration
		attempts *int64
		accept   bool
	}{
		{"absent inherits", nil, nil, true},
		{"positive exact", pbduration.FromMilliseconds(1000), proto.Int64(1), true},
		{"explicit zero timeout", &durationpb.Duration{}, nil, false},
		{"explicit zero attempts", nil, proto.Int64(0), false},
		{"negative", pbduration.FromMilliseconds(-1), nil, false},
		{"submillisecond", &durationpb.Duration{Nanos: 1}, nil, false},
		{"nanosecond overflow", pbduration.FromMilliseconds(math.MaxInt64/int64(time.Millisecond) + 1), nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			policy.InstructionDefaults = contract.InstructionDefaults{TimeoutMilliseconds: 1000, MaxAttempts: 1}
			c.Program.Entrypoints[0].Instructions[0].Limits = &testpilotspb.InstructionLimits{Timeout: tc.timeout, MaxAttempts: tc.attempts}
			prepared, err := Prepare(c, catalog, policy)
			if !tc.accept {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, 1000, prepared.Entrypoints()[0].Instructions()[0].TimeoutMilliseconds())
		})
	}
}

func TestDurationProgramPresenceAndCeilings(t *testing.T) {
	for _, field := range []string{"max_duration", "cleanup_duration"} {
		for _, value := range []*durationpb.Duration{nil, {}, {Nanos: 1}, pbduration.FromMilliseconds(-1), pbduration.FromMilliseconds(86400001), pbduration.FromMilliseconds(math.MaxInt64/int64(time.Millisecond) + 1)} {
			c, catalog, policy := fixture(t)
			if field == "max_duration" {
				policy.Limits.MaxDuration = value
			} else {
				policy.Limits.CleanupDuration = value
			}
			_, err := Prepare(c, catalog, policy)
			require.Error(t, err, field)
		}
	}
}

func TestDurationElapsedStartsPresentAndNeverRewinds(t *testing.T) {
	r, now := recorderFixture(t, &recorderMonitor{})
	require.NotNil(t, r.run.Events[0].Elapsed)
	require.Zero(t, r.run.Events[0].Elapsed.AsDuration())
	*now = now.Add(5 * time.Millisecond)
	_, err := r.publish(t.Context(), []*testpilotspb.RunEvent{recorderFact("first")}, nil)
	require.NoError(t, err)
	*now = now.Add(-time.Second)
	_, err = r.publish(t.Context(), []*testpilotspb.RunEvent{recorderFact("second")}, nil)
	require.NoError(t, err)
	require.Equal(t, 5*time.Millisecond, r.run.Events[1].Elapsed.AsDuration())
	require.Equal(t, 5*time.Millisecond, r.run.Events[2].Elapsed.AsDuration())
}

func TestDurationEvidencePresenceAndCeilings(t *testing.T) {
	for _, hint := range []bool{false, true} {
		for _, value := range []*durationpb.Duration{nil, {}, {Nanos: 1}, pbduration.FromMilliseconds(-1), pbduration.FromMilliseconds(math.MaxInt64/int64(time.Millisecond) + 1)} {
			c, catalog, policy := hintedFixture(t)
			if hint {
				hintedPoll(c).WaitHints[0].AtMost = value
			} else {
				hintedPoll(c).WaitHints = nil
				hintedPoll(c).Instruction.GetReadEvidence().Interval = value
			}
			_, err := Prepare(c, catalog, policy)
			if !hint && value == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		}
	}
}
