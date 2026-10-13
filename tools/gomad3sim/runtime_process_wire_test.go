package gomad3sim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessFrameRejectsMalformedValuesAndInvalidSemantics(t *testing.T) {
	for _, encoded := range []string{
		"", "{", `{"request":-1}`, `{"request":1} {}`, `{"request":1} trailing`, `{"request":1,"unknown":true}`,
	} {
		var frame processFrame
		require.Error(t, decodeProcessValue([]byte(encoded), &frame), "value %q", encoded)
	}
	for _, frame := range []processFrame{
		{Profile: "unsupported", Kind: processFrameStart, Request: 1},
		{Profile: processProtocol, Kind: "unknown", Request: 1},
		{Profile: processProtocol, Kind: processFrameStart},
		{Profile: processProtocol, Kind: processFrameResponse, Request: 1, Arrivals: 1},
		{Profile: processProtocol, Kind: processFrameStart, Request: 1, Node: strings.Repeat("n", 257)},
		{Profile: processProtocol, Kind: processFrameTerminal, Request: 1, Error: strings.Repeat("e", MaximumTerminalReasonBytes+1)},
	} {
		require.Error(t, validateProcessFrame(frame))
	}
}

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
