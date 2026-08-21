package gomadv3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessFrameRoundTripAndStrictDecode(t *testing.T) {
	want := processFrame{Profile: processProtocol, Kind: processFrameStart, Request: 7, Node: "history", Incarnation: 2, Payload: []byte("bootstrap")}
	encoded, err := encodeProcessValue(want)
	require.NoError(t, err)
	var got processFrame
	require.NoError(t, decodeProcessValue(encoded, &got))
	require.Equal(t, want, got)
	require.NoError(t, validateProcessFrame(got))
	require.Error(t, decodeProcessValue([]byte(`{"kind":"start","request":7,"unknown":true}`), &got))
}

func TestProcessActivationFramesAreValid(t *testing.T) {
	for _, kind := range []processFrameKind{processFrameActivate, processFrameActivated} {
		require.NoError(t, validateProcessFrame(processFrame{Profile: processProtocol, Kind: kind, Request: 1, Node: "history", Incarnation: 2}))
	}
}

func TestProcessExplorationFramesAreValid(t *testing.T) {
	require.NoError(t, validateProcessFrame(processFrame{Profile: processProtocol, Kind: processFrameExplorationPlan, Request: 1}))
	require.NoError(t, validateProcessFrame(processFrame{Profile: processProtocol, Kind: processFrameExplorationRecord, Request: 2, Payload: []byte("record")}))
}
