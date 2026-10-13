// Code generated from simulation/schema/processwire.json by gomadtool protocol-generate; DO NOT EDIT.

package gomad3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedProcessWaitVector(t *testing.T) {
	want := processFrame{Profile: processProtocol, Kind: processFrameWait, Request: 7}
	encoded, err := encodeProcessValue(want)
	require.NoError(t, err)
	require.Equal(t, "{\"profile\":\"gomad3.simulation-process/v3\",\"kind\":\"wait\",\"request\":7}", string(encoded))
	var got processFrame
	require.NoError(t, decodeProcessValue(encoded, &got))
	require.Equal(t, want, got)
	require.NoError(t, validateProcessFrame(got))
}

func TestGeneratedProcessFrameKinds(t *testing.T) {
	for _, kind := range []processFrameKind{
		processFrameStart, processFrameActivate, processFrameActivated, processFrameModel, processFrameStop, processFrameCrash, processFrameWait, processFrameReady, processFrameTerminal, processFrameResponse, processFrameExplorationPlan, processFrameExplorationRecord,
	} {
		t.Run(string(kind), func(t *testing.T) {
			want := processFrame{Profile: processProtocol, Kind: kind, Request: 1}
			encoded, err := encodeProcessValue(want)
			require.NoError(t, err)
			var got processFrame
			require.NoError(t, decodeProcessValue(encoded, &got))
			require.Equal(t, want, got)
			require.NoError(t, validateProcessFrame(got))
		})
	}
}
