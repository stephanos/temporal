package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad3/internal/model/cluster/internal/history"
)

func TestRuntimeVolumeTypedReplayError(t *testing.T) {
	transition := VolumeTransition{Ordinal: 3, Kind: VolumeOperationWrite, Dependencies: []uint64{}, SelectedOperations: []uint64{}, Bytes: 2, Outcome: VolumeOutcomeOK}
	bridge := history.RuntimeVolumeError{Kind: "replay", Message: "diverged", Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual", Expected: &transition, Actual: &transition}
	var divergence *ReplayDivergenceError
	err := runtimeVolumeErrorValue(bridge)
	require.ErrorAs(t, err, &divergence)
	require.ErrorIs(t, err, ErrReplayDiverged)
	require.Equal(t, ReplayDivergence{
		Dimension: ReplayDimensionVolume, Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual",
		ExpectedVolume: &transition, ActualVolume: &transition,
	}, divergence.Divergence)
}
